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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/cachemocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
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

func (m *mockHandler) OnOperationUpdate(ctx context.Context, op *core.Operation, update *core.OperationUpdate) error {
	return m.UpdateErr
}

func newTestOperations(t *testing.T) (*operationsManager, func()) {
	coreconfig.Reset()
	config.Set(coreconfig.OpUpdateWorkerCount, 1)
	ctx, cancel := context.WithCancel(context.Background())
	mdi := &databasemocks.Plugin{}
	mdi.On("Capabilities").Return(&database.Capabilities{
		Concurrency: true,
	})
	mdm := &datamocks.Manager{}
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := txcommon.NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	ns := "ns1"
	om, err := NewOperationsManager(ctx, ns, mdi, txHelper, cmi)
	assert.NoError(t, err)
	cmi.AssertCalled(t, "GetCache", cache.NewCacheConfig(
		ctx,
		coreconfig.CacheOperationsLimit,
		coreconfig.CacheOperationsTTL,
		ns,
	))
	return om.(*operationsManager), cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewOperationsManager(context.Background(), "ns1", nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestCacheInitFail(t *testing.T) {
	cacheInitError := errors.New("Initialization error.")
	coreconfig.Reset()
	config.Set(coreconfig.OpUpdateWorkerCount, 1)
	mdi := &databasemocks.Plugin{}
	mdi.On("Capabilities").Return(&database.Capabilities{
		Concurrency: true,
	})
	mdm := &datamocks.Manager{}
	ctx := context.Background()
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(ctx, 100, 5*time.Minute), nil)
	txHelper, _ := txcommon.NewTransactionHelper(ctx, "ns1", mdi, mdm, cmi)
	ns := "ns1"
	ecmi := &cachemocks.Manager{}
	ecmi.On("GetCache", mock.Anything).Return(nil, cacheInitError)
	_, err := NewOperationsManager(ctx, ns, mdi, txHelper, ecmi)
	assert.Equal(t, cacheInitError, err)
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

	om.updater.workQueues = []chan *core.OperationUpdate{
		make(chan *core.OperationUpdate),
	}
	om.updater.cancelFunc()

	ctx := context.Background()
	op := &core.PreparedOperation{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Type:      core.OpTypeBlockchainPinBatch,
	}

	om.RegisterHandler(ctx, &mockHandler{Complete: true}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RunOperation(ctx, op)

	assert.NoError(t, err)
}

func TestRunOperationFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	om.updater.workQueues = []chan *core.OperationUpdate{
		make(chan *core.OperationUpdate),
	}
	om.updater.cancelFunc()

	ctx := context.Background()
	op := &core.PreparedOperation{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Type:      core.OpTypeBlockchainPinBatch,
	}

	om.RegisterHandler(ctx, &mockHandler{RunErr: fmt.Errorf("pop")}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RunOperation(ctx, op)

	assert.EqualError(t, err, "pop")
}

func TestRunOperationFailRemainPending(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	om.updater.workQueues = []chan *core.OperationUpdate{
		make(chan *core.OperationUpdate),
	}
	om.updater.cancelFunc()

	ctx := context.Background()
	op := &core.PreparedOperation{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Type:      core.OpTypeBlockchainPinBatch,
	}

	om.RegisterHandler(ctx, &mockHandler{RunErr: fmt.Errorf("pop")}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RunOperation(ctx, op, RemainPendingOnFailure)

	assert.EqualError(t, err, "pop")
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

	om.cache = cache.NewUmanagedCache(ctx, 100, 10*time.Minute)
	om.cacheOperation(op)

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("InsertOperation", ctx, mock.MatchedBy(func(newOp *core.Operation) bool {
		assert.NotEqual(t, opID, newOp.ID)
		assert.Equal(t, "blockchain", newOp.Plugin)
		assert.Equal(t, core.OpStatusPending, newOp.Status)
		assert.Equal(t, core.OpTypeBlockchainPinBatch, newOp.Type)
		return true
	})).Return(nil)
	mdi.On("UpdateOperation", ctx, "ns1", op.ID, mock.MatchedBy(func(update ffapi.Update) bool {
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
	newOp, err := om.RetryOperation(ctx, op.ID)

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
	mdi.On("GetOperationByID", ctx, "ns1", opID).Return(op, fmt.Errorf("pop"))

	om.RegisterHandler(ctx, &mockHandler{Prepared: po}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RetryOperation(ctx, op.ID)

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
	mdi.On("GetOperationByID", ctx, "ns1", opID).Return(op, nil)
	mdi.On("GetOperationByID", ctx, "ns1", opID2).Return(op2, nil)
	mdi.On("InsertOperation", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	om.RegisterHandler(ctx, &mockHandler{Prepared: po}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RetryOperation(ctx, op.ID)

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
	mdi.On("GetOperationByID", ctx, "ns1", opID).Return(op, nil)
	mdi.On("InsertOperation", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	om.RegisterHandler(ctx, &mockHandler{Prepared: po}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RetryOperation(ctx, op.ID)

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
	mdi.On("GetOperationByID", ctx, "ns1", opID).Return(op, nil)
	mdi.On("InsertOperation", ctx, mock.Anything).Return(nil)
	mdi.On("UpdateOperation", ctx, "ns1", op.ID, mock.Anything).Return(fmt.Errorf("pop"))

	om.RegisterHandler(ctx, &mockHandler{Prepared: po}, []core.OpType{core.OpTypeBlockchainPinBatch})
	_, err := om.RetryOperation(ctx, op.ID)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestResolveOperationByNamespacedIDOk(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()
	errStr := "my error"
	opUpdate := &core.OperationUpdateDTO{
		Status: core.OpStatusSucceeded,
		Error:  &errStr,
		Output: fftypes.JSONObject{
			"my": "data",
		},
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("UpdateOperation", ctx, "ns1", opID, mock.MatchedBy(updateMatcher([][]string{
		{"status", "Succeeded"},
		{"error", errStr},
		{"output", opUpdate.Output.String()},
	}))).Return(nil)

	err := om.ResolveOperationByID(ctx, opID, opUpdate)

	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}
