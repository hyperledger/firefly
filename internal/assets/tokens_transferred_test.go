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

package assets

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTokensTransferredAddBalanceSucceedWithRetries(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:           fftypes.TokenTransferTypeTransfer,
		PoolProtocolID: "F1",
		TokenIndex:     "0",
		From:           "0x1",
		To:             "0x2",
		Amount:         *big.NewInt(1),
	}
	fromBalance := &fftypes.TokenBalanceChange{
		PoolProtocolID: "F1",
		TokenIndex:     "0",
		Identity:       "0x1",
		Amount:         *big.NewInt(-1),
	}
	toBalance := &fftypes.TokenBalanceChange{
		PoolProtocolID: "F1",
		TokenIndex:     "0",
		Identity:       "0x2",
		Amount:         *big.NewInt(1),
	}
	pool := &fftypes.TokenPool{
		Namespace: "ns1",
	}

	mdi.On("GetTokenPoolByProtocolID", am.ctx, "F1").Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("GetTokenPoolByProtocolID", am.ctx, "F1").Return(pool, nil).Times(4)
	mdi.On("UpsertTokenTransfer", am.ctx, transfer).Return(fmt.Errorf("pop")).Once()
	mdi.On("UpsertTokenTransfer", am.ctx, transfer).Return(nil).Times(3)
	mdi.On("AddTokenAccountBalance", am.ctx, fromBalance).Return(fmt.Errorf("pop")).Once()
	mdi.On("AddTokenAccountBalance", am.ctx, fromBalance).Return(nil).Times(2)
	mdi.On("AddTokenAccountBalance", am.ctx, toBalance).Return(fmt.Errorf("pop")).Once()
	mdi.On("AddTokenAccountBalance", am.ctx, toBalance).Return(nil).Once()
	mdi.On("InsertEvent", am.ctx, mock.MatchedBy(func(ev *fftypes.Event) bool {
		return ev.Type == fftypes.EventTypeTransferConfirmed && ev.Reference == pool.ID && ev.Namespace == pool.Namespace
	})).Return(nil).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokensTransferred(mti, transfer, "0x12345", "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestTokensTransferredAddBalanceIgnore(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()
	mdi := am.database.(*databasemocks.Plugin)
	mti := &tokenmocks.Plugin{}

	transfer := &fftypes.TokenTransfer{
		Type:           fftypes.TokenTransferTypeTransfer,
		PoolProtocolID: "F1",
		TokenIndex:     "0",
		From:           "0x1",
		To:             "0x2",
		Amount:         *big.NewInt(1),
	}

	mdi.On("GetTokenPoolByProtocolID", am.ctx, "F1").Return(nil, nil)

	info := fftypes.JSONObject{"some": "info"}
	err := am.TokensTransferred(mti, transfer, "0x12345", "tx1", info)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mti.AssertExpectations(t)
}
