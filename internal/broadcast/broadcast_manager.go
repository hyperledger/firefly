// Copyright © 2021 Kaleido, Inc.
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
	"time"

	"github.com/kaleido-io/firefly/internal/batching"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/p2pfs"
)

type BroadcastManager interface {
	BroadcastMessage(ctx context.Context, msg *fftypes.Message) error
	Close()
}

type broadcastManager struct {
	ctx        context.Context
	database   database.Plugin
	blockchain blockchain.Plugin
	p2pfs      p2pfs.Plugin
	batch      batching.BatchManager
}

func NewBroadcastManager(ctx context.Context, database database.Plugin, blockchain blockchain.Plugin, p2pfs p2pfs.Plugin, batch batching.BatchManager) (BroadcastManager, error) {
	if database == nil || blockchain == nil || batch == nil || p2pfs == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	b := &broadcastManager{
		ctx:        ctx,
		database:   database,
		blockchain: blockchain,
		p2pfs:      p2pfs,
		batch:      batch,
	}
	bo := batching.BatchOptions{
		BatchMaxSize:   config.GetUint(config.BroadcastBatchSize),
		BatchTimeout:   time.Duration(config.GetUint(config.BroadcastBatchTimeout)) * time.Millisecond,
		DisposeTimeout: time.Duration(config.GetUint(config.BroadcastBatchAgentTimeout)) * time.Millisecond,
	}
	batch.RegisterDispatcher(fftypes.MessageTypeBroadcast, b.dispatchBatch, bo)
	batch.RegisterDispatcher(fftypes.MessageTypeDefinition, b.dispatchBatch, bo)
	return b, nil
}

func (b *broadcastManager) dispatchBatch(ctx context.Context, batch *fftypes.Batch) error {

	// Serialize the full payload, which has already been sealed for us by the BatchManager
	payload, err := json.Marshal(batch)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Write it to IPFS to get a payload reference hash (might not be the sha256 data hash).
	// The payload ref will be persisted back to the batch, as well as being used in the TX
	var p2pfsID string
	batch.PayloadRef, p2pfsID, err = b.p2pfs.PublishData(ctx, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	return b.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return b.submitTXAndUpdateDB(ctx, batch, batch.PayloadRef, p2pfsID)
	})
}

func (b *broadcastManager) submitTXAndUpdateDB(ctx context.Context, batch *fftypes.Batch, payloadRef *fftypes.Bytes32, p2pfsID string) error {
	// Write the transation to our DB, to collect transaction submission updates
	tx := &fftypes.Transaction{
		ID: batch.Payload.TX.ID,
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypePin,
			Namespace: batch.Namespace,
			Author:    batch.Author,
			Batch:     batch.ID,
		},
		Created: fftypes.Now(),
		Status:  fftypes.TransactionStatusPending,
	}
	tx.Hash = tx.Subject.Hash()
	err := b.database.UpsertTransaction(ctx, tx, false /* should be new, or idempotent replay */)
	if err != nil {
		return err
	}

	// Update the batch to store the payloadRef
	err = b.database.UpdateBatch(ctx, batch.ID, database.BatchQueryFactory.NewUpdate(ctx).Set("payloadref", payloadRef))
	if err != nil {
		return err
	}

	// Write the batch pin to the blockchain
	blockchainTrackingID, err := b.blockchain.SubmitBroadcastBatch(ctx, batch.Author, &blockchain.BroadcastBatch{
		TransactionID:  batch.Payload.TX.ID,
		BatchID:        batch.ID,
		BatchPaylodRef: batch.PayloadRef,
	})
	if err != nil {
		return err
	}

	// Store the operations for each message
	for _, msg := range batch.Payload.Messages {

		// The pending blockchain transaction
		op := fftypes.NewMessageOp(
			b.blockchain,
			blockchainTrackingID,
			msg,
			fftypes.OpTypeBlockchainBatchPin,
			fftypes.OpDirectionOutbound,
			fftypes.OpStatusPending,
			"")
		if err := b.database.UpsertOperation(ctx, op); err != nil {
			return err
		}

		// The completed P2PFS upload
		op = fftypes.NewMessageOp(
			b.p2pfs,
			p2pfsID,
			msg,
			fftypes.OpTypeP2PFSBatchBroadcast,
			fftypes.OpDirectionOutbound,
			fftypes.OpStatusSucceeded, // Note we performed the action synchronously above
			"")
		if err := b.database.UpsertOperation(ctx, op); err != nil {
			return err
		}
	}

	return nil
}

func (b *broadcastManager) BroadcastMessage(ctx context.Context, msg *fftypes.Message) (err error) {

	// Seal the message
	if err = msg.Seal(ctx); err != nil {
		return err
	}

	// Store the message - this asynchronously triggers the next step in process
	return b.database.UpsertMessage(ctx, msg, false /* should be new, or idempotent replay */)
}

func (b *broadcastManager) Close() {}
