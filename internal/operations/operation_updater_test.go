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

package operations

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestOperationUpdater(t *testing.T) *operationUpdater {
	return newTestOperationUpdaterCommon(t, &database.Capabilities{Concurrency: true})
}

func newTestOperationUpdaterNoConcrrency(t *testing.T) *operationUpdater {
	return newTestOperationUpdaterCommon(t, &database.Capabilities{Concurrency: false})
}

func newTestOperationUpdaterCommon(t *testing.T, dbCapabilities *database.Capabilities) *operationUpdater {
	config.Reset()
	config.Set(config.OpUpdateWorkerCount, 1)
	config.Set(config.OpUpdateWorkerBatchTimeout, "1s")
	config.Set(config.OpUpdateWorkerBatchMaxInserts, 200)
	logrus.SetLevel(logrus.DebugLevel)

	mdi := &databasemocks.Plugin{}
	mdi.On("Capabilities").Return(dbCapabilities)
	mdm := &datamocks.Manager{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	return newOperationUpdater(context.Background(), mdi, txHelper)
}

func TestNewOperationUpdaterNoConcurrency(t *testing.T) {
	ou := newTestOperationUpdaterNoConcrrency(t)
	defer ou.close()
	assert.Zero(t, ou.conf.workerCount)
}

func TestSubmitUpdateClosed(t *testing.T) {
	ou := newTestOperationUpdater(t)
	ou.close()
	ou.workQueues = []chan *OperationUpdate{
		make(chan *OperationUpdate),
	}
	ou.cancelFunc()
	err := ou.SubmitOperationUpdate(ou.ctx, &OperationUpdate{
		ID: fftypes.NewUUID(),
	})
	assert.Regexp(t, "FF10158", err)
}

func TestSubmitUpdateSyncFallbackOpNotFound(t *testing.T) {
	ou := newTestOperationUpdaterNoConcrrency(t)
	defer ou.close()
	customCtx := context.WithValue(context.Background(), "dbtx", "on this context")

	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", customCtx, mock.Anything).Run(func(args mock.Arguments) {
		err := args[1].(func(context.Context) error)(customCtx)
		assert.NoError(t, err)
	}).Return(nil)
	mdi.On("GetOperations", customCtx, mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("GetTransactions", customCtx, mock.Anything, mock.Anything).Return(nil, nil, nil)

	err := ou.SubmitOperationUpdate(customCtx, &OperationUpdate{
		ID: fftypes.NewUUID(),
	})

	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestSubmitUpdateWorkerE2ESuccess(t *testing.T) {
	om, cancel := newTestOperations(t)
	defer om.WaitStop()
	defer cancel()
	om.updater.conf.maxInserts = 2

	opID1 := fftypes.NewUUID()
	opID2 := fftypes.NewUUID()
	opID3 := fftypes.NewUUID()
	tx1 := &fftypes.Transaction{ID: fftypes.NewUUID()}

	done := make(chan struct{})

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{ID: opID1, Type: fftypes.OpTypeBlockchainInvoke, Transaction: tx1.ID},
		{ID: opID2, Type: fftypes.OpTypeTokenTransfer, Input: fftypes.JSONObject{"test": "test"}},
		{ID: opID3, Type: fftypes.OpTypeTokenApproval, Input: fftypes.JSONObject{"test": "test"}},
	}, nil, nil)
	mdi.On("GetTransactions", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Transaction{tx1}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, opID1, fftypes.OpStatusSucceeded, "", fftypes.JSONObject(nil)).Return(nil)
	mdi.On("UpdateTransaction", mock.Anything, tx1.ID, mock.Anything).Return(nil)
	mdi.On("ResolveOperation", mock.Anything, opID2, fftypes.OpStatusFailed, "err1", fftypes.JSONObject{"test": true}).Return(nil)
	mdi.On("ResolveOperation", mock.Anything, opID3, fftypes.OpStatusFailed, "err2", fftypes.JSONObject(nil)).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferOpFailed
	})).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeApprovalOpFailed
	})).Return(nil).Run(func(args mock.Arguments) {
		close(done)
	})

	om.Start()

	err := om.SubmitOperationUpdate(&OperationUpdate{
		ID:             opID1,
		State:          fftypes.OpStatusSucceeded,
		BlockchainTXID: "tx12345",
	})
	assert.NoError(t, err)
	err = om.SubmitOperationUpdate(&OperationUpdate{
		ID:           opID2,
		State:        fftypes.OpStatusFailed,
		ErrorMessage: "err1",
		Output:       fftypes.JSONObject{"test": true},
	})
	assert.NoError(t, err)
	err = om.SubmitOperationUpdate(&OperationUpdate{
		ID:           opID3,
		State:        fftypes.OpStatusFailed,
		ErrorMessage: "err2",
	})
	assert.NoError(t, err)

	<-done

	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestUpdateLoopExitRetryCancelledContext(t *testing.T) {
	ou := newTestOperationUpdater(t)
	defer ou.close()
	ou.conf.maxInserts = 1
	ou.initQueues()

	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Run(func(args mock.Arguments) {
		ou.cancelFunc()
	})

	err := ou.SubmitOperationUpdate(ou.ctx, &OperationUpdate{
		ID: fftypes.NewUUID(),
	})
	assert.NoError(t, err)

	ou.updaterLoop(0)

	mdi.AssertExpectations(t)
}

