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

package events

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOperationUpdateSuccess(t *testing.T) {
	em, cancel := newTestEventManagerWithMetrics(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)

	opID := fftypes.NewUUID()
	txid := fftypes.NewUUID()
	info := fftypes.JSONObject{"some": "info"}
	mdi.On("RunAsGroup", em.ctx, mock.Anything).Run(func(args mock.Arguments) {
		args[1].(func(ctx context.Context) error)(em.ctx)
	}).Return(nil)
	mdi.On("GetOperationByID", em.ctx, opID).Return(&fftypes.Operation{ID: opID, Transaction: txid}, nil)
	mdi.On("ResolveOperation", mock.Anything, opID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mth.On("AddBlockchainTX", mock.Anything, txid, "0x12345").Return(nil)

	err := em.OperationUpdate(mdi, opID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	opID := fftypes.NewUUID()
	mdi.On("GetOperationByID", em.ctx, opID).Return(nil, fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	err := em.operationUpdateCtx(em.ctx, opID, fftypes.OpStatusFailed, "", "some error", info)
	assert.NoError(t, err) // swallowed after logging

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	opID := fftypes.NewUUID()
	txid := fftypes.NewUUID()
	info := fftypes.JSONObject{"some": "info"}
	mdi.On("GetOperationByID", em.ctx, opID).Return(&fftypes.Operation{ID: opID, Transaction: txid}, nil)
	mdi.On("ResolveOperation", mock.Anything, opID, fftypes.OpStatusFailed, "some error", info).Return(fmt.Errorf("pop"))

	err := em.operationUpdateCtx(em.ctx, opID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationTXUpdateError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)

	opID := fftypes.NewUUID()
	txid := fftypes.NewUUID()
	info := fftypes.JSONObject{"some": "info"}
	mdi.On("GetOperationByID", em.ctx, opID).Return(&fftypes.Operation{ID: opID, Transaction: txid}, nil)
	mdi.On("ResolveOperation", mock.Anything, opID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mth.On("AddBlockchainTX", mock.Anything, txid, "0x12345").Return(fmt.Errorf("pop"))

	err := em.operationUpdateCtx(em.ctx, opID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateTransferFailBadData(t *testing.T) {
	em, cancel := newTestEventManagerWithMetrics(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)

	op := &fftypes.Operation{
		ID:          fftypes.NewUUID(),
		Type:        fftypes.OpTypeTokenTransfer,
		Namespace:   "ns1",
		Transaction: fftypes.NewUUID(),
	}
	info := fftypes.JSONObject{"some": "info"}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("ResolveOperation", mock.Anything, op.ID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeTransferOpFailed && e.Namespace == "ns1"
	})).Return(nil)
	mth.On("AddBlockchainTX", mock.Anything, op.Transaction, "0x12345").Return(nil)

	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateTransferFail(t *testing.T) {
	em, cancel := newTestEventManagerWithMetrics(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)

	localID := fftypes.NewUUID()
	op := &fftypes.Operation{
		ID:          fftypes.NewUUID(),
		Type:        fftypes.OpTypeTokenTransfer,
		Namespace:   "ns1",
		Transaction: fftypes.NewUUID(),
		Input: fftypes.JSONObject{
			"localId": localID.String(),
			"type":    "transfer",
		},
	}
	info := fftypes.JSONObject{"some": "info"}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("ResolveOperation", mock.Anything, op.ID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeTransferOpFailed && e.Namespace == "ns1" && e.Correlator.Equals(localID)
	})).Return(nil)
	mth.On("AddBlockchainTX", mock.Anything, op.Transaction, "0x12345").Return(nil)

	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateTransferTransactionFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)

	op := &fftypes.Operation{
		ID:          fftypes.NewUUID(),
		Type:        fftypes.OpTypeTokenTransfer,
		Namespace:   "ns1",
		Transaction: fftypes.NewUUID(),
	}
	info := fftypes.JSONObject{"some": "info"}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("ResolveOperation", mock.Anything, op.ID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(nil)
	mth.On("AddBlockchainTX", mock.Anything, op.Transaction, "0x12345").Return(fmt.Errorf("pop"))

	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateTransferEventFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	op := &fftypes.Operation{
		ID:        fftypes.NewUUID(),
		Type:      fftypes.OpTypeTokenTransfer,
		Namespace: "ns1",
	}
	info := fftypes.JSONObject{"some": "info"}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("ResolveOperation", mock.Anything, op.ID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeTransferOpFailed && e.Namespace == "ns1"
	})).Return(fmt.Errorf("pop"))

	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateApprovalFailBadInput(t *testing.T) {
	em, cancel := newTestEventManagerWithMetrics(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)

	op := &fftypes.Operation{
		ID:          fftypes.NewUUID(),
		Type:        fftypes.OpTypeTokenApproval,
		Namespace:   "ns1",
		Transaction: fftypes.NewUUID(),
	}
	info := fftypes.JSONObject{"some": "info"}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("ResolveOperation", mock.Anything, op.ID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeApprovalOpFailed && e.Namespace == "ns1"
	})).Return(nil)
	mth.On("AddBlockchainTX", mock.Anything, op.Transaction, "0x12345").Return(nil)

	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateApprovalFail(t *testing.T) {
	em, cancel := newTestEventManagerWithMetrics(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)

	localID := fftypes.NewUUID()
	op := &fftypes.Operation{
		ID:          fftypes.NewUUID(),
		Type:        fftypes.OpTypeTokenApproval,
		Namespace:   "ns1",
		Transaction: fftypes.NewUUID(),
		Input: fftypes.JSONObject{
			"localId": localID.String(),
		},
	}
	info := fftypes.JSONObject{"some": "info"}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("ResolveOperation", mock.Anything, op.ID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeApprovalOpFailed && e.Namespace == "ns1" && e.Correlator.Equals(localID)
	})).Return(nil)
	mth.On("AddBlockchainTX", mock.Anything, op.Transaction, "0x12345").Return(nil)

	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}
func TestOperationUpdateApprovalTransactionFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}
	mth := em.txHelper.(*txcommonmocks.Helper)

	op := &fftypes.Operation{
		ID:          fftypes.NewUUID(),
		Type:        fftypes.OpTypeTokenApproval,
		Namespace:   "ns1",
		Transaction: fftypes.NewUUID(),
	}
	info := fftypes.JSONObject{"some": "info"}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("ResolveOperation", mock.Anything, op.ID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(nil)
	mth.On("AddBlockchainTX", mock.Anything, op.Transaction, "0x12345").Return(fmt.Errorf("pop"))

	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateApprovalEventFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	op := &fftypes.Operation{
		ID:        fftypes.NewUUID(),
		Type:      fftypes.OpTypeTokenApproval,
		Namespace: "ns1",
	}
	info := fftypes.JSONObject{"some": "info"}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("ResolveOperation", mock.Anything, op.ID, fftypes.OpStatusFailed, "some error", info).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeApprovalOpFailed && e.Namespace == "ns1"
	})).Return(fmt.Errorf("pop"))

	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "0x12345", "some error", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}
