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
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockHandler struct {
	Complete bool
	Err      error
}

func (m *mockHandler) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	return nil, m.Err
}

func (m *mockHandler) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (complete bool, err error) {
	return m.Complete, m.Err
}

func newTestOperations(t *testing.T) (*operationsManager, func()) {
	config.Reset()
	mdi := &databasemocks.Plugin{}
	mti := &tokenmocks.Plugin{}
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
	assert.Regexp(t, "FF10346", err)
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
	assert.Regexp(t, "FF10346", err)
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
	err := om.RunOperation(context.Background(), op)

	assert.NoError(t, err)
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
	err := om.RunOperation(context.Background(), op)

	assert.EqualError(t, err, "pop")
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
