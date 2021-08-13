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

	"github.com/hyperledger-labs/firefly/internal/batch"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/data"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/syncasync"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/dataexchange"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/identity"
	"github.com/hyperledger-labs/firefly/pkg/publicstorage"
)

type Manager interface {
	BroadcastDatatype(ctx context.Context, ns string, datatype *fftypes.Datatype, waitConfirm bool) (msg *fftypes.Message, err error)
	BroadcastNamespace(ctx context.Context, ns *fftypes.Namespace, waitConfirm bool) (msg *fftypes.Message, err error)
	BroadcastMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error)
	BroadcastMessageWithID(ctx context.Context, ns string, unresolved *fftypes.MessageInOut, resolved *fftypes.Message, waitConfirm bool) (out *fftypes.Message, err error)
	BroadcastDefinition(ctx context.Context, def fftypes.Definition, signingIdentity *fftypes.Identity, tag fftypes.SystemTag, waitConfirm bool) (msg *fftypes.Message, err error)
	GetNodeSigningIdentity(ctx context.Context) (*fftypes.Identity, error)
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
	syncasync     syncasync.Bridge
}

func NewBroadcastManager(ctx context.Context, di database.Plugin, ii identity.Plugin, dm data.Manager, bi blockchain.Plugin, dx dataexchange.Plugin, pi publicstorage.Plugin, ba batch.Manager, sa syncasync.Bridge) (Manager, error) {
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
		syncasync:     sa,
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
	orgIdentity := config.GetString(config.OrgIdentity)
	id, err := bm.identity.Resolve(ctx, orgIdentity)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func SubmitPinnedBatch(ctx context.Context, bi blockchain.Plugin, id identity.Plugin, db database.Plugin, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {

	signingIdentity, err := id.Resolve(ctx, batch.Author)
	if err == nil {
		err = bi.VerifyIdentitySyntax(ctx, signingIdentity)
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
			Signer:    signingIdentity.OnChain, // The transaction records on the on-chain identity
			Reference: batch.ID,
		},
		Created: fftypes.Now(),
		Status:  fftypes.OpStatusPending,
	}
	tx.Hash = tx.Subject.Hash()
	err = db.UpsertTransaction(ctx, tx, true, false /* should be new, or idempotent replay */)
	if err != nil {
		return err
	}

	// Write the batch pin to the blockchain
	err = bi.SubmitBatchPin(ctx, nil, signingIdentity, &blockchain.BatchPin{
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
		bi,
		batch.Namespace,
		batch.Payload.TX.ID,
		"",
		fftypes.OpTypeBlockchainBatchPin,
		fftypes.OpStatusPending,
		"")
	return db.UpsertOperation(ctx, op, false)
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
		"")
	err = bm.database.UpsertOperation(ctx, op, false)
	if err != nil {
		return err
	}

	return SubmitPinnedBatch(ctx, bm.blockchain, bm.identity, bm.database, batch, contexts)
}

func (bm *broadcastManager) broadcastMessageCommon(ctx context.Context, msg *fftypes.Message, waitConfirm bool) (*fftypes.Message, error) {

	if !waitConfirm {
		// Seal the message
		if err := msg.Seal(ctx); err != nil {
			return nil, err
		}

		// Store the message - this asynchronously triggers the next step in process
		return msg, bm.database.InsertMessageLocal(ctx, msg)
	}

	return bm.syncasync.SendConfirm(ctx, msg)
}

func (bm *broadcastManager) Start() error {
	return nil
}

func (bm *broadcastManager) WaitStop() {
	// No go routines
}
