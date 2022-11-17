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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/multiparty"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
)

const broadcastDispatcherName = "pinned_broadcast"

type Manager interface {
	core.Named

	NewBroadcast(in *core.MessageInOut) syncasync.Sender
	BroadcastMessage(ctx context.Context, in *core.MessageInOut, waitConfirm bool) (out *core.Message, err error)
	PublishDataValue(ctx context.Context, id string, idempotencyKey core.IdempotencyKey) (*core.Data, error)
	PublishDataBlob(ctx context.Context, id string, idempotencyKey core.IdempotencyKey) (*core.Data, error)
	Start() error
	WaitStop()

	// From operations.OperationHandler
	PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error)
	RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error)
}

type broadcastManager struct {
	ctx                   context.Context
	namespace             *core.Namespace
	database              database.Plugin
	identity              identity.Manager
	data                  data.Manager
	blockchain            blockchain.Plugin
	exchange              dataexchange.Plugin
	sharedstorage         sharedstorage.Plugin
	syncasync             syncasync.Bridge
	multiparty            multiparty.Manager
	maxBatchPayloadLength int64
	metrics               metrics.Manager
	operations            operations.Manager
	txHelper              txcommon.Helper
}

func NewBroadcastManager(ctx context.Context, ns *core.Namespace, di database.Plugin, bi blockchain.Plugin, dx dataexchange.Plugin, si sharedstorage.Plugin, im identity.Manager, dm data.Manager, ba batch.Manager, sa syncasync.Bridge, mult multiparty.Manager, mm metrics.Manager, om operations.Manager, txHelper txcommon.Helper) (Manager, error) {
	if di == nil || im == nil || dm == nil || bi == nil || dx == nil || si == nil || mm == nil || om == nil || txHelper == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "BroadcastManager")
	}
	bm := &broadcastManager{
		ctx:                   ctx,
		namespace:             ns,
		database:              di,
		identity:              im,
		data:                  dm,
		blockchain:            bi,
		exchange:              dx,
		sharedstorage:         si,
		syncasync:             sa,
		multiparty:            mult,
		maxBatchPayloadLength: config.GetByteSize(coreconfig.BroadcastBatchPayloadLimit),
		metrics:               mm,
		operations:            om,
		txHelper:              txHelper,
	}

	if ba != nil && mult != nil {
		bo := batch.DispatcherOptions{
			BatchType:      core.BatchTypeBroadcast,
			BatchMaxSize:   config.GetUint(coreconfig.BroadcastBatchSize),
			BatchMaxBytes:  bm.maxBatchPayloadLength,
			BatchTimeout:   config.GetDuration(coreconfig.BroadcastBatchTimeout),
			DisposeTimeout: config.GetDuration(coreconfig.BroadcastBatchAgentTimeout),
		}

		ba.RegisterDispatcher(broadcastDispatcherName,
			core.TransactionTypeBatchPin,
			[]core.MessageType{
				core.MessageTypeBroadcast,
				core.MessageTypeDefinition,
				core.MessageTypeTransferBroadcast,
			}, bm.dispatchBatch, bo)
	}

	om.RegisterHandler(ctx, bm, []core.OpType{
		core.OpTypeSharedStorageUploadBatch,
		core.OpTypeSharedStorageUploadBlob,
		core.OpTypeSharedStorageUploadValue,
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
	op := core.NewOperation(
		bm.sharedstorage,
		bm.namespace.Name,
		state.Persisted.TX.ID,
		core.OpTypeSharedStorageUploadBatch)
	addUploadBatchInputs(op, state.Persisted.ID)
	if err := bm.operations.AddOrReuseOperation(ctx, op); err != nil {
		return err
	}
	batch := state.Persisted.GenInflight(state.Messages, state.Data)

	// We are in an (indefinite) retry cycle from the batch processor to dispatch this batch, that is only
	// terminated with shutdown. So we leave the operation pending on failure, as it is still being retried.
	// The user will still have the failure details recorded.
	outputs, err := bm.operations.RunOperation(ctx, opUploadBatch(op, batch), operations.RemainPendingOnFailure)
	if err != nil {
		return err
	}
	payloadRef := outputs.GetString("payloadRef")
	log.L(ctx).Infof("Pinning broadcast batch %s with author=%s key=%s payloadRef=%s", batch.ID, batch.Author, batch.Key, payloadRef)
	return bm.multiparty.SubmitBatchPin(ctx, &state.Persisted, state.Pins, payloadRef)
}

func (bm *broadcastManager) uploadBlobs(ctx context.Context, tx *fftypes.UUID, data core.DataArray) error {
	for _, d := range data {
		// We only need to send a blob if there is one, and it's not been uploaded to the shared storage
		if d.Blob != nil && d.Blob.Hash != nil && d.Blob.Public == "" {
			if err := bm.uploadDataBlob(ctx, tx, d); err != nil {
				return err
			}
		}
	}

	return nil
}

func (bm *broadcastManager) resolveData(ctx context.Context, id string) (*core.Data, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}

	d, err := bm.database.GetDataByID(ctx, bm.namespace.Name, u, true)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}

	return d, nil
}

func (bm *broadcastManager) uploadDataBlob(ctx context.Context, tx *fftypes.UUID, d *core.Data) error {
	if d.Blob == nil || d.Blob.Hash == nil {
		return i18n.NewError(ctx, coremsgs.MsgDataDoesNotHaveBlob)
	}

	op := core.NewOperation(
		bm.sharedstorage,
		bm.namespace.Name,
		tx,
		core.OpTypeSharedStorageUploadBlob)
	addUploadBlobInputs(op, d.ID)
	if err := bm.operations.AddOrReuseOperation(ctx, op); err != nil {
		return err
	}

	blob, err := bm.database.GetBlobMatchingHash(ctx, d.Blob.Hash)
	if err != nil {
		return err
	} else if blob == nil {
		return i18n.NewError(ctx, coremsgs.MsgBlobNotFound, d.Blob.Hash)
	}

	_, err = bm.operations.RunOperation(ctx, opUploadBlob(op, d, blob))
	return err
}

func (bm *broadcastManager) PublishDataValue(ctx context.Context, id string, idempotencyKey core.IdempotencyKey) (*core.Data, error) {

	d, err := bm.resolveData(ctx, id)
	if err != nil {
		return nil, err
	}

	txid, err := bm.txHelper.SubmitNewTransaction(ctx, core.TransactionTypeDataPublish, idempotencyKey)
	if err != nil {
		return nil, err
	}

	op := core.NewOperation(
		bm.sharedstorage,
		bm.namespace.Name,
		txid,
		core.OpTypeSharedStorageUploadValue)
	addUploadValueInputs(op, d.ID)
	if err := bm.operations.AddOrReuseOperation(ctx, op); err != nil {
		return nil, err
	}

	if _, err := bm.operations.RunOperation(ctx, opUploadValue(op, d)); err != nil {
		return nil, err
	}

	return d, nil
}

func (bm *broadcastManager) PublishDataBlob(ctx context.Context, id string, idempotencyKey core.IdempotencyKey) (*core.Data, error) {

	d, err := bm.resolveData(ctx, id)
	if err != nil {
		return nil, err
	}

	txid, err := bm.txHelper.SubmitNewTransaction(ctx, core.TransactionTypeDataPublish, idempotencyKey)
	if err != nil {
		return nil, err
	}

	if err = bm.uploadDataBlob(ctx, txid, d); err != nil {
		return nil, err
	}

	return d, nil
}

func (bm *broadcastManager) Start() error {
	return nil
}

func (bm *broadcastManager) WaitStop() {
	// No go routines
}
