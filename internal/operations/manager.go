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

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type Manager interface {
	StartOperation(ctx context.Context, op *fftypes.Operation) error
}

type operationsManager struct {
	ctx      context.Context
	database database.Plugin
	txHelper txcommon.Helper
	tokens   map[string]tokens.Plugin
}

func NewOperationsManager(ctx context.Context, di database.Plugin, ti map[string]tokens.Plugin) (Manager, error) {
	if di == nil || ti == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	om := &operationsManager{
		ctx:      ctx,
		database: di,
		txHelper: txcommon.NewTransactionHelper(di),
		tokens:   ti,
	}
	return om, nil
}

func (om *operationsManager) StartOperation(ctx context.Context, op *fftypes.Operation) error {
	switch op.Type {
	case fftypes.OpTypeTokenCreatePool:
		pool, err := txcommon.RetrieveTokenPoolCreateInputs(ctx, op)
		if err != nil {
			return err
		}
		return om.createTokenPool(ctx, op.ID, pool)

	case fftypes.OpTypeTokenActivatePool:
		poolID, blockchainInfo, err := txcommon.RetrieveTokenPoolActivateInputs(ctx, op)
		if err != nil {
			return err
		}
		pool, err := om.database.GetTokenPoolByID(ctx, poolID)
		if err != nil {
			return err
		} else if pool == nil {
			return i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		return om.activateTokenPool(ctx, op.ID, pool, blockchainInfo)

	case fftypes.OpTypeTokenTransfer:
		transfer, err := txcommon.RetrieveTokenTransferInputs(ctx, op)
		if err != nil {
			return err
		}
		pool, err := om.database.GetTokenPoolByID(ctx, transfer.Pool)
		if err != nil {
			return err
		} else if pool == nil {
			return i18n.NewError(ctx, i18n.Msg404NotFound)
		}
		return om.transferTokens(ctx, op.ID, pool.ProtocolID, transfer)

	default:
		return i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
}

func (om *operationsManager) selectTokenPlugin(ctx context.Context, name string) (tokens.Plugin, error) {
	for pluginName, plugin := range om.tokens {
		if pluginName == name {
			return plugin, nil
		}
	}
	return nil, i18n.NewError(ctx, i18n.MsgUnknownTokensPlugin, name)
}

func (om *operationsManager) createTokenPool(ctx context.Context, opID *fftypes.UUID, pool *fftypes.TokenPool) error {
	plugin, err := om.selectTokenPlugin(ctx, pool.Connector)
	if err != nil {
		return err
	}
	if complete, err := plugin.CreateTokenPool(ctx, opID, pool); err != nil {
		om.txHelper.WriteOperationFailure(ctx, opID, err)
		return err
	} else if complete {
		om.txHelper.WriteOperationSuccess(ctx, opID, nil)
	}
	return nil
}

func (om *operationsManager) activateTokenPool(ctx context.Context, opID *fftypes.UUID, pool *fftypes.TokenPool, blockchainInfo fftypes.JSONObject) error {
	plugin, err := om.selectTokenPlugin(ctx, pool.Connector)
	if err != nil {
		return err
	}
	if complete, err := plugin.ActivateTokenPool(ctx, opID, pool, blockchainInfo); err != nil {
		om.txHelper.WriteOperationFailure(ctx, opID, err)
		return err
	} else if complete {
		om.txHelper.WriteOperationSuccess(ctx, opID, nil)
	}
	return nil
}

func (om *operationsManager) transferTokens(ctx context.Context, opID *fftypes.UUID, poolProtocolID string, transfer *fftypes.TokenTransfer) error {
	plugin, err := om.selectTokenPlugin(ctx, transfer.Connector)
	if err != nil {
		return err
	}
	switch transfer.Type {
	case fftypes.TokenTransferTypeMint:
		err = plugin.MintTokens(ctx, opID, poolProtocolID, transfer)
	case fftypes.TokenTransferTypeTransfer:
		err = plugin.TransferTokens(ctx, opID, poolProtocolID, transfer)
	case fftypes.TokenTransferTypeBurn:
		err = plugin.BurnTokens(ctx, opID, poolProtocolID, transfer)
	default:
		panic(fmt.Sprintf("unknown transfer type: %v", transfer.Type))
	}
	if err != nil {
		om.txHelper.WriteOperationFailure(ctx, opID, err)
		return err
	}
	return nil
}
