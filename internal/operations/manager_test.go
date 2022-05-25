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
package operations

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockHandler struct {
	Complete  bool
	RunErr    error
	Prepared  *core.PreparedOperation
	Outputs   fftypes.JSONObject
	UpdateErr error
}

func (m *mockHandler) Name() string {
	return "MockHandler"
}

func (m *mockHandler) PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error) {
	return m.Prepared, m.RunErr
}

func (m *mockHandler) RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error) {
	return m.Outputs, m.Complete, m.RunErr
}

func (m *mockHandler) OnOperationUpdate(ctx context.Context, op *core.Operation, update *OperationUpdate) error {
	return m.UpdateErr
}

func newTestOperations(t *testing.T) (*operationsManager, func()) {
	coreconfig.Reset()
	config.Set(coreconfig.OpUpdateWorkerCount, 1)
	mdi := &databasemocks.Plugin{}
	mdi.On("Capabilities").Return(&database.Capabilities{
		Concurrency: true,
	})
	mdm := &datamocks.Manager{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	om, err := NewOperationsManager(ctx, mdi, txHelper)
	assert.NoError(t, err)
	return om.(*operationsManager), cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewOperationsManager(context.Background(), nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestPrepareOperationNotSupported(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	op := &core.Operation{}

	_, err := om.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10371", err)
}

func TestPrepareOperationSuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op := &core.Operation{
		Type: core.OpTypeBlockchainPinBatch,
	}

	om.RegisterHandler(ctx, &mockHandler{}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.PrepareOperation(context.Background(), op)

	assert.NoError(t, err)
}

func TestRunOperationNotSupported(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	op := &core.PreparedOperation{}

	_, err := om.RunOperation(context.Background(), op)
	assert.Regexp(t, "FF10371", err)
}

func TestRunOperationSuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op := &core.PreparedOperation{
		Type: core.OpTypeBlockchainPinBatch,
	}

	om.RegisterHandler(ctx, &mockHandler{Outputs: fftypes.JSONObject{"test": "output"}}, []core.OpType{core.OpTypeBlockchainPinBatch})
	outputs, err := om.RunOperation(context.Background(), op)
	assert.Equal(t, "output", outputs.GetString("test"))

	assert.NoError(t, err)
}

func TestRunOperationSyncSuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op := &core.PreparedOperation{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Type:      core.OpTypeBlockchainPinBatch,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", ctx, "ns1", op.ID, core.OpStatusSucceeded, "", mock.Anything).Return(nil)

	om.RegisterHandler(ctx, &mockHandler{Complete: true}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RunOperation(ctx, op)

	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestRunOperationFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op := &core.PreparedOperation{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Type:      core.OpTypeBlockchainPinBatch,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", ctx, "ns1", op.ID, core.OpStatusFailed, "pop", mock.Anything).Return(nil)

	om.RegisterHandler(ctx, &mockHandler{RunErr: fmt.Errorf("pop")}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RunOperation(ctx, op)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestRunOperationFailRemainPending(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op := &core.PreparedOperation{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Type:      core.OpTypeBlockchainPinBatch,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", ctx, "ns1", op.ID, core.OpStatusPending, "pop", mock.Anything).Return(nil)

	om.RegisterHandler(ctx, &mockHandler{RunErr: fmt.Errorf("pop")}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RunOperation(ctx, op, RemainPendingOnFailure)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestRetryOperationSuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()
	op := &core.Operation{
		ID:        opID,
		Namespace: "ns1",
		Plugin:    "blockchain",
		Type:      core.OpTypeBlockchainPinBatch,
		Status:    core.OpStatusFailed,
	}
	po := &core.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", ctx, opID).Return(op, nil)
	mdi.On("InsertOperation", ctx, mock.MatchedBy(func(newOp *core.Operation) bool {
		assert.NotEqual(t, opID, newOp.ID)
		assert.Equal(t, "blockchain", newOp.Plugin)
		assert.Equal(t, core.OpStatusPending, newOp.Status)
		assert.Equal(t, core.OpTypeBlockchainPinBatch, newOp.Type)
		return true
	})).Return(nil)
	mdi.On("UpdateOperation", ctx, "ns1", op.ID, mock.MatchedBy(func(update database.Update) bool {
		info, err := update.Finalize()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(info.SetOperations))
		assert.Equal(t, "retry", info.SetOperations[0].Field)
		val, err := info.SetOperations[0].Value.Value()
		assert.NoError(t, err)
		assert.Equal(t, op.ID.String(), val)
		return true
	})).Return(nil)

	om.RegisterHandler(ctx, &mockHandler{Prepared: po}, []core.OpType{core.OpTypeBlockchainPinBatch})
	newOp, err := om.RetryOperation(ctx, "ns1", op.ID)

	assert.NoError(t, err)
	assert.NotNil(t, newOp)

	mdi.AssertExpectations(t)
}

func TestRetryOperationGetFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()
	op := &core.Operation{
		ID:     opID,
		Plugin: "blockchain",
		Type:   core.OpTypeBlockchainPinBatch,
		Status: core.OpStatusFailed,
	}
	po := &core.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", ctx, opID).Return(op, fmt.Errorf("pop"))

	om.RegisterHandler(ctx, &mockHandler{Prepared: po}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RetryOperation(ctx, "ns1", op.ID)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestRetryTwiceOperationInsertFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()
	opID2 := fftypes.NewUUID()
	op := &core.Operation{
		ID:     opID,
		Plugin: "blockchain",
		Type:   core.OpTypeBlockchainPinBatch,
		Status: core.OpStatusFailed,
		Retry:  opID2,
	}
	op2 := &core.Operation{
		ID:     opID2,
		Plugin: "blockchain",
		Type:   core.OpTypeBlockchainPinBatch,
		Status: core.OpStatusFailed,
	}
	po := &core.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", ctx, opID).Return(op, nil)
	mdi.On("GetOperationByID", ctx, opID2).Return(op2, nil)
	mdi.On("InsertOperation", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	om.RegisterHandler(ctx, &mockHandler{Prepared: po}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RetryOperation(ctx, "ns1", op.ID)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestRetryOperationInsertFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()
	op := &core.Operation{
		ID:     opID,
		Plugin: "blockchain",
		Type:   core.OpTypeBlockchainPinBatch,
		Status: core.OpStatusFailed,
	}
	po := &core.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", ctx, opID).Return(op, nil)
	mdi.On("InsertOperation", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	om.RegisterHandler(ctx, &mockHandler{Prepared: po}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RetryOperation(ctx, "ns1", op.ID)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestRetryOperationUpdateFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()
	op := &core.Operation{
		ID:        opID,
		Namespace: "ns1",
		Plugin:    "blockchain",
		Type:      core.OpTypeBlockchainPinBatch,
		Status:    core.OpStatusFailed,
	}
	po := &core.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", ctx, opID).Return(op, nil)
	mdi.On("InsertOperation", ctx, mock.Anything).Return(nil)
	mdi.On("UpdateOperation", ctx, "ns1", op.ID, mock.Anything).Return(fmt.Errorf("pop"))

	om.RegisterHandler(ctx, &mockHandler{Prepared: po}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RetryOperation(ctx, "ns1", op.ID)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestWriteOperationSuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", ctx, "ns1", opID, core.OpStatusSucceeded, "", mock.Anything).Return(fmt.Errorf("pop"))

	om.writeOperationSuccess(ctx, "ns1", opID, nil)

	mdi.AssertExpectations(t)
}

func TestWriteOperationFailure(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", ctx, "ns1", opID, core.OpStatusFailed, "pop", mock.Anything).Return(fmt.Errorf("pop"))

	om.writeOperationFailure(ctx, "ns1", opID, nil, fmt.Errorf("pop"), core.OpStatusFailed)

	mdi.AssertExpectations(t)
}

func TestTransferResultBadUUID(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mdx.On("Capabilities").Return(&dataexchange.Capabilities{
		Manifest: true,
	})
	mde := &dataexchangemocks.DXEvent{}
	mde.On("TransferResult").Return(&dataexchange.TransferResult{
		TrackingID: "wrongun",
		Status:     core.OpStatusSucceeded,
		TransportStatusUpdate: core.TransportStatusUpdate{
			Info:     fftypes.JSONObject{"extra": "info"},
			Manifest: "Sally",
		},
	})
	om.TransferResult(mdx, mde)
}

func TestTransferResultManifestMismatch(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()
	om.updater.conf.workerCount = 0

	opID1 := fftypes.NewUUID()
	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*core.Operation{
		{
			ID:        opID1,
			Namespace: "ns1",
			Type:      core.OpTypeDataExchangeSendBatch,
			Input: fftypes.JSONObject{
				"batch": fftypes.NewUUID().String(),
			},
		},
	}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, "ns1", opID1, core.OpStatusFailed, mock.MatchedBy(func(errorMsg string) bool {
		return strings.Contains(errorMsg, "FF10329")
	}), fftypes.JSONObject{
		"extra": "info",
	}).Return(nil)
	mdi.On("GetBatchByID", mock.Anything, mock.Anything).Return(&core.BatchPersisted{
		Manifest: fftypes.JSONAnyPtr("my-manifest"),
	}, nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mdx.On("Capabilities").Return(&dataexchange.Capabilities{
		Manifest: true,
	})
	mde := &dataexchangemocks.DXEvent{}
	mde.On("Ack").Return()
	mde.On("TransferResult").Return(&dataexchange.TransferResult{
		TrackingID: opID1.String(),
		Status:     core.OpStatusSucceeded,
		TransportStatusUpdate: core.TransportStatusUpdate{
			Info:     fftypes.JSONObject{"extra": "info"},
			Manifest: "Sally",
		},
	})
	om.TransferResult(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestTransferResultHashMismatch(t *testing.T) {

	om, cancel := newTestOperations(t)
	cancel()
	om.updater.conf.workerCount = 0

	opID1 := fftypes.NewUUID()
	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*core.Operation{
		{
			ID:        opID1,
			Namespace: "ns1",
			Type:      core.OpTypeDataExchangeSendBlob,
			Input: fftypes.JSONObject{
				"hash": "Bob",
			},
		},
	}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, "ns1", opID1, core.OpStatusFailed, mock.MatchedBy(func(errorMsg string) bool {
		return strings.Contains(errorMsg, "FF10348")
	}), fftypes.JSONObject{
		"extra": "info",
	}).Return(nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mdx.On("Capabilities").Return(&dataexchange.Capabilities{
		Manifest: true,
	})
	mde := &dataexchangemocks.DXEvent{}
	mde.On("Ack").Return()
	mde.On("TransferResult").Return(&dataexchange.TransferResult{
		TrackingID: opID1.String(),
		Status:     core.OpStatusSucceeded,
		TransportStatusUpdate: core.TransportStatusUpdate{
			Info: fftypes.JSONObject{"extra": "info"},
			Hash: "Sally",
		},
	})
	om.TransferResult(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestTransferResultBatchLookupFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	cancel()
	om.updater.conf.workerCount = 0

	opID1 := fftypes.NewUUID()
	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*core.Operation{
		{
			ID:   opID1,
			Type: core.OpTypeDataExchangeSendBatch,
			Input: fftypes.JSONObject{
				"batch": fftypes.NewUUID().String(),
			},
		},
	}, nil, nil)
	mdi.On("GetBatchByID", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mdx.On("Capabilities").Return(&dataexchange.Capabilities{
		Manifest: true,
	})
	mde := &dataexchangemocks.DXEvent{}
	mde.On("TransferResult").Return(&dataexchange.TransferResult{
		TrackingID: opID1.String(),
		Status:     core.OpStatusSucceeded,
		TransportStatusUpdate: core.TransportStatusUpdate{
			Info:     fftypes.JSONObject{"extra": "info"},
			Manifest: "Sally",
		},
	})
	om.TransferResult(mdx, mde)

	mdi.AssertExpectations(t)

}

func TestResolveOperationByIDOk(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op := &core.Operation{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Type:      core.OpTypeBlockchainPinBatch,
		Status:    core.OpStatusSucceeded,
		Error:     "my error",
		Output: fftypes.JSONObject{
			"my": "data",
		},
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", ctx, "ns1", op.ID, core.OpStatusSucceeded, "my error", fftypes.JSONObject{
		"my": "data",
	}).Return(nil)

	_, err := om.ResolveOperationByID(ctx, op.ID.String(), op)

	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestResolveOperationBadID(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op := &core.Operation{}

	_, err := om.ResolveOperationByID(ctx, "badness", op)

	assert.Regexp(t, "FF00138", err)

}
