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
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/p2pfs"
	"github.com/kaleido-io/firefly/internal/database"
)

type Broadcast interface {
	BroadcastMessage(ctx context.Context, identity string, msg *fftypes.Message) error
	Close()
}

type broadcast struct {
	ctx         context.Context
	database database.Plugin
	blockchain  blockchain.Plugin
	p2pfs       p2pfs.Plugin
	batch       batching.BatchManager
}

func NewBroadcast(ctx context.Context, database database.Plugin, blockchain blockchain.Plugin, p2pfs p2pfs.Plugin, batch batching.BatchManager) (Broadcast, error) {
	if database == nil || blockchain == nil || batch == nil || p2pfs == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	b := &broadcast{
		ctx:         ctx,
		database: database,
		blockchain:  blockchain,
		p2pfs:       p2pfs,
		batch:       batch,
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

func (b *broadcast) dispatchBatch(ctx context.Context, batch *fftypes.Batch, updates database.Update) error {

	// In a retry scenario we don't need to re-write the batch itself to IPFS
	if batch.PayloadRef == nil {
		// Serialize the full payload, which has already been sealed for us by the BatchManager
		payload, err := json.Marshal(batch)
		if err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
		}

		// Write it to IPFS to get a payload reference hash (might not be the sha256 data hash)
		// Note the BatchManager is responsible for writing the updated batch
		batch.PayloadRef, err = b.p2pfs.PublishData(ctx, bytes.NewReader(payload))
		if err != nil {
			return err
		}
	}

	// Write it to the blockchain
	txid := fftypes.NewUUID()
	updates.Set("tx.id", txid)
	updates.Set("tx.type", string(fftypes.TransactionTypePin))
	trackingId, err := b.blockchain.SubmitBroadcastBatch(ctx, batch.Author, &blockchain.BroadcastBatch{
		Timestamp:      batch.Created,
		BatchID:        batch.ID,
		BatchPaylodRef: batch.PayloadRef,
	})
	if err != nil {
		return err
	}

	// Write the transation to our DB, to collect transaction submission updates
	tx := &fftypes.Transaction{
		ID:         txid,
		Type:       fftypes.TransactionTypePin,
		Namespace:  batch.Namespace,
		Author:     batch.Author,
		Created:    fftypes.NowMillis(),
		TrackingID: trackingId,
	}
	err = b.database.UpsertTransaction(ctx, tx)
	if err != nil {
		return err
	}

	return nil
}

func (b *broadcast) BroadcastMessage(ctx context.Context, identity string, msg *fftypes.Message) (err error) {

	// Seal the message
	if err = msg.Seal(ctx); err != nil {
		return err
	}

	// Store the message - this asynchronously triggers the next step in process
	return b.database.UpsertMessage(ctx, msg)
}

func (b *broadcast) Close() {}
