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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPrepareAndRunBatchPin(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	op := &core.Operation{
		Type: core.OpTypeBlockchainPinBatch,
		ID:   fftypes.NewUUID(),
	}
	batch := &core.BatchPersisted{
		BatchHeader: core.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Key: "0x123",
			},
		},
	}
	contexts := []*fftypes.Bytes32{
		fftypes.NewRandB32(),
		fftypes.NewRandB32(),
	}
	addBatchPinInputs(op, batch.ID, contexts, "payload1")

	mbi := bp.blockchain.(*blockchainmocks.Plugin)
	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", context.Background(), batch.ID).Return(batch, nil)
	mbi.On("SubmitBatchPin", context.Background(), op.ID, "0x123", mock.Anything).Return(nil)

	po, err := bp.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, batch, po.Data.(batchPinData).Batch)

	_, complete, err := bp.RunOperation(context.Background(), opBatchPin(op, batch, contexts, "payload1"))

	assert.False(t, complete)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestPrepareOperationNotSupported(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	po, err := bp.PrepareOperation(context.Background(), &core.Operation{})

	assert.Nil(t, po)
	assert.Regexp(t, "FF10371", err)
}

func TestPrepareOperationBatchPinBadBatch(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	op := &core.Operation{
		Type:  core.OpTypeBlockchainPinBatch,
		Input: fftypes.JSONObject{"batch": "bad"},
	}

	_, err := bp.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF00138", err)
}

func TestPrepareOperationBatchPinBadContext(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	op := &core.Operation{
		Type: core.OpTypeBlockchainPinBatch,
		Input: fftypes.JSONObject{
			"batch":    fftypes.NewUUID().String(),
			"contexts": []string{"bad"},
		},
	}

	_, err := bp.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF00107", err)
}

func TestRunOperationNotSupported(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	_, complete, err := bp.RunOperation(context.Background(), &core.PreparedOperation{})

	assert.False(t, complete)
	assert.Regexp(t, "FF10378", err)
}

func TestPrepareOperationBatchPinError(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	batchID := fftypes.NewUUID()
	op := &core.Operation{
		Type: core.OpTypeBlockchainPinBatch,
		Input: fftypes.JSONObject{
			"batch":    batchID.String(),
			"contexts": []string{},
		},
	}

	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", context.Background(), batchID).Return(nil, fmt.Errorf("pop"))

	_, err := bp.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")
}

func TestPrepareOperationBatchPinNotFound(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	batchID := fftypes.NewUUID()
	op := &core.Operation{
		Type: core.OpTypeBlockchainPinBatch,
		Input: fftypes.JSONObject{
			"batch":    batchID.String(),
			"contexts": []string{},
		},
	}

	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", context.Background(), batchID).Return(nil, nil)

	_, err := bp.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)
}

func TestOperationUpdate(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)
	assert.NoError(t, bp.OnOperationUpdate(context.Background(), nil, nil))
}
