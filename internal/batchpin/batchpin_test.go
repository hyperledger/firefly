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
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("metrics")

func newTestBatchPinSubmitter(t *testing.T, enableMetrics bool) *batchPinSubmitter {
	config.Reset()

	mdi := &databasemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	mbi := &blockchainmocks.Plugin{}
	mmi := &metricsmocks.Manager{}
	mom := &operationmocks.Manager{}
	mmi.On("IsMetricsEnabled").Return(enableMetrics)
	mom.On("RegisterHandler", mock.Anything, mock.Anything, mock.Anything)
	if enableMetrics {
		mmi.On("CountBatchPin").Return()
	}
	mbi.On("Name").Return("ut").Maybe()
	bps, err := NewBatchPinSubmitter(context.Background(), mdi, mim, mbi, mmi, mom)
	assert.NoError(t, err)
	return bps.(*batchPinSubmitter)
}

func TestInitFail(t *testing.T) {
	_, err := NewBatchPinSubmitter(context.Background(), nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestName(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)
	assert.Equal(t, "BatchPinSubmitter", bp.Name())
}

func TestSubmitPinnedBatchOk(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)
	ctx := context.Background()

	mdi := bp.database.(*databasemocks.Plugin)
	mmi := bp.metrics.(*metricsmocks.Manager)
	mom := bp.operations.(*operationmocks.Manager)

	batch := &fftypes.BatchPersisted{
		BatchHeader: fftypes.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: fftypes.SignerRef{
				Author: "id1",
				Key:    "0x12345",
			},
		},
		TX: fftypes.TransactionRef{
			ID: fftypes.NewUUID(),
		},
	}
	contexts := []*fftypes.Bytes32{}

	mom.On("AddOrReuseOperation", ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		assert.Equal(t, fftypes.OpTypeBlockchainPinBatch, op.Type)
		assert.Equal(t, "ut", op.Plugin)
		assert.Equal(t, *batch.TX.ID, *op.Transaction)
		return true
	})).Return(nil)
	mmi.On("IsMetricsEnabled").Return(false)
	mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(batchPinData)
		return op.Type == fftypes.OpTypeBlockchainPinBatch && data.Batch == batch
	})).Return(nil)

	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mmi.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestSubmitPinnedBatchWithMetricsOk(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, true)
	ctx := context.Background()

	mdi := bp.database.(*databasemocks.Plugin)
	mmi := bp.metrics.(*metricsmocks.Manager)
	mom := bp.operations.(*operationmocks.Manager)

	batch := &fftypes.BatchPersisted{
		BatchHeader: fftypes.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: fftypes.SignerRef{
				Author: "id1",
				Key:    "0x12345",
			},
		},
		TX: fftypes.TransactionRef{
			ID: fftypes.NewUUID(),
		},
	}
	contexts := []*fftypes.Bytes32{}

	mom.On("AddOrReuseOperation", ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		assert.Equal(t, fftypes.OpTypeBlockchainPinBatch, op.Type)
		assert.Equal(t, "ut", op.Plugin)
		assert.Equal(t, *batch.TX.ID, *op.Transaction)
		return true
	})).Return(nil)
	mmi.On("IsMetricsEnabled").Return(true)
	mom.On("RunOperation", mock.Anything, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(batchPinData)
		return op.Type == fftypes.OpTypeBlockchainPinBatch && data.Batch == batch
	})).Return(nil)

	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mmi.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestSubmitPinnedBatchOpFail(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)
	ctx := context.Background()

	mom := bp.operations.(*operationmocks.Manager)
	mmi := bp.metrics.(*metricsmocks.Manager)

	batch := &fftypes.BatchPersisted{
		BatchHeader: fftypes.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: fftypes.SignerRef{
				Author: "id1",
				Key:    "0x12345",
			},
		},
		TX: fftypes.TransactionRef{
			ID: fftypes.NewUUID(),
		},
	}
	contexts := []*fftypes.Bytes32{}

	mom.On("AddOrReuseOperation", ctx, mock.Anything).Return(fmt.Errorf("pop"))
	mmi.On("IsMetricsEnabled").Return(false)
	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.Regexp(t, "pop", err)

}
