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

package batchpin

import (
	"context"

	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type Submitter interface {
	SubmitPinnedBatch(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error
}

type batchPinSubmitter struct {
	database   database.Plugin
	identity   identity.Manager
	blockchain blockchain.Plugin
}

func NewBatchPinSubmitter(di database.Plugin, im identity.Manager, bi blockchain.Plugin) Submitter {
	return &batchPinSubmitter{
		database:   di,
		identity:   im,
		blockchain: bi,
	}
}

func (bp *batchPinSubmitter) SubmitPinnedBatch(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {

	tx := &fftypes.Transaction{
		ID: batch.Payload.TX.ID,
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Namespace: batch.Namespace,
			Signer:    batch.Key, // The transaction records on the on-chain identity
			Reference: batch.ID,
		},
		Created: fftypes.Now(),
		Status:  fftypes.OpStatusPending,
	}
	tx.Hash = tx.Subject.Hash()
	err := bp.database.UpsertTransaction(ctx, tx, false /* should be new, or idempotent replay */)
	if err != nil {
		return err
	}

	// The pending blockchain transaction
	op := fftypes.NewTXOperation(
		bp.blockchain,
		batch.Namespace,
		batch.Payload.TX.ID,
		"",
		fftypes.OpTypeBlockchainBatchPin,
		fftypes.OpStatusPending,
		"")
	err = bp.database.UpsertOperation(ctx, op, false)
	if err != nil {
		return err
	}

	// Write the batch pin to the blockchain
	return bp.blockchain.SubmitBatchPin(ctx, op.ID, nil /* TODO: ledger selection */, batch.Key, &blockchain.BatchPin{
		Namespace:      batch.Namespace,
		TransactionID:  batch.Payload.TX.ID,
		BatchID:        batch.ID,
		BatchHash:      batch.Hash,
		BatchPaylodRef: batch.PayloadRef,
		Contexts:       contexts,
	})
}
