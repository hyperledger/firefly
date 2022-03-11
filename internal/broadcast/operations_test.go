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

package broadcast

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPrepareAndRunBatchBroadcast(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeSharedStorageBatchBroadcast,
	}
	bp := &fftypes.BatchPersisted{
		BatchHeader: fftypes.BatchHeader{
			ID: fftypes.NewUUID(),
		},
	}
	batch := &fftypes.Batch{
		BatchHeader: bp.BatchHeader,
	}
	addBatchBroadcastInputs(op, bp.ID)

	mps := bm.sharedstorage.(*sharedstoragemocks.Plugin)
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mdm.On("HydrateBatch", context.Background(), bp).Return(batch, nil)
	mdi.On("GetBatchByID", context.Background(), bp.ID).Return(bp, nil)
	mps.On("PublishData", context.Background(), mock.Anything).Return("123", nil)
	mdi.On("UpdateBatch", context.Background(), bp.ID, mock.MatchedBy(func(update database.Update) bool {
		info, _ := update.Finalize()
		assert.Equal(t, 1, len(info.SetOperations))
		assert.Equal(t, "payloadref", info.SetOperations[0].Field)
		val, _ := info.SetOperations[0].Value.Value()
		assert.Equal(t, "123", val)
		return true
	})).Return(nil)

	po, err := bm.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, bp.ID, po.Data.(batchBroadcastData).Batch.ID)

	complete, err := bm.RunOperation(context.Background(), opBatchBroadcast(op, batch))

	assert.True(t, complete)
	assert.NoError(t, err)

	mps.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestPrepareAndRunBatchBroadcastHydrateFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeSharedStorageBatchBroadcast,
	}
	bp := &fftypes.BatchPersisted{
		BatchHeader: fftypes.BatchHeader{
			ID: fftypes.NewUUID(),
		},
	}
	addBatchBroadcastInputs(op, bp.ID)

	mps := bm.sharedstorage.(*sharedstoragemocks.Plugin)
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mdm.On("HydrateBatch", context.Background(), bp).Return(nil, fmt.Errorf("pop"))
	mdi.On("GetBatchByID", context.Background(), bp.ID).Return(bp, nil)

	_, err := bm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "pop", err)

	mps.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestPrepareOperationNotSupported(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	_, err := bm.PrepareOperation(context.Background(), &fftypes.Operation{})

	assert.Regexp(t, "FF10371", err)
}

func TestPrepareOperationBatchBroadcastBadInput(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	op := &fftypes.Operation{
		Type:  fftypes.OpTypeSharedStorageBatchBroadcast,
		Input: fftypes.JSONObject{"id": "bad"},
	}

	_, err := bm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10142", err)
}

func TestPrepareOperationBatchBroadcastError(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	batchID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type:  fftypes.OpTypeSharedStorageBatchBroadcast,
		Input: fftypes.JSONObject{"id": batchID.String()},
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", context.Background(), batchID).Return(nil, fmt.Errorf("pop"))

	_, err := bm.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")
}

func TestPrepareOperationBatchBroadcastNotFound(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	batchID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type:  fftypes.OpTypeSharedStorageBatchBroadcast,
		Input: fftypes.JSONObject{"id": batchID.String()},
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetBatchByID", context.Background(), batchID).Return(nil, nil)

	_, err := bm.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)
}

func TestRunOperationNotSupported(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	complete, err := bm.RunOperation(context.Background(), &fftypes.PreparedOperation{})

	assert.False(t, complete)
	assert.Regexp(t, "FF10371", err)
}

func TestRunOperationBatchBroadcastInvalidData(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	op := &fftypes.Operation{}
	batch := &fftypes.Batch{
		Payload: fftypes.BatchPayload{
			Data: fftypes.DataArray{
				{Value: fftypes.JSONAnyPtr(`!json`)},
			},
		},
	}

	complete, err := bm.RunOperation(context.Background(), opBatchBroadcast(op, batch))

	assert.False(t, complete)
	assert.Regexp(t, "FF10137", err)
}

func TestRunOperationBatchBroadcastPublishFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	op := &fftypes.Operation{}
	batch := &fftypes.Batch{
		BatchHeader: fftypes.BatchHeader{
			ID: fftypes.NewUUID(),
		},
	}

	mps := bm.sharedstorage.(*sharedstoragemocks.Plugin)
	mps.On("PublishData", context.Background(), mock.Anything).Return("", fmt.Errorf("pop"))

	complete, err := bm.RunOperation(context.Background(), opBatchBroadcast(op, batch))

	assert.False(t, complete)
	assert.EqualError(t, err, "pop")

	mps.AssertExpectations(t)
}

func TestRunOperationBatchBroadcast(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	op := &fftypes.Operation{}
	batch := &fftypes.Batch{
		BatchHeader: fftypes.BatchHeader{
			ID: fftypes.NewUUID(),
		},
	}

	mps := bm.sharedstorage.(*sharedstoragemocks.Plugin)
	mdi := bm.database.(*databasemocks.Plugin)
	mps.On("PublishData", context.Background(), mock.Anything).Return("123", nil)
	mdi.On("UpdateBatch", context.Background(), batch.ID, mock.MatchedBy(func(update database.Update) bool {
		info, _ := update.Finalize()
		assert.Equal(t, 1, len(info.SetOperations))
		assert.Equal(t, "payloadref", info.SetOperations[0].Field)
		val, _ := info.SetOperations[0].Value.Value()
		assert.Equal(t, "123", val)
		return true
	})).Return(nil)

	complete, err := bm.RunOperation(context.Background(), opBatchBroadcast(op, batch))

	assert.True(t, complete)
	assert.NoError(t, err)

	mps.AssertExpectations(t)
	mdi.AssertExpectations(t)
}
