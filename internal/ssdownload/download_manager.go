// Copyright © 2022 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ssdownload

import (
	"context"
	"database/sql/driver"
	"math"
	"time"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
)

type Manager interface {
	Start() error
	WaitStop()

	InitiateDownloadBatch(ctx context.Context, ns string, tx *fftypes.UUID, payloadRef string) error
	InitiateDownloadBlob(ctx context.Context, ns string, tx *fftypes.UUID, dataID *fftypes.UUID, payloadRef string) error
}

// downloadManager operates a number of workers that can perform downloads/retries. Each download
// will stay in pending state until a number of retries has been executed against, but each retry
// will be dispatched individually to the workers. So a retrying downloads do not block new
// downloads from getting a chance to use the workers.
// Pending download operations are recovered on startup, and start a new retry loop.
type downloadManager struct {
	ctx                        context.Context
	cancelFunc                 func()
	database                   database.Plugin
	sharedstorage              sharedstorage.Plugin
	dataexchange               dataexchange.Plugin
	operations                 operations.Manager
	callbacks                  Callbacks
	workerCount                int
	workers                    []*downloadWorker
	work                       chan *downloadWork
	recoveryComplete           chan struct{}
	broadcastBatchPayloadLimit int64
	retryMaxAttempts           int
	retryInitDelay             time.Duration
	retryMaxDelay              time.Duration
	retryFactor                float64
}

type downloadWork struct {
	op         *fftypes.Operation
	preparedOp *fftypes.PreparedOperation
	attempts   int
}

type Callbacks interface {
	SharedStorageBatchDownloaded(ns string, payloadRef string, data []byte) (batchID *fftypes.UUID, err error)
	SharedStorageBLOBDownloaded(hash fftypes.Bytes32, size int64, payloadRef string) error
}

func NewDownloadManager(ctx context.Context, di database.Plugin, ss sharedstorage.Plugin, dx dataexchange.Plugin, om operations.Manager, cb Callbacks) (Manager, error) {
	if di == nil || dx == nil || ss == nil || cb == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}

	dmCtx, cancelFunc := context.WithCancel(ctx)
	dm := &downloadManager{
		ctx:                        dmCtx,
		cancelFunc:                 cancelFunc,
		database:                   di,
		sharedstorage:              ss,
		dataexchange:               dx,
		operations:                 om,
		callbacks:                  cb,
		broadcastBatchPayloadLimit: config.GetByteSize(config.BroadcastBatchPayloadLimit),
		workerCount:                config.GetInt(config.DownloadWorkerCount),
		retryMaxAttempts:           config.GetInt(config.DownloadRetryMaxAttempts),
		retryInitDelay:             config.GetDuration(config.DownloadRetryInitDelay),
		retryMaxDelay:              config.GetDuration(config.DownloadRetryMaxDelay),
		retryFactor:                config.GetFloat64(config.DownloadRetryFactor),
	}
	// Work queue is twice
	workQueueLength := config.GetInt(config.DownloadWorkerQueueLength)
	if workQueueLength <= 0 {
		workQueueLength = 2 * dm.workerCount
	}
	if dm.retryMaxAttempts <= 0 {
		dm.retryMaxAttempts = 1
	}
	dm.work = make(chan *downloadWork, workQueueLength)

	dm.operations.RegisterHandler(ctx, dm, []fftypes.OpType{
		fftypes.OpTypeSharedStorageDownloadBatch,
		fftypes.OpTypeSharedStorageDownloadBlob,
	})

	return dm, nil
}

func (dm *downloadManager) Start() error {
	dm.workers = make([]*downloadWorker, dm.workerCount)
	for i := 0; i < dm.workerCount; i++ {
		dm.workers[i] = newDownloadWorker(dm, i)
	}
	dm.recoveryComplete = make(chan struct{})
	go dm.recoverDownloads(fftypes.Now())
	return nil
}

func (dm *downloadManager) Name() string {
	return "SharedStorageDownloadManager"
}

func (dm *downloadManager) WaitStop() {
	dm.cancelFunc()
	for _, w := range dm.workers {
		<-w.done
	}
}

func (dm *downloadManager) calcDelay(attempts int) time.Duration {
	delay := dm.retryInitDelay
	for i := 0; i < attempts; i++ {
		delay = time.Duration(math.Ceil(float64(delay) * dm.retryFactor))
	}
	if delay > dm.retryMaxDelay {
		delay = dm.retryMaxDelay
	}
	return delay
}

