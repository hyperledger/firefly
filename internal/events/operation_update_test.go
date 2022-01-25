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
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOperationUpdateSuccess(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	opID := fftypes.NewUUID()
	txid := fftypes.NewUUID()
	mdi.On("RunAsGroup", em.ctx, mock.Anything).Run(func(args mock.Arguments) {
		args[1].(func(ctx context.Context) error)(em.ctx)
	}).Return(nil)
	mdi.On("GetOperationByID", em.ctx, opID).Return(&fftypes.Operation{ID: opID}, nil)
	mdi.On("UpdateOperation", em.ctx, opID, mock.Anything).Return(nil)
	mdi.On("GetTransactionByID", em.ctx, mock.Anything).Return(&fftypes.Transaction{ID: txid}, nil)
	mdi.On("UpsertTransaction", em.ctx, mock.Anything).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.OperationUpdate(mdi, opID, fftypes.OpStatusFailed, "", "some error", info)
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
	mdi.On("GetOperationByID", em.ctx, opID).Return(&fftypes.Operation{ID: opID}, nil)
	mdi.On("UpdateOperation", em.ctx, opID, mock.Anything).Return(fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	err := em.operationUpdateCtx(em.ctx, opID, fftypes.OpStatusFailed, "", "some error", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateTransferFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	op := &fftypes.Operation{
		ID:        fftypes.NewUUID(),
		Type:      fftypes.OpTypeTokenTransfer,
		Namespace: "ns1",
	}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("UpdateOperation", em.ctx, op.ID, mock.Anything).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeTransferOpFailed && e.Namespace == "ns1"
	})).Return(nil)
	mdi.On("GetTransactionByID", em.ctx, mock.Anything).Return(&fftypes.Transaction{ID: fftypes.NewUUID()}, nil)
	mdi.On("UpsertTransaction", em.ctx, mock.Anything).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "", "some error", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestOperationUpdateTransferTransactionFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	op := &fftypes.Operation{
		ID:        fftypes.NewUUID(),
		Type:      fftypes.OpTypeTokenTransfer,
		Namespace: "ns1",
	}

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("UpdateOperation", em.ctx, op.ID, mock.Anything).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(nil)
	mdi.On("GetTransactionByID", em.ctx, mock.Anything).Return(&fftypes.Transaction{ID: fftypes.NewUUID()}, nil)
	mdi.On("UpsertTransaction", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "", "some error", info)
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

	mdi.On("GetOperationByID", em.ctx, op.ID).Return(op, nil)
	mdi.On("UpdateOperation", em.ctx, op.ID, mock.Anything).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.MatchedBy(func(e *fftypes.Event) bool {
		return e.Type == fftypes.EventTypeTransferOpFailed && e.Namespace == "ns1"
	})).Return(fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	err := em.operationUpdateCtx(em.ctx, op.ID, fftypes.OpStatusFailed, "", "some error", info)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mbi.AssertExpectations(t)
}
