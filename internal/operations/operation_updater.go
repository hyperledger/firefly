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

package operations

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/config"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
	"github.com/hyperledger/firefly/pkg/log"
)

// OperationUpdate is dispatched asynchronously to perform an update.
type OperationUpdate struct {
	ID             *fftypes.UUID
	Status         fftypes.OpStatus
	BlockchainTXID string
	ErrorMessage   string
	Output         fftypes.JSONObject
	VerifyManifest bool
	DXManifest     string
	DXHash         string
	OnComplete     func()
}

type operationUpdaterBatch struct {
	updates        []*OperationUpdate
	timeoutContext context.Context
	timeoutCancel  func()
}

// operationUpdater
type operationUpdater struct {
	ctx         context.Context
	cancelFunc  func()
	database    database.Plugin
	txHelper    txcommon.Helper
	workQueues  []chan *OperationUpdate
	workersDone []chan struct{}
	conf        operationUpdaterConf
	closed      bool
	retry       *retry.Retry
}

type operationUpdaterConf struct {
	workerCount  int
	batchTimeout time.Duration
	maxInserts   int
	queueLength  int
}

func newOperationUpdater(ctx context.Context, di database.Plugin, txHelper txcommon.Helper) *operationUpdater {
	ou := &operationUpdater{
		database: di,
		txHelper: txHelper,
		conf: operationUpdaterConf{
			workerCount:  config.GetInt(coreconfig.OpUpdateWorkerCount),
			batchTimeout: config.GetDuration(coreconfig.OpUpdateWorkerBatchTimeout),
			maxInserts:   config.GetInt(coreconfig.OpUpdateWorkerBatchMaxInserts),
			queueLength:  config.GetInt(coreconfig.OpUpdateWorkerQueueLength),
		},
		retry: &retry.Retry{
			InitialDelay: config.GetDuration(coreconfig.OpUpdateRetryInitDelay),
			MaximumDelay: config.GetDuration(coreconfig.OpUpdateRetryMaxDelay),
			Factor:       config.GetFloat64(coreconfig.OpUpdateRetryFactor),
		},
	}
	ou.ctx, ou.cancelFunc = context.WithCancel(ctx)
	if !di.Capabilities().Concurrency {
		log.L(ctx).Infof("Database plugin not configured for concurrency. Batched operation updates disabled")
		ou.conf.workerCount = 0
	}
	return ou
}

// pickWorker ensures multiple updates for the same ID go to the same worker
func (ou *operationUpdater) pickWorker(ctx context.Context, update *OperationUpdate) chan *OperationUpdate {
	worker := update.ID.HashBucket(ou.conf.workerCount)
	log.L(ctx).Debugf("Submitting operation update id=%s status=%s blockchainTX=%s worker=opu_%.3d", update.ID, update.Status, update.BlockchainTXID, worker)
	return ou.workQueues[worker]
}

func (ou *operationUpdater) SubmitOperationUpdate(ctx context.Context, update *OperationUpdate) {
	if ou.conf.workerCount > 0 {
		select {
		case ou.pickWorker(ctx, update) <- update:
		case <-ou.ctx.Done():
			log.L(ctx).Debugf("Not submitting operation update due to cancelled context")
		}
		return
	}
	// Otherwise do it in-line on this context
	err := ou.doBatchUpdateWithRetry(ctx, []*OperationUpdate{update})
	if err != nil {
		log.L(ctx).Warnf("Exiting while updating operation: %s", err)
	}
}

func (ou *operationUpdater) initQueues() {
	ou.workQueues = make([]chan *OperationUpdate, ou.conf.workerCount)
	ou.workersDone = make([]chan struct{}, ou.conf.workerCount)
	for i := 0; i < ou.conf.workerCount; i++ {
		ou.workQueues[i] = make(chan *OperationUpdate, ou.conf.queueLength)
		ou.workersDone[i] = make(chan struct{})
	}
}

func (ou *operationUpdater) start() {
	if ou.conf.workerCount > 0 {
		ou.initQueues()
		for i := 0; i < ou.conf.workerCount; i++ {
			go ou.updaterLoop(i)
		}
	}
}