// recoverDownloads grabs all pending operations on startup, to restart them
func (dm *downloadManager) recoverDownloads(startupTime *fftypes.FFTime) {

	defer close(dm.recoveryComplete)
	recovered := 0
	pageSize := uint64(25)
	page := uint64(0)
	errorAttempts := 0
	for {
		fb := database.OperationQueryFactory.NewFilter(dm.ctx)
		filter := fb.And(
			fb.In("type", []driver.Value{
				fftypes.OpTypeSharedStorageDownloadBatch,
				fftypes.OpTypeSharedStorageDownloadBlob,
			}),
			fb.Eq("status", fftypes.OpStatusPending),
			fb.Lt("created", startupTime), // retry is handled completely separately
		).
			Sort("created").
			Skip(page * pageSize).
			Limit(pageSize)
		pendingOps, _, err := dm.database.GetOperations(dm.ctx, filter)
		if err != nil {
			log.L(dm.ctx).Errorf("Error while recovering pending downloads (retries=%d): %s", errorAttempts, err)
			errorAttempts++
			time.Sleep(dm.calcDelay(errorAttempts))
			continue
		}
		errorAttempts = 0 // reset on success
		page++
		if len(pendingOps) == 0 {
			log.L(dm.ctx).Infof("Download manager completed startup after recovering %d pending downloads", recovered)
			return
		}
		for _, op := range pendingOps {
			preparedOp, err := dm.PrepareOperation(dm.ctx, op)
			if err != nil {
				log.L(dm.ctx).Errorf("Failed to recover pending download %s/%s: %s", op.Type, op.ID, err)
				continue
			}
			log.L(dm.ctx).Infof("Recovering pending download %s/%s", op.Type, op.ID)
			dm.dispatchWork(&downloadWork{
				op:         op,
				preparedOp: preparedOp,
			})
		}
	}

}

func (dm *downloadManager) dispatchWork(work *downloadWork) {
	dm.work <- work
	// Log after dispatching so we can see the dispatch delay if the queue got full
	log.L(dm.ctx).Debugf("Dispatched download operation %s/%s (attempts=%d) to worker pool", work.preparedOp.Type, work.preparedOp.ID, work.attempts)
}

// waitAndRetryDownload is a go routine to wait and re-dispatch a retrying download.
// Note this go routine is short lived and completely separate to the workers.
func (dm *downloadManager) waitAndRetryDownload(work *downloadWork) {
	startedWaiting := time.Now()
	delay := dm.calcDelay(work.attempts)
	<-time.After(delay)
	delayTimeMS := time.Since(startedWaiting).Milliseconds()
	totalTimeMS := time.Since(*work.op.Created.Time()).Milliseconds()
	log.L(dm.ctx).Infof("Retrying download operation %s/%s after %dms (total=%dms,attempts=%d)",
		work.preparedOp.Type, work.preparedOp.ID, delayTimeMS, totalTimeMS, work.attempts)
	dm.dispatchWork(work)
}

func (dm *downloadManager) InitiateDownloadBatch(ctx context.Context, ns string, tx *fftypes.UUID, payloadRef string) error {
	op := fftypes.NewOperation(dm.sharedstorage, ns, tx, fftypes.OpTypeSharedStorageDownloadBatch)
	addDownloadBatchInputs(op, payloadRef)
	return dm.createAndDispatchOp(ctx, op, opDownloadBatch(op, ns, payloadRef))
}

func (dm *downloadManager) InitiateDownloadBlob(ctx context.Context, ns string, tx *fftypes.UUID, dataID *fftypes.UUID, payloadRef string) error {
	op := fftypes.NewOperation(dm.sharedstorage, ns, tx, fftypes.OpTypeSharedStorageDownloadBlob)
	addDownloadBlobInputs(op, payloadRef)
	return dm.createAndDispatchOp(ctx, op, opDownloadBlob(op, ns, dataID, payloadRef))
}

func (dm *downloadManager) createAndDispatchOp(ctx context.Context, op *fftypes.Operation, preparedOp *fftypes.PreparedOperation) error {
	err := dm.database.InsertOperation(ctx, op, func() {
		// Use a closure hook to dispatch the work once the operation is successfully in the DB.
		// Note we have crash recovery of pending operations on startup.
		dm.dispatchWork(&downloadWork{
			op:         op,
			preparedOp: preparedOp,
		})
	})
	if err != nil {
		return err
	}
	return nil
}
