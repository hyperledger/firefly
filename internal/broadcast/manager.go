// Copyright © 2021 Kaleido, Inc.
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

	"github.com/kaleido-io/firefly/internal/batch"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/data"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/dataexchange"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/kaleido-io/firefly/pkg/identity"
	"github.com/kaleido-io/firefly/pkg/publicstorage"
)

type Manager interface {
	BroadcastDatatype(ctx context.Context, ns string, datatype *fftypes.Datatype) (msg *fftypes.Message, err error)
	BroadcastNamespace(ctx context.Context, ns *fftypes.Namespace) (msg *fftypes.Message, err error)
	BroadcastDefinition(ctx context.Context, def fftypes.Definition, signingIdentity *fftypes.Identity, tag fftypes.SystemTag) (msg *fftypes.Message, err error)
	BroadcastMessage(ctx context.Context, ns string, in *fftypes.MessageInput) (out *fftypes.Message, err error)
	GetNodeSigningIdentity(ctx context.Context) (*fftypes.Identity, error)
	HandleSystemBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error)
	Start() error
	WaitStop()
}

type broadcastManager struct {
	ctx           context.Context
	database      database.Plugin
	identity      identity.Plugin
	data          data.Manager
	blockchain    blockchain.Plugin
	exchange      dataexchange.Plugin
	publicstorage publicstorage.Plugin
	batch         batch.Manager
}

func NewBroadcastManager(ctx context.Context, di database.Plugin, ii identity.Plugin, dm data.Manager, bi blockchain.Plugin, dx dataexchange.Plugin, pi publicstorage.Plugin, ba batch.Manager) (Manager, error) {
	if di == nil || ii == nil || dm == nil || bi == nil || dx == nil || pi == nil || ba == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	bm := &broadcastManager{
		ctx:           ctx,
		database:      di,
		identity:      ii,
		data:          dm,
		blockchain:    bi,
		exchange:      dx,
		publicstorage: pi,
		batch:         ba,
	}
	bo := batch.Options{
		BatchMaxSize:   config.GetUint(config.BroadcastBatchSize),
		BatchTimeout:   config.GetDuration(config.BroadcastBatchTimeout),
		DisposeTimeout: config.GetDuration(config.BroadcastBatchAgentTimeout),
	}
	ba.RegisterDispatcher([]fftypes.MessageType{
		fftypes.MessageTypeBroadcast,
		fftypes.MessageTypeDefinition,
	}, bm.dispatchBatch, bo)
	return bm, nil
}

func (bm *broadcastManager) GetNodeSigningIdentity(ctx context.Context) (*fftypes.Identity, error) {
	nodeIdentity := config.GetString(config.NodeIdentity)
	id, err := bm.identity.Resolve(ctx, nodeIdentity)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func (bm *broadcastManager) dispatchBatch(ctx context.Context, batch *fftypes.Batch, pins []*fftypes.Bytes32) error {

	// Serialize the full payload, which has already been sealed for us by the BatchManager
	payload, err := json.Marshal(batch)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Write it to IPFS to get a payload reference hash (might not be the sha256 data hash).
	// The payload ref will be persisted back to the batch, as well as being used in the TX
	var publicstorageID string
	batch.PayloadRef, publicstorageID, err = bm.publicstorage.PublishData(ctx, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	return bm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return bm.submitTXAndUpdateDB(ctx, batch, pins, publicstorageID)
	})
}

func (bm *broadcastManager) submitTXAndUpdateDB(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32, publicstorageID string) error {

	id, err := bm.identity.Resolve(ctx, batch.Author)
	if err == nil {
		err = bm.blockchain.VerifyIdentitySyntax(ctx, id)
	}
	if err != nil {
		log.L(ctx).Errorf("Invalid signing identity '%s': %s", batch.Author, err)
		return err
	}

	tx := &fftypes.Transaction{
		ID: batch.Payload.TX.ID,
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Namespace: batch.Namespace,
			Signer:    id.OnChain, // The transaction records on the on-chain identity
			Reference: batch.ID,
		},
		Created: fftypes.Now(),
		Status:  fftypes.OpStatusPending,
	}
	tx.Hash = tx.Subject.Hash()
	err = bm.database.UpsertTransaction(ctx, tx, true, false /* should be new, or idempotent replay */)
	if err != nil {
		return err
	}

	// Update the batch to store the payloadRef
	err = bm.database.UpdateBatch(ctx, batch.ID, database.BatchQueryFactory.NewUpdate(ctx).Set("payloadref", batch.PayloadRef))
	if err != nil {
		return err
	}

	// Write the batch pin to the blockchain
	blockchainTrackingID, err := bm.blockchain.SubmitBatchPin(ctx, nil, id, &blockchain.BatchPin{
		Namespace:      batch.Namespace,
		TransactionID:  batch.Payload.TX.ID,
		BatchID:        batch.ID,
		BatchHash:      batch.Hash,
		BatchPaylodRef: batch.PayloadRef,
		Contexts:       contexts,
	})
	if err != nil {
		return err
	}

	// The pending blockchain transaction
	op := fftypes.NewTXOperation(
		bm.blockchain,
		batch.Payload.TX.ID,
		blockchainTrackingID,
		fftypes.OpTypeBlockchainBatchPin,
		fftypes.OpStatusPending,
		"")
	if err := bm.database.UpsertOperation(ctx, op, false); err != nil {
		return err
	}

	// The completed PublicStorage upload
	op = fftypes.NewTXOperation(
		bm.publicstorage,
		batch.Payload.TX.ID,
		publicstorageID,
		fftypes.OpTypePublicStorageBatchBroadcast,
		fftypes.OpStatusSucceeded, // Note we performed the action synchronously above
		"")
	return bm.database.UpsertOperation(ctx, op, false)
}

func (bm *broadcastManager) broadcastMessageCommon(ctx context.Context, msg *fftypes.Message) (err error) {

	// Seal the message
	if err = msg.Seal(ctx); err != nil {
		return err
	}

	// Store the message - this asynchronously triggers the next step in process
	return bm.database.InsertMessageLocal(ctx, msg)
}

func (bm *broadcastManager) Start() error {
	return nil
}

func (bm *broadcastManager) WaitStop() {
	// No go routines
}
