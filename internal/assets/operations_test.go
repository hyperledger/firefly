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

package assets

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestPrepareAndRunCreatePool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeTokenCreatePool,
	}
	pool := &fftypes.TokenPool{
		Connector:  "magic-tokens",
		ProtocolID: "F1",
	}
	err := txcommon.AddTokenPoolCreateInputs(op, pool)
	assert.NoError(t, err)

	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mti.On("CreateTokenPool", context.Background(), op.ID, pool).Return(false, nil)

	po, err := am.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, pool, po.Data.(createPoolData).Pool)

	complete, err := am.RunOperation(context.Background(), po)

	assert.False(t, complete)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
}

func TestPrepareAndRunActivatePool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeTokenActivatePool,
	}
	pool := &fftypes.TokenPool{
		Connector:  "magic-tokens",
		ID:         fftypes.NewUUID(),
		ProtocolID: "F1",
	}
	info := fftypes.JSONObject{
		"some": "info",
	}
	txcommon.AddTokenPoolActivateInputs(op, pool.ID, info)

	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mdi := am.database.(*databasemocks.Plugin)
	mti.On("ActivateTokenPool", context.Background(), op.ID, pool, info).Return(true, nil)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(pool, nil)

	po, err := am.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, pool, po.Data.(activatePoolData).Pool)
	assert.Equal(t, info, po.Data.(activatePoolData).BlockchainInfo)

	complete, err := am.RunOperation(context.Background(), po)

	assert.True(t, complete)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPrepareAndRunTransfer(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeTokenTransfer,
	}
	pool := &fftypes.TokenPool{
		Connector:  "magic-tokens",
		ProtocolID: "F1",
	}
	transfer := &fftypes.TokenTransfer{
		LocalID: fftypes.NewUUID(),
		Pool:    pool.ID,
		Type:    fftypes.TokenTransferTypeTransfer,
	}
	txcommon.AddTokenTransferInputs(op, transfer)

	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mdi := am.database.(*databasemocks.Plugin)
	mti.On("TransferTokens", context.Background(), op.ID, "F1", transfer).Return(nil)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(pool, nil)

	po, err := am.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, pool, po.Data.(transferData).Pool)
	assert.Equal(t, transfer, po.Data.(transferData).Transfer)

	complete, err := am.RunOperation(context.Background(), po)

	assert.False(t, complete)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPrepareAndRunApproval(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		Type: fftypes.OpTypeTokenApproval,
	}
	pool := &fftypes.TokenPool{
		Connector:  "magic-tokens",
		ProtocolID: "F1",
	}
	approval := &fftypes.TokenApproval{
		LocalID:  fftypes.NewUUID(),
		Pool:     pool.ID,
		Approved: true,
	}
	txcommon.AddTokenApprovalInputs(op, approval)

	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mdi := am.database.(*databasemocks.Plugin)
	mti.On("TokensApproval", context.Background(), op.ID, "F1", approval).Return(nil)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(pool, nil)

	po, err := am.PrepareOperation(context.Background(), op)
	assert.NoError(t, err)
	assert.Equal(t, pool, po.Data.(approvalData).Pool)
	assert.Equal(t, approval, po.Data.(approvalData).Approval)

	complete, err := am.RunOperation(context.Background(), po)

	assert.False(t, complete)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPrepareOperationNotSupported(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	po, err := am.PrepareOperation(context.Background(), &fftypes.Operation{})

	assert.Nil(t, po)
	assert.Regexp(t, "FF10349", err)
}

func TestPrepareOperationCreatePoolBadInput(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenCreatePool,
		Input: fftypes.JSONObject{"id": "bad"},
	}

	_, err := am.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10151", err)
}

func TestPrepareOperationActivatePoolBadInput(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenActivatePool,
		Input: fftypes.JSONObject{"id": "bad"},
	}

	_, err := am.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10142", err)
}

func TestPrepareOperationActivatePoolError(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	poolID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenActivatePool,
		Input: fftypes.JSONObject{"id": poolID.String()},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), poolID).Return(nil, fmt.Errorf("pop"))

	_, err := am.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPrepareOperationActivatePoolNotFound(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	poolID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenActivatePool,
		Input: fftypes.JSONObject{"id": poolID.String()},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), poolID).Return(nil, nil)

	_, err := am.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestPrepareOperationTransferBadInput(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenTransfer,
		Input: fftypes.JSONObject{"localId": "bad"},
	}

	_, err := am.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10151", err)
}

func TestPrepareOperationTransferError(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	poolID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenTransfer,
		Input: fftypes.JSONObject{"pool": poolID.String()},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), poolID).Return(nil, fmt.Errorf("pop"))

	_, err := am.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPrepareOperationTransferNotFound(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	poolID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenTransfer,
		Input: fftypes.JSONObject{"pool": poolID.String()},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), poolID).Return(nil, nil)

	_, err := am.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestPrepareOperationApprovalBadInput(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenApproval,
		Input: fftypes.JSONObject{"localId": "bad"},
	}

	_, err := am.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10151", err)
}

