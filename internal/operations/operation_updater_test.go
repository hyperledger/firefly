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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockplug struct{}

func (mp *mockplug) Name() string {
	return "unittest"
}

func newTestOperationUpdater(t *testing.T) *operationUpdater {
	return newTestOperationUpdaterCommon(t, &database.Capabilities{Concurrency: true})
}

func newTestOperationUpdaterNoConcurrency(t *testing.T) *operationUpdater {
	return newTestOperationUpdaterCommon(t, &database.Capabilities{Concurrency: false})
}

func newTestOperationUpdaterCommon(t *testing.T, dbCapabilities *database.Capabilities) *operationUpdater {
	coreconfig.Reset()
	config.Set(coreconfig.OpUpdateWorkerCount, 1)
	config.Set(coreconfig.OpUpdateWorkerBatchTimeout, "1s")
	config.Set(coreconfig.OpUpdateWorkerBatchMaxInserts, 200)
	logrus.SetLevel(logrus.DebugLevel)

	mom := &operationsManager{
		namespace: "ns1",
		handlers:  make(map[fftypes.FFEnum]OperationHandler),
	}
	mdi := &databasemocks.Plugin{}
	mdi.On("Capabilities").Return(dbCapabilities)
	mdm := &datamocks.Manager{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	return newOperationUpdater(context.Background(), mom, mdi, txHelper)
}

func TestNewOperationUpdaterNoConcurrency(t *testing.T) {
	ou := newTestOperationUpdaterNoConcurrency(t)
	defer ou.close()
	assert.Zero(t, ou.conf.workerCount)
}

func TestSubmitUpdateBadIDIgnored(t *testing.T) {
	ou := newTestOperationUpdater(t)
	ou.close()
	ou.workQueues = []chan *OperationUpdate{
		make(chan *OperationUpdate),
	}
	ou.cancelFunc()
	ou.SubmitOperationUpdate(ou.ctx, &OperationUpdate{
		NamespacedOpID: "!!!" + fftypes.NewUUID().String(),
	})
}

func TestSubmitUpdateClosed(t *testing.T) {
	ou := newTestOperationUpdater(t)
	ou.close()
	ou.workQueues = []chan *OperationUpdate{
		make(chan *OperationUpdate),
	}
	ou.cancelFunc()
	ou.SubmitOperationUpdate(ou.ctx, &OperationUpdate{
		NamespacedOpID: "ns1:" + fftypes.NewUUID().String(),
	})
}

func TestSubmitUpdateSyncFallbackOpNotFound(t *testing.T) {
	ou := newTestOperationUpdaterNoConcurrency(t)
	defer ou.close()
	customCtx := context.WithValue(context.Background(), "dbtx", "on this context")

	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", customCtx, mock.Anything).Run(func(args mock.Arguments) {
		err := args[1].(func(context.Context) error)(customCtx)
		assert.NoError(t, err)
	}).Return(nil)
	mdi.On("GetOperations", customCtx, mock.Anything, mock.Anything).Return(nil, nil, nil)

	ou.SubmitOperationUpdate(customCtx, &OperationUpdate{
		NamespacedOpID: "ns1:" + fftypes.NewUUID().String(),
	})

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
	tx1 := &core.Transaction{ID: fftypes.NewUUID()}

	done := make(chan struct{})

	mdi := om.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything, mock.Anything).Return([]*core.Operation{
		{ID: opID1, Namespace: "ns1", Type: core.OpTypeBlockchainInvoke, Transaction: tx1.ID},
		{ID: opID2, Namespace: "ns1", Type: core.OpTypeTokenTransfer, Input: fftypes.JSONObject{"test": "test"}},
		{ID: opID3, Namespace: "ns1", Type: core.OpTypeTokenApproval, Input: fftypes.JSONObject{"test": "test"}},
	}, nil, nil)
	mdi.On("GetTransactions", mock.Anything, mock.Anything, mock.Anything).Return([]*core.Transaction{tx1}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, "ns1", opID1, core.OpStatusSucceeded, mock.Anything, fftypes.JSONObject(nil)).Return(nil)
	mdi.On("UpdateTransaction", mock.Anything, tx1.ID, mock.Anything).Return(nil)
	err1Str := "err1"
	mdi.On("ResolveOperation", mock.Anything, "ns1", opID2, core.OpStatusFailed, &err1Str, fftypes.JSONObject{"test": true}).Return(nil)
	err2Str := "err2"
	mdi.On("ResolveOperation", mock.Anything, "ns1", opID3, core.OpStatusFailed, &err2Str, fftypes.JSONObject(nil)).Return(nil).
		Run(func(args mock.Arguments) {
			close(done)
		})

	om.Start()

	om.SubmitOperationUpdate(&mockplug{}, &OperationUpdate{
		NamespacedOpID: "ns1:" + opID1.String(),
		Status:         core.OpStatusSucceeded,
		BlockchainTXID: "tx12345",
	})
	om.SubmitOperationUpdate(&mockplug{}, &OperationUpdate{
		NamespacedOpID: "ns1:" + opID2.String(),
		Status:         core.OpStatusFailed,
		ErrorMessage:   "err1",
		Output:         fftypes.JSONObject{"test": true},
	})
	om.SubmitOperationUpdate(&mockplug{}, &OperationUpdate{
		NamespacedOpID: "ns1:" + opID3.String(),
		Status:         core.OpStatusFailed,
		ErrorMessage:   "err2",
	})

	<-done

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

	ou.SubmitOperationUpdate(ou.ctx, &OperationUpdate{
		NamespacedOpID: "ns1:" + fftypes.NewUUID().String(),
	})

	ou.updaterLoop(0)

	mdi.AssertExpectations(t)
}

