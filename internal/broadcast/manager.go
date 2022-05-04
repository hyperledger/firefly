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

package broadcast

import (
	"context"

	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/batchpin"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/config"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
	"github.com/hyperledger/firefly/pkg/log"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
)

const broadcastDispatcherName = "pinned_broadcast"

type Manager interface {
	fftypes.Named

	NewBroadcast(ns string, in *fftypes.MessageInOut) sysmessaging.MessageSender
	BroadcastMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error)
	Start() error
	WaitStop()

	// From operations.OperationHandler
	PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error)
	RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error)
}

type broadcastManager struct {
	ctx                   context.Context
	database              database.Plugin
	identity              identity.Manager
	data                  data.Manager
	blockchain            blockchain.Plugin
	exchange              dataexchange.Plugin
	sharedstorage         sharedstorage.Plugin
	batch                 batch.Manager
	syncasync             syncasync.Bridge
	batchpin              batchpin.Submitter
	maxBatchPayloadLength int64
	metrics               metrics.Manager
	operations            operations.Manager
}

func NewBroadcastManager(ctx context.Context, di database.Plugin, im identity.Manager, dm data.Manager, bi blockchain.Plugin, dx dataexchange.Plugin, si sharedstorage.Plugin, ba batch.Manager, sa syncasync.Bridge, bp batchpin.Submitter, mm metrics.Manager, om operations.Manager) (Manager, error) {
	if di == nil || im == nil || dm == nil || bi == nil || dx == nil || si == nil || ba == nil || mm == nil || om == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError)
	}
	bm := &broadcastManager{
		ctx:                   ctx,
		database:              di,
		identity:              im,
		data:                  dm,
		blockchain:            bi,
		exchange:              dx,
		sharedstorage:         si,
		batch:                 ba,
		syncasync:             sa,
		batchpin:              bp,
		maxBatchPayloadLength: config.GetByteSize(coreconfig.BroadcastBatchPayloadLimit),
		metrics:               mm,
		operations:            om,
	}

	bo := batch.DispatcherOptions{
		BatchType:      fftypes.BatchTypeBroadcast,
		BatchMaxSize:   config.GetUint(coreconfig.BroadcastBatchSize),
		BatchMaxBytes:  bm.maxBatchPayloadLength,
		BatchTimeout:   config.GetDuration(coreconfig.BroadcastBatchTimeout),
		DisposeTimeout: config.GetDuration(coreconfig.BroadcastBatchAgentTimeout),
	}

	ba.RegisterDispatcher(broadcastDispatcherName,
		fftypes.TransactionTypeBatchPin,
		[]fftypes.MessageType{
			fftypes.MessageTypeBroadcast,
			fftypes.MessageTypeDefinition,
			fftypes.MessageTypeTransferBroadcast,
		}, bm.dispatchBatch, bo)

	om.RegisterHandler(ctx, bm, []fftypes.OpType{
		fftypes.OpTypeSharedStorageUploadBatch,
		fftypes.OpTypeSharedStorageUploadBlob,
	})

	return bm, nil
}

func (bm *broadcastManager) Name() string {
	return "BroadcastManager"
}

func (bm *broadcastManager) dispatchBatch(ctx context.Context, state *batch.DispatchState) error {

	// Ensure all the blobs are published
	if err := bm.uploadBlobs(ctx, state.Persisted.TX.ID, state.Data); err != nil {
		return err
	}

	// Upload the batch itself
	op := fftypes.NewOperation(
		bm.sharedstorage,
		state.Persisted.Namespace,
		state.Persisted.TX.ID,
		fftypes.OpTypeSharedStorageUploadBatch)
	addUploadBatchInputs(op, state.Persisted.ID)
	if err := bm.operations.AddOrReuseOperation(ctx, op); err != nil {
		return err
	}
	batch := state.Persisted.GenInflight(state.Messages, state.Data)

	// We are in an (indefinite) retry cycle from the batch processor to dispatch this batch, that is only
	// termianted with shutdown. So we leave the operation pending on failure, as it is still being retried.
	// The user will still have the failure details recorded.
	outputs, err := bm.operations.RunOperation(ctx, opUploadBatch(op, batch, &state.Persisted), operations.RemainPendingOnFailure)
	if err != nil {
		return err
	}
	payloadRef := outputs.GetString("payloadRef")
	log.L(ctx).Infof("Pinning broadcast batch %s with author=%s key=%s payloadRef=%s", batch.ID, batch.Author, batch.Key, payloadRef)
	return bm.batchpin.SubmitPinnedBatch(ctx, &state.Persisted, state.Pins, payloadRef)
}

func (bm *broadcastManager) uploadBlobs(ctx context.Context, tx *fftypes.UUID, data fftypes.DataArray) error {
	for _, d := range data {
		// We only need to send a blob if there is one, and it's not been uploaded to the shared storage
		if d.Blob != nil && d.Blob.Hash != nil && d.Blob.Public == "" {

			op := fftypes.NewOperation(
				bm.sharedstorage,
				d.Namespace,
				tx,
				fftypes.OpTypeSharedStorageUploadBlob)
			addUploadBlobInputs(op, d.ID)

			blob, err := bm.database.GetBlobMatchingHash(ctx, d.Blob.Hash)
			if err != nil {
				return err
			} else if blob == nil {
				return i18n.NewError(ctx, coremsgs.MsgBlobNotFound, d.Blob.Hash)
			}

			_, err = bm.operations.RunOperation(ctx, opUploadBlob(op, d, blob))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (bm *broadcastManager) Start() error {
	return nil
}

func (bm *broadcastManager) WaitStop() {
	// No go routines
}
