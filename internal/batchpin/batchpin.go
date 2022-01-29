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

package batchpin

import (
	"context"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type Submitter interface {
	SubmitPinnedBatch(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error
}

type batchPinSubmitter struct {
	database       database.Plugin
	identity       identity.Manager
	blockchain     blockchain.Plugin
	metricsEnabled bool
}

func NewBatchPinSubmitter(di database.Plugin, im identity.Manager, bi blockchain.Plugin) Submitter {
	return &batchPinSubmitter{
		database:       di,
		identity:       im,
		blockchain:     bi,
		metricsEnabled: config.GetBool(config.MetricsEnabled),
	}
}

func (bp *batchPinSubmitter) SubmitPinnedBatch(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {

	// The pending blockchain transaction
	op := fftypes.NewTXOperation(
		bp.blockchain,
		batch.Namespace,
		batch.Payload.TX.ID,
		"",
		fftypes.OpTypeBlockchainBatchPin,
		fftypes.OpStatusPending)
	if err := bp.database.InsertOperation(ctx, op); err != nil {
		return err
	}

	if bp.metricsEnabled {
		metrics.BatchPinCounter.Inc()
	}
	// Write the batch pin to the blockchain
	return bp.blockchain.SubmitBatchPin(ctx, op.ID, nil /* TODO: ledger selection */, batch.Key, &blockchain.BatchPin{
		Namespace:       batch.Namespace,
		TransactionID:   batch.Payload.TX.ID,
		BatchID:         batch.ID,
		BatchHash:       batch.Hash,
		BatchPayloadRef: batch.PayloadRef,
		Contexts:        contexts,
	})
}