func TestDoBatchUpdateIgnoreBadID(t *testing.T) {
	ou := newTestOperationUpdaterNoConcurrency(t)
	defer ou.close()

	ou.initQueues()

	err := ou.doBatchUpdate(ou.ctx, []*OperationUpdate{
		{NamespacedOpID: "!!Bad", Status: core.OpStatusSucceeded},
	})
	assert.NoError(t, err)

}

func TestDoBatchUpdateFailUpdate(t *testing.T) {
	ou := newTestOperationUpdaterNoConcurrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything, mock.Anything).Return([]*core.Operation{
		{ID: opID1, Namespace: "ns1", Type: core.OpTypeBlockchainInvoke},
	}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, "ns1", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doBatchUpdate(ou.ctx, []*OperationUpdate{
		{NamespacedOpID: "ns1:" + opID1.String(), Status: core.OpStatusSucceeded},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDoBatchUpdateFailGetTransactions(t *testing.T) {
	ou := newTestOperationUpdaterNoConcurrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything, mock.Anything).Return([]*core.Operation{
		{Namespace: "ns1", ID: opID1, Type: core.OpTypeBlockchainInvoke, Transaction: fftypes.NewUUID()},
	}, nil, nil)
	mdi.On("GetTransactions", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doBatchUpdate(ou.ctx, []*OperationUpdate{
		{NamespacedOpID: "ns1:" + opID1.String(), Status: core.OpStatusSucceeded},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDoBatchUpdateFailGetOperations(t *testing.T) {
	ou := newTestOperationUpdaterNoConcurrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doBatchUpdate(ou.ctx, []*OperationUpdate{
		{NamespacedOpID: "ns1:" + opID1.String(), Status: core.OpStatusSucceeded},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDoUpdateIgnoreBadID(t *testing.T) {
	ou := newTestOperationUpdaterNoConcurrency(t)
	defer ou.close()

	ou.initQueues()

	err := ou.doUpdate(ou.ctx, &OperationUpdate{
		NamespacedOpID: "!!!bad", Status: core.OpStatusSucceeded, BlockchainTXID: "0x12345",
	}, []*core.Operation{}, []*core.Transaction{})
	assert.NoError(t, err)

}

func TestDoUpdateFailTransactionUpdate(t *testing.T) {
	ou := newTestOperationUpdaterNoConcurrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	txID1 := fftypes.NewUUID()
	mdi := ou.database.(*databasemocks.Plugin)
	mdi.On("UpdateTransaction", mock.Anything, txID1, mock.Anything).Return(fmt.Errorf("pop"))

	ou.initQueues()

	err := ou.doUpdate(ou.ctx, &OperationUpdate{
		NamespacedOpID: "ns1:" + opID1.String(), Status: core.OpStatusSucceeded, BlockchainTXID: "0x12345",
	}, []*core.Operation{
		{Namespace: "ns1", ID: opID1, Type: core.OpTypeBlockchainInvoke, Transaction: txID1},
	}, []*core.Transaction{
		{ID: txID1},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestDoUpdateFailExternalHandler(t *testing.T) {
	ou := newTestOperationUpdaterNoConcurrency(t)
	defer ou.close()

	opID1 := fftypes.NewUUID()
	txID1 := fftypes.NewUUID()
	ou.manager.handlers[core.OpTypeBlockchainInvoke] = &mockHandler{UpdateErr: fmt.Errorf("pop")}

	ou.initQueues()

	err := ou.doUpdate(ou.ctx, &OperationUpdate{
		NamespacedOpID: "ns1:" + opID1.String(), Status: core.OpStatusSucceeded,
	}, []*core.Operation{
		{Namespace: "ns1", ID: opID1, Type: core.OpTypeBlockchainInvoke, Transaction: txID1},
	}, []*core.Transaction{})
	assert.Regexp(t, "pop", err)
}