func (ou *operationUpdater) updaterLoop(index int) {
	defer close(ou.workersDone[index])
	workQueue := ou.workQueues[index]

	ctx := log.WithLogField(ou.ctx, "opupdater", fmt.Sprintf("opu_%.3d", index))

	var batch *operationUpdaterBatch
	for !ou.closed {
		var timeoutContext context.Context
		var timedOut bool
		if batch != nil {
			timeoutContext = batch.timeoutContext
		} else {
			timeoutContext = ctx
		}
		select {
		case work := <-workQueue:
			if batch == nil {
				batch = &operationUpdaterBatch{}
				batch.timeoutContext, batch.timeoutCancel = context.WithTimeout(ctx, ou.conf.batchTimeout)
			}
			batch.updates = append(batch.updates, work)
		case <-timeoutContext.Done():
			timedOut = true
		}

		if batch != nil && (timedOut || len(batch.updates) >= ou.conf.maxInserts) {
			batch.timeoutCancel()
			err := ou.doBatchUpdateWithRetry(ctx, batch.updates)
			if err != nil {
				log.L(ctx).Debugf("Operation update worker exiting: %s", err)
				return
			}
			batch = nil
		}
	}
}

func (ou *operationUpdater) doBatchUpdateWithRetry(ctx context.Context, updates []*OperationUpdate) error {
	return ou.retry.Do(ctx, "operation update", func(attempt int) (retry bool, err error) {
		err = ou.database.RunAsGroup(ctx, func(ctx context.Context) error {
			return ou.doBatchUpdate(ctx, updates)
		})
		if err != nil {
			return true, err
		}
		for _, update := range updates {
			if update.OnComplete != nil {
				update.OnComplete()
			}
		}
		return false, nil
	})
}

func (ou *operationUpdater) doBatchUpdate(ctx context.Context, updates []*OperationUpdate) error {

	// Get all the operations that match
	opIDs := make([]driver.Value, len(updates))
	for idx, update := range updates {
		opIDs[idx] = update.ID
	}
	opFilter := database.OperationQueryFactory.NewFilter(ctx).In("id", opIDs)
	ops, _, err := ou.database.GetOperations(ctx, opFilter)
	if err != nil {
		return err
	}

	// Get all the transactions for these operations
	txIDs := make([]driver.Value, 0, len(ops))
	for _, op := range ops {
		if op.Transaction != nil {
			txIDs = append(txIDs, op.Transaction)
		}
	}
	var transactions []*fftypes.Transaction
	if len(txIDs) > 0 {
		txFilter := database.TransactionQueryFactory.NewFilter(ctx).In("id", txIDs)
		transactions, _, err = ou.database.GetTransactions(ctx, txFilter)
		if err != nil {
			return err
		}
	}

	// Spin through each update seeing what DB updates we need to do
	for _, update := range updates {
		if err := ou.doUpdate(ctx, update, ops, transactions); err != nil {
			return err
		}
	}

	return nil
}