func TestPrepareOperationApprovalError(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	poolID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenApproval,
		Input: fftypes.JSONObject{"pool": poolID.String()},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), poolID).Return(nil, fmt.Errorf("pop"))

	_, err := am.PrepareOperation(context.Background(), op)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestPrepareOperationApprovalNotFound(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	poolID := fftypes.NewUUID()
	op := &fftypes.Operation{
		Type:  fftypes.OpTypeTokenApproval,
		Input: fftypes.JSONObject{"pool": poolID.String()},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), poolID).Return(nil, nil)

	_, err := am.PrepareOperation(context.Background(), op)
	assert.Regexp(t, "FF10109", err)

	mdi.AssertExpectations(t)
}

func TestRunOperationNotSupported(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	complete, err := am.RunOperation(context.Background(), &fftypes.PreparedOperation{})

	assert.False(t, complete)
	assert.Regexp(t, "FF10349", err)
}

func TestRunOperationCreatePoolBadPlugin(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{}
	pool := &fftypes.TokenPool{}

	complete, err := am.RunOperation(context.Background(), opCreatePool(op, pool))

	assert.False(t, complete)
	assert.Regexp(t, "FF10272", err)
}

func TestRunOperationCreatePool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
	}
	pool := &fftypes.TokenPool{
		Connector: "magic-tokens",
	}

	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mti.On("CreateTokenPool", context.Background(), op.ID, pool).Return(false, nil)

	complete, err := am.RunOperation(context.Background(), opCreatePool(op, pool))

	assert.False(t, complete)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
}

func TestRunOperationActivatePoolBadPlugin(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{}
	pool := &fftypes.TokenPool{}
	info := fftypes.JSONObject{}

	complete, err := am.RunOperation(context.Background(), opActivatePool(op, pool, info))

	assert.False(t, complete)
	assert.Regexp(t, "FF10272", err)
}

func TestRunOperationTransferBadPlugin(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{}
	pool := &fftypes.TokenPool{}
	transfer := &fftypes.TokenTransfer{}

	complete, err := am.RunOperation(context.Background(), opTransfer(op, pool, transfer))

	assert.False(t, complete)
	assert.Regexp(t, "FF10272", err)
}

func TestRunOperationApprovalBadPlugin(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{}
	pool := &fftypes.TokenPool{}
	approval := &fftypes.TokenApproval{}

	complete, err := am.RunOperation(context.Background(), opApproval(op, pool, approval))

	assert.False(t, complete)
	assert.Regexp(t, "FF10272", err)
}

func TestRunOperationTransferUnknownType(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
	}
	pool := &fftypes.TokenPool{
		Connector: "magic-tokens",
	}
	transfer := &fftypes.TokenTransfer{
		Type: "bad",
	}

	assert.PanicsWithValue(t, "unknown transfer type: bad", func() {
		am.RunOperation(context.Background(), opTransfer(op, pool, transfer))
	})
}

func TestRunOperationTransferMint(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
	}
	pool := &fftypes.TokenPool{
		Connector:  "magic-tokens",
		ProtocolID: "F1",
	}
	transfer := &fftypes.TokenTransfer{
		Type: fftypes.TokenTransferTypeMint,
	}

	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mti.On("MintTokens", context.Background(), op.ID, "F1", transfer).Return(nil)

	complete, err := am.RunOperation(context.Background(), opTransfer(op, pool, transfer))

	assert.False(t, complete)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
}

func TestRunOperationTransferBurn(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
	}
	pool := &fftypes.TokenPool{
		Connector:  "magic-tokens",
		ProtocolID: "F1",
	}
	transfer := &fftypes.TokenTransfer{
		Type: fftypes.TokenTransferTypeBurn,
	}

	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mti.On("BurnTokens", context.Background(), op.ID, "F1", transfer).Return(nil)

	complete, err := am.RunOperation(context.Background(), opTransfer(op, pool, transfer))

	assert.False(t, complete)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
}

func TestRunOperationTransfer(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
	}
	pool := &fftypes.TokenPool{
		Connector:  "magic-tokens",
		ProtocolID: "F1",
	}
	transfer := &fftypes.TokenTransfer{
		Type: fftypes.TokenTransferTypeTransfer,
	}

	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mti.On("TransferTokens", context.Background(), op.ID, "F1", transfer).Return(nil)

	complete, err := am.RunOperation(context.Background(), opTransfer(op, pool, transfer))

	assert.False(t, complete)
	assert.NoError(t, err)

	mti.AssertExpectations(t)
}
