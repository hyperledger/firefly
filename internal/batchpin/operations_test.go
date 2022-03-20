// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in comdiliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or imdilied.
// See the License for the specific language governing permissions and
// limitations under the License.

package batchpin

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPrepareAndRunBatchPin(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	op := &fftypes.Operation{
		Type: fftypes.OpTypeBlockchainPinBatch,
		ID:   fftypes.NewUUID(),
	}
	batch := &fftypes.BatchPersisted{
		BatchHeader: fftypes.BatchHeader{
			ID: fftypes.NewUUID(),
			SignerRef: fftypes.SignerRef{
				Key: "0x123",
			},
		},
	}
	contexts := []*fftypes.Bytes32{
		fftypes.NewRandB32(),
		fftypes.NewRandB32(),
	}
	addBatchPinInputs(op, batch.ID, contexts)

	mbi := bp.blockchain.(*blockchainmocks.Plugin)
	mdi := bp.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", context.Background(), batch.ID).Return(batch, nil)
	mbi.On("SubmitBatchPin", context.Background(), op.ID, mock.Anything, "0x123", mock.Anything).Return(nil)

	po, err := bp.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, batch, po.Data.(batchPinData).Batch)

	_, complete, err := bp.RunOperation(context.Background(), opBatchPin(op, batch, contexts))

	assert.False(t, complete)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestPrepareOperationNotSupported(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	po, err := bp.PrepareOperation(context.Background(), &fftypes.Operation{})

	assert.Nil(t, po)
	assert.Regexp(t, "FF10371", err)
}

func TestPrepareOperationBatchPinBadBatch(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	op := &fftypes.Operation{
		Type:  fftypes.OpTypeBlockchainPinBatch,
		Input: fftypes.JSONObject{"batch": "bad"},
	}

	_, err := bp.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10142", err)
}

func TestPrepareOperationBatchPinBadContext(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	op := &fftypes.Operation{
		Type: fftypes.OpTypeBlockchainPinBatch,
		Input: fftypes.JSONObject{
			"batch":    fftypes.NewUUID().String(),
			"contexts": []string{"bad"},
		},
	}

	_, err := bp.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10232", err)
}

func TestRunOperationNotSupported(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	_, complete, err := bp.RunOperation(context.Background(), &fftypes.PreparedOperation{})

	assert.False(t, complete)
	assert.Regexp(t, "FF10371", err)
}

func TestPrepareOperationBatchPinError(t *testing.T) {
	bp := newTestBatchPinSubmitter(t, false)

	batchID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type: fftypes.OpTypeBlockchainPinBatch,
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
	op := &fftypes.Operation{
		Type: fftypes.OpTypeBlockchainPinBatch,
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
