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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

type Submitter interface {
	core.Named

	SubmitPinnedBatch(ctx context.Context, batch *core.BatchPersisted, contexts []*fftypes.Bytes32, payloadRef string) error

	// From operations.OperationHandler
	PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error)
	RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error)
}

type batchPinSubmitter struct {
	database   database.Plugin
	identity   identity.Manager
	blockchain blockchain.Plugin
	metrics    metrics.Manager
	operations operations.Manager
}

func NewBatchPinSubmitter(ctx context.Context, di database.Plugin, im identity.Manager, bi blockchain.Plugin, mm metrics.Manager, om operations.Manager) (Submitter, error) {
	if di == nil || im == nil || bi == nil || mm == nil || om == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "BatchPinSubmitter")
	}
	bp := &batchPinSubmitter{
		database:   di,
		identity:   im,
		blockchain: bi,
		metrics:    mm,
		operations: om,
	}
	om.RegisterHandler(ctx, bp, []core.OpType{
		core.OpTypeBlockchainPinBatch,
	})
	return bp, nil
}

func (bp *batchPinSubmitter) Name() string {
	return "BatchPinSubmitter"
}

func (bp *batchPinSubmitter) SubmitPinnedBatch(ctx context.Context, batch *core.BatchPersisted, contexts []*fftypes.Bytes32, payloadRef string) error {
	// The pending blockchain transaction
	op := core.NewOperation(
		bp.blockchain,
		batch.Namespace,
		batch.TX.ID,
		core.OpTypeBlockchainPinBatch)
	addBatchPinInputs(op, batch.ID, contexts, payloadRef)
	if err := bp.operations.AddOrReuseOperation(ctx, op); err != nil {
		return err
	}

	if bp.metrics.IsMetricsEnabled() {
		bp.metrics.CountBatchPin()
	}
	_, err := bp.operations.RunOperation(ctx, opBatchPin(op, batch, contexts, payloadRef))
	return err
}