func (ou *operationUpdater) doUpdate(ctx context.Context, update *OperationUpdate, ops []*fftypes.Operation, transactions []*fftypes.Transaction) error {

	// Find the operation we already retrieved, and do the update
	var op *fftypes.Operation
	for _, candidate := range ops {
		if update.ID.Equals(candidate.ID) {
			op = candidate
			break
		}
	}
	if op == nil {
		log.L(ctx).Warnf("Operation update '%s' ignored, as it was not submitted by this node", update.ID)
		return nil
	}

	// Match a TX we already retireved, if found add a specified Blockchain Transaction ID to it
	var tx *fftypes.Transaction
	if op.Transaction != nil && update.BlockchainTXID != "" {
		for _, candidate := range transactions {
			if op.Transaction.Equals(candidate.ID) {
				tx = candidate
				break
			}
		}
	}
	if tx != nil {
		if err := ou.txHelper.AddBlockchainTX(ctx, tx, update.BlockchainTXID); err != nil {
			return err
		}
	}

	// Special handling for OpTypeTokenTransfer, which writes an event when it fails
	if op.Type == fftypes.OpTypeTokenTransfer && update.Status == fftypes.OpStatusFailed {
		tokenTransfer, err := txcommon.RetrieveTokenTransferInputs(ctx, op)
		topic := ""
		if tokenTransfer != nil {
			topic = tokenTransfer.Pool.String()
		}
		event := fftypes.NewEvent(fftypes.EventTypeTransferOpFailed, op.Namespace, op.ID, op.Transaction, topic)
		if err != nil || tokenTransfer.LocalID == nil || tokenTransfer.Type == "" {
			log.L(ctx).Warnf("Could not parse token transfer: %s (%+v)", err, op.Input)
		} else {
			event.Correlator = tokenTransfer.LocalID
		}
		if err := ou.database.InsertEvent(ctx, event); err != nil {
			return err
		}
	}

	// Special handling for OpTypeTokenApproval, which writes an event when it fails
	if op.Type == fftypes.OpTypeTokenApproval && update.Status == fftypes.OpStatusFailed {
		tokenApproval, err := txcommon.RetrieveTokenApprovalInputs(ctx, op)
		topic := ""
		if tokenApproval != nil {
			topic = tokenApproval.Pool.String()
		}
		event := fftypes.NewEvent(fftypes.EventTypeApprovalOpFailed, op.Namespace, op.ID, op.Transaction, topic)
		if err != nil || tokenApproval.LocalID == nil {
			log.L(ctx).Warnf("Could not parse token approval: %s (%+v)", err, op.Input)
		} else {
			event.Correlator = tokenApproval.LocalID
		}
		if err := ou.database.InsertEvent(ctx, event); err != nil {
			return err
		}
	}

	// Special handling for data exchange manifests
	if update.VerifyManifest {
		if err := ou.verifyManifest(ctx, update, op); err != nil {
			return err
		}
	}

	if err := ou.database.ResolveOperation(ctx, op.ID, update.Status, update.ErrorMessage, update.Output); err != nil {
		return err
	}

	return nil
}

func (ou *operationUpdater) verifyManifest(ctx context.Context, update *OperationUpdate, op *fftypes.Operation) error {

	if op.Type == fftypes.OpTypeDataExchangeSendBatch && update.Status == fftypes.OpStatusSucceeded {
		batchID, _ := fftypes.ParseUUID(ctx, op.Input.GetString("batch"))
		expectedManifest := ""
		if batchID != nil {
			batch, err := ou.database.GetBatchByID(ctx, batchID)
			if err != nil {
				return err
			}
			if batch != nil {
				expectedManifest = batch.Manifest.String()
			}
		}
		if update.DXManifest != expectedManifest {
			// Log and map to failure for user to see that the receiver did not provide a matching acknowledgement
			mismatchErr := i18n.NewError(ctx, coremsgs.MsgManifestMismatch, fftypes.OpStatusSucceeded, update.DXManifest)
			log.L(ctx).Errorf("DX transfer %s: %s", op.ID, mismatchErr.Error())
			update.ErrorMessage = mismatchErr.Error()
			update.Status = fftypes.OpStatusFailed
		}
	}

	if op.Type == fftypes.OpTypeDataExchangeSendBlob && update.Status == fftypes.OpStatusSucceeded {
		expectedHash := op.Input.GetString("hash")
		if update.DXHash != expectedHash {
			// Log and map to failure for user to see that the receiver did not provide a matching hash
			mismatchErr := i18n.NewError(ctx, coremsgs.MsgBlobHashMismatch, expectedHash, update.DXHash)
			log.L(ctx).Errorf("DX transfer %s: %s", op.ID, mismatchErr.Error())
			update.ErrorMessage = mismatchErr.Error()
			update.Status = fftypes.OpStatusFailed
		}
	}

	return nil
}

func (ou *operationUpdater) close() {
	if !ou.closed {
		ou.closed = true
		ou.cancelFunc()
		for _, workerDone := range ou.workersDone {
			<-workerDone
		}
	}
}
