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

	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/identity"
)

type Submitter interface {
	SubmitPinnedBatch(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error
}

type batchPinSubmitter struct {
	database   database.Plugin
	identity   identity.Plugin
	blockchain blockchain.Plugin
}

func NewBatchPinSubmitter(di database.Plugin, ii identity.Plugin, bi blockchain.Plugin) Submitter {
	return &batchPinSubmitter{
		database:   di,
		identity:   ii,
		blockchain: bi,
	}
}

func (bp *batchPinSubmitter) SubmitPinnedBatch(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {

	signingIdentity, err := bp.identity.Resolve(ctx, batch.Author)
	if err == nil {
		err = bp.blockchain.VerifyIdentitySyntax(ctx, signingIdentity)
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
	err = bp.database.UpsertTransaction(ctx, tx, true, false /* should be new, or idempotent replay */)
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
	return bp.blockchain.SubmitBatchPin(ctx, nil /* TODO: ledger selection */, signingIdentity, &blockchain.BatchPin{
		Namespace:      batch.Namespace,
		TransactionID:  batch.Payload.TX.ID,
		BatchID:        batch.ID,
		BatchHash:      batch.Hash,
		BatchPaylodRef: batch.PayloadRef,
		Contexts:       contexts,
	})
}