func TestDoBatchUpdateFailUpdate(t *testing.T) {
	ou := newTestOperationUpdaterNoConcrrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{ID: opID1, Type: fftypes.OpTypeBlockchainInvoke},
	}, nil, nil)
	mdi.On("GetTransactions", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doBatchUpdate(ou.ctx, []*OperationUpdate{
		{ID: opID1, State: fftypes.OpStatusSucceeded},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDoBatchUpdateFailGetTransactions(t *testing.T) {
	ou := newTestOperationUpdaterNoConcrrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{ID: opID1, Type: fftypes.OpTypeBlockchainInvoke},
	}, nil, nil)
	mdi.On("GetTransactions", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doBatchUpdate(ou.ctx, []*OperationUpdate{
		{ID: opID1, State: fftypes.OpStatusSucceeded},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDoBatchUpdateFailGetOperations(t *testing.T) {
	ou := newTestOperationUpdaterNoConcrrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doBatchUpdate(ou.ctx, []*OperationUpdate{
		{ID: opID1, State: fftypes.OpStatusSucceeded},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDoUpdateFailTransactionUpdate(t *testing.T) {
	ou := newTestOperationUpdaterNoConcrrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	txID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", mock.Anything, opID1, fftypes.OpStatusSucceeded, "", mock.Anything).Return(nil)
	mdi.On("UpdateTransaction", mock.Anything, txID1, mock.Anything).Return(fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doUpdate(ou.ctx, &OperationUpdate{
		ID: opID1, State: fftypes.OpStatusSucceeded, BlockchainTXID: "0x12345",
	}, []*fftypes.Operation{
		{ID: opID1, Type: fftypes.OpTypeBlockchainInvoke, Transaction: txID1},
	}, []*fftypes.Transaction{
		{ID: txID1},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDoUpdateFailTransferFailTransferEventInsert(t *testing.T) {
	ou := newTestOperationUpdaterNoConcrrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	txID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", mock.Anything, opID1, fftypes.OpStatusFailed, "", mock.Anything).Return(nil)
	mdi.On("UpdateTransaction", mock.Anything, txID1, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferOpFailed
	})).Return(fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doUpdate(ou.ctx, &OperationUpdate{
		ID: opID1, State: fftypes.OpStatusFailed, BlockchainTXID: "0x12345",
	}, []*fftypes.Operation{
		{ID: opID1, Type: fftypes.OpTypeTokenTransfer, Transaction: txID1, Input: fftypes.JSONObject{
			"localId": fftypes.NewUUID().String(),
			"type":    fftypes.TokenTransferTypeMint,
		}},
	}, []*fftypes.Transaction{
		{ID: txID1},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDoUpdateFailTransferFailApprovalEventInsert(t *testing.T) {
	ou := newTestOperationUpdaterNoConcrrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	txID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("ResolveOperation", mock.Anything, opID1, fftypes.OpStatusFailed, "", mock.Anything).Return(nil)
	mdi.On("UpdateTransaction", mock.Anything, txID1, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeApprovalOpFailed
	})).Return(fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doUpdate(ou.ctx, &OperationUpdate{
		ID: opID1, State: fftypes.OpStatusFailed, BlockchainTXID: "0x12345",
	}, []*fftypes.Operation{
		{ID: opID1, Type: fftypes.OpTypeTokenApproval, Transaction: txID1, Input: fftypes.JSONObject{
			"localId": fftypes.NewUUID().String(),
		}},
	}, []*fftypes.Transaction{
		{ID: txID1},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}
