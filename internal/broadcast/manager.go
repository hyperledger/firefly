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
	"bytes"
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/batchpin"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/publicstorage"
)

type Manager interface {
	NewBroadcast(ns string, in *fftypes.MessageInOut) sysmessaging.MessageSender
	BroadcastDatatype(ctx context.Context, ns string, datatype *fftypes.Datatype, waitConfirm bool) (msg *fftypes.Message, err error)
	BroadcastNamespace(ctx context.Context, ns *fftypes.Namespace, waitConfirm bool) (msg *fftypes.Message, err error)
	BroadcastMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error)
	BroadcastDefinitionAsNode(ctx context.Context, ns string, def fftypes.Definition, tag fftypes.SystemTag, waitConfirm bool) (msg *fftypes.Message, err error)
	BroadcastDefinition(ctx context.Context, ns string, def fftypes.Definition, signingIdentity *fftypes.Identity, tag fftypes.SystemTag, waitConfirm bool) (msg *fftypes.Message, err error)
	BroadcastRootOrgDefinition(ctx context.Context, def *fftypes.Organization, signingIdentity *fftypes.Identity, tag fftypes.SystemTag, waitConfirm bool) (msg *fftypes.Message, err error)
	BroadcastTokenPool(ctx context.Context, ns string, pool *fftypes.TokenPoolAnnouncement, waitConfirm bool) (msg *fftypes.Message, err error)
	Start() error
	WaitStop()
}

type broadcastManager struct {
	ctx                   context.Context
	database              database.Plugin
	identity              identity.Manager
	data                  data.Manager
	blockchain            blockchain.Plugin
	exchange              dataexchange.Plugin
	publicstorage         publicstorage.Plugin
	batch                 batch.Manager
	syncasync             syncasync.Bridge
	batchpin              batchpin.Submitter
	maxBatchPayloadLength int64
}

func NewBroadcastManager(ctx context.Context, di database.Plugin, im identity.Manager, dm data.Manager, bi blockchain.Plugin, dx dataexchange.Plugin, pi publicstorage.Plugin, ba batch.Manager, sa syncasync.Bridge, bp batchpin.Submitter) (Manager, error) {
	if di == nil || im == nil || dm == nil || bi == nil || dx == nil || pi == nil || ba == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	bm := &broadcastManager{
		ctx:                   ctx,
		database:              di,
		identity:              im,
		data:                  dm,
		blockchain:            bi,
		exchange:              dx,
		publicstorage:         pi,
		batch:                 ba,
		syncasync:             sa,
		batchpin:              bp,
		maxBatchPayloadLength: config.GetByteSize(config.BroadcastBatchPayloadLimit),
	}
	bo := batch.Options{
		BatchMaxSize:   config.GetUint(config.BroadcastBatchSize),
		BatchMaxBytes:  bm.maxBatchPayloadLength,
		BatchTimeout:   config.GetDuration(config.BroadcastBatchTimeout),
		DisposeTimeout: config.GetDuration(config.BroadcastBatchAgentTimeout),
	}
	ba.RegisterDispatcher([]fftypes.MessageType{
		fftypes.MessageTypeBroadcast,
		fftypes.MessageTypeDefinition,
		fftypes.MessageTypeTransferBroadcast,
	}, bm.dispatchBatch, bo)
	return bm, nil
}

func (bm *broadcastManager) dispatchBatch(ctx context.Context, batch *fftypes.Batch, pins []*fftypes.Bytes32) error {

	// Serialize the full payload, which has already been sealed for us by the BatchManager
	payload, err := json.Marshal(batch)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Write it to IPFS to get a payload reference
	// The payload ref will be persisted back to the batch, as well as being used in the TX
	batch.PayloadRef, err = bm.publicstorage.PublishData(ctx, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	return bm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return bm.submitTXAndUpdateDB(ctx, batch, pins)
	})
}

func (bm *broadcastManager) submitTXAndUpdateDB(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {

	// Update the batch to store the payloadRef
	err := bm.database.UpdateBatch(ctx, batch.ID, database.BatchQueryFactory.NewUpdate(ctx).Set("payloadref", batch.PayloadRef))
	if err != nil {
		return err
	}

	// The completed PublicStorage upload
	op := fftypes.NewTXOperation(
		bm.publicstorage,
		batch.Namespace,
		batch.Payload.TX.ID,
		batch.PayloadRef,
		fftypes.OpTypePublicStorageBatchBroadcast,
		fftypes.OpStatusSucceeded, // Note we performed the action synchronously above
	)
	err = bm.database.InsertOperation(ctx, op)
	if err != nil {
		return err
	}

	return bm.batchpin.SubmitPinnedBatch(ctx, batch, contexts)
}

func (bm *broadcastManager) publishBlobs(ctx context.Context, dataToPublish []*fftypes.DataAndBlob) error {
	for _, d := range dataToPublish {
		// Stream from the local data exchange ...
		reader, err := bm.exchange.DownloadBLOB(ctx, d.Blob.PayloadRef)
		if err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgDownloadBlobFailed, d.Blob.PayloadRef)
		}
		defer reader.Close()

		// ... to the public storage
		publicRef, err := bm.publicstorage.PublishData(ctx, reader)
		if err != nil {
			return err
		}
		log.L(ctx).Infof("Published blob with hash '%s' for data '%s' to public storage: '%s'", d.Blob.Hash, d.Data.ID, publicRef)

		// Update the data in the database, with the public reference.
		// We do this independently for each piece of data
		update := database.DataQueryFactory.NewUpdate(ctx).Set("blob.public", publicRef)
		err = bm.database.UpdateData(ctx, d.Data.ID, update)
		if err != nil {
			return err
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
