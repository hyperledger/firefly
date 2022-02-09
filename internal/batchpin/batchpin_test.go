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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("metrics")

func newTestBatchPinSubmitter(t *testing.T) *batchPinSubmitter {
	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mmi := &metricsmocks.Manager{}
	mbi.On("Name").Return("ut").Maybe()
	bps := NewBatchPinSubmitter(mdi, mim, mbi, mmi).(*batchPinSubmitter)
	return bps
}

func TestSubmitPinnedBatchOk(t *testing.T) {
	bp := newTestBatchPinSubmitter(t)
	ctx := context.Background()

	mbi := bp.blockchain.(*blockchainmocks.Plugin)
	mdi := bp.database.(*databasemocks.Plugin)

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "id1",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	contexts := []*fftypes.Bytes32{}

	mdi.On("InsertOperation", ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		assert.Equal(t, fftypes.OpTypeBlockchainBatchPin, op.Type)
		assert.Equal(t, "ut", op.Plugin)
		assert.Equal(t, *batch.Payload.TX.ID, *op.Transaction)
		return true
	})).Return(nil)
	mbi.On("SubmitBatchPin", ctx, mock.Anything, (*fftypes.UUID)(nil), "0x12345", mock.Anything).Return(nil)

	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.NoError(t, err)
}

func TestSubmitPinnedBatchWithMetricsOk(t *testing.T) {
	metrics.Registry()
	config.Set(config.MetricsEnabled, true)
	bp := newTestBatchPinSubmitter(t)
	ctx := context.Background()

	mbi := bp.blockchain.(*blockchainmocks.Plugin)
	mdi := bp.database.(*databasemocks.Plugin)

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "id1",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	contexts := []*fftypes.Bytes32{}

	mdi.On("InsertOperation", ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		assert.Equal(t, fftypes.OpTypeBlockchainBatchPin, op.Type)
		assert.Equal(t, "ut", op.Plugin)
		assert.Equal(t, *batch.Payload.TX.ID, *op.Transaction)
		return true
	})).Return(nil)
	mbi.On("SubmitBatchPin", ctx, mock.Anything, (*fftypes.UUID)(nil), "0x12345", mock.Anything).Return(nil)

	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.NoError(t, err)
}

func TestSubmitPinnedBatchOpFail(t *testing.T) {

	bp := newTestBatchPinSubmitter(t)
	ctx := context.Background()

	mdi := bp.database.(*databasemocks.Plugin)

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "id1",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	contexts := []*fftypes.Bytes32{}

	mdi.On("InsertOperation", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.Regexp(t, "pop", err)

}
