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

package operations

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockHandler struct {
	Complete bool
	Err      error
	Prepared *fftypes.PreparedOperation
}

func (m *mockHandler) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	return m.Prepared, m.Err
}

func (m *mockHandler) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (complete bool, err error) {
	return m.Complete, m.Err
}

func newTestOperations(t *testing.T) (*operationsManager, func()) {
	config.Reset()
	mdi := &databasemocks.Plugin{}
	mti := &tokenmocks.Plugin{}

	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything).Maybe()
	rag.RunFn = func(a mock.Arguments) {
		rag.ReturnArguments = mock.Arguments{
			a[1].(func(context.Context) error)(a[0].(context.Context)),
		}
	}

	mti.On("Name").Return("ut_tokens").Maybe()
	ctx, cancel := context.WithCancel(context.Background())
	om, err := NewOperationsManager(ctx, mdi, map[string]tokens.Plugin{"magic-tokens": mti})
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

	op := &fftypes.Operation{}

	_, err := om.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10349", err)
}

func TestPrepareOperationSuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeBlockchainBatchPin,
	}

	om.RegisterHandler(&mockHandler{}, []fftypes.OpType{fftypes.OpTypeBlockchainBatchPin})
	_, err := om.PrepareOperation(context.Background(), op)

	assert.NoError(t, err)
}

func TestRunOperationNotSupported(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	op := &fftypes.PreparedOperation{}

	err := om.RunOperation(context.Background(), op)
	assert.Regexp(t, "FF10349", err)
}

func TestRunOperationSuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	op := &fftypes.PreparedOperation{
		Type: fftypes.OpTypeBlockchainBatchPin,
	}

	om.RegisterHandler(&mockHandler{}, []fftypes.OpType{fftypes.OpTypeBlockchainBatchPin})
	err := om.RunOperation(context.Background(), op)

	assert.NoError(t, err)
}

func TestRunOperationSyncSuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op := &fftypes.PreparedOperation{
		ID:   fftypes.NewUUID(),
		Type: fftypes.OpTypeBlockchainBatchPin,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", ctx, op.ID, fftypes.OpStatusSucceeded, "", mock.Anything).Return(nil)

	om.RegisterHandler(&mockHandler{Complete: true}, []fftypes.OpType{fftypes.OpTypeBlockchainBatchPin})
	err := om.RunOperation(ctx, op)

	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestRunOperationFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	op := &fftypes.PreparedOperation{
		ID:   fftypes.NewUUID(),
		Type: fftypes.OpTypeBlockchainBatchPin,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", ctx, op.ID, fftypes.OpStatusFailed, "pop", mock.Anything).Return(nil)

	om.RegisterHandler(&mockHandler{Err: fmt.Errorf("pop")}, []fftypes.OpType{fftypes.OpTypeBlockchainBatchPin})
	err := om.RunOperation(ctx, op)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestRetryOperationSuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()
	op := &fftypes.Operation{
		ID:     opID,
		Plugin: "blockchain",
		Type:   fftypes.OpTypeBlockchainBatchPin,
		Status: fftypes.OpStatusFailed,
	}
	po := &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", ctx, opID).Return(op, nil)
	mdi.On("InsertOperation", ctx, mock.MatchedBy(func(newOp *fftypes.Operation) bool {
		assert.NotEqual(t, opID, newOp.ID)
		assert.Equal(t, "blockchain", newOp.Plugin)
		assert.Equal(t, fftypes.OpStatusPending, newOp.Status)
		assert.Equal(t, fftypes.OpTypeBlockchainBatchPin, newOp.Type)
		return true
	})).Return(nil)
	mdi.On("UpdateOperation", ctx, op.ID, mock.MatchedBy(func(update database.Update) bool {
		info, err := update.Finalize()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(info.SetOperations))
		assert.Equal(t, "retry", info.SetOperations[0].Field)
		val, err := info.SetOperations[0].Value.Value()
		assert.NoError(t, err)
		assert.Equal(t, op.ID.String(), val)
		return true
	})).Return(nil)

	om.RegisterHandler(&mockHandler{Prepared: po}, []fftypes.OpType{fftypes.OpTypeBlockchainBatchPin})
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
	op := &fftypes.Operation{
		ID:     opID,
		Plugin: "blockchain",
		Type:   fftypes.OpTypeBlockchainBatchPin,
		Status: fftypes.OpStatusFailed,
	}
	po := &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", ctx, opID).Return(op, fmt.Errorf("pop"))

	om.RegisterHandler(&mockHandler{Prepared: po}, []fftypes.OpType{fftypes.OpTypeBlockchainBatchPin})
	_, err := om.RetryOperation(ctx, "ns1", op.ID)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestRetryOperationInsertFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()
	op := &fftypes.Operation{
		ID:     opID,
		Plugin: "blockchain",
		Type:   fftypes.OpTypeBlockchainBatchPin,
		Status: fftypes.OpStatusFailed,
	}
	po := &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", ctx, opID).Return(op, nil)
	mdi.On("InsertOperation", ctx, mock.Anything).Return(fmt.Errorf("pop"))

	om.RegisterHandler(&mockHandler{Prepared: po}, []fftypes.OpType{fftypes.OpTypeBlockchainBatchPin})
	_, err := om.RetryOperation(ctx, "ns1", op.ID)

	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestRetryOperationUpdateFail(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()
	op := &fftypes.Operation{
		ID:     opID,
		Plugin: "blockchain",
		Type:   fftypes.OpTypeBlockchainBatchPin,
		Status: fftypes.OpStatusFailed,
	}
	po := &fftypes.PreparedOperation{
		ID:   op.ID,
		Type: op.Type,
	}

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", ctx, opID).Return(op, nil)
	mdi.On("InsertOperation", ctx, mock.Anything).Return(nil)
	mdi.On("UpdateOperation", ctx, op.ID, mock.Anything).Return(fmt.Errorf("pop"))

	om.RegisterHandler(&mockHandler{Prepared: po}, []fftypes.OpType{fftypes.OpTypeBlockchainBatchPin})
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
	mdi.On("ResolveOperation", ctx, opID, fftypes.OpStatusSucceeded, "", mock.Anything).Return(fmt.Errorf("pop"))

	om.writeOperationSuccess(ctx, opID)

	mdi.AssertExpectations(t)
}

func TestWriteOperationFailure(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer cancel()

	ctx := context.Background()
	opID := fftypes.NewUUID()

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", ctx, opID, fftypes.OpStatusFailed, "pop", mock.Anything).Return(fmt.Errorf("pop"))

	om.writeOperationFailure(ctx, opID, fmt.Errorf("pop"))

	mdi.AssertExpectations(t)
}
