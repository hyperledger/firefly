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

package assets

import (
	"context"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (am *assetManager) CreateTokenPool(ctx context.Context, ns string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	if err := am.data.VerifyNamespaceExists(ctx, ns); err != nil {
		return nil, err
	}
	if err := fftypes.ValidateFFNameFieldNoUUID(ctx, pool.Name, "name"); err != nil {
		return nil, err
	}
	pool.ID = fftypes.NewUUID()
	pool.Namespace = ns

	if pool.Connector == "" {
		connector, err := am.getTokenConnectorName(ctx, ns)
		if err != nil {
			return nil, err
		}
		pool.Connector = connector
	}

	if pool.Key == "" {
		org, err := am.identity.GetLocalOrganization(ctx)
		if err != nil {
			return nil, err
		}
		pool.Key = org.Identity
	}
	return am.createTokenPoolInternal(ctx, pool, waitConfirm)
}

func (am *assetManager) createTokenPoolInternal(ctx context.Context, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	plugin, err := am.selectTokenPlugin(ctx, pool.Connector)
	if err != nil {
		return nil, err
	}

	if waitConfirm {
		return am.syncasync.WaitForTokenPool(ctx, pool.Namespace, pool.ID, func(ctx context.Context) error {
			_, err := am.createTokenPoolInternal(ctx, pool, false)
			return err
		})
	}

	var op *fftypes.Operation
	err = am.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		txid, err := am.txHelper.SubmitNewTransaction(ctx, pool.Namespace, fftypes.TransactionTypeTokenPool)
		if err != nil {
			return err
		}

		pool.TX.ID = txid
		pool.TX.Type = fftypes.TransactionTypeTokenPool

		op = fftypes.NewOperation(
			plugin,
			pool.Namespace,
			txid,
			"",
			fftypes.OpTypeTokenCreatePool)
		txcommon.AddTokenPoolCreateInputs(op, pool)

		return am.database.InsertOperation(ctx, op)
	})
	if err != nil {
		return nil, err
	}

	return pool, plugin.CreateTokenPool(ctx, op.ID, pool)
}

func (am *assetManager) ActivateTokenPool(ctx context.Context, pool *fftypes.TokenPool, event *fftypes.BlockchainEvent) error {
	plugin, err := am.selectTokenPlugin(ctx, pool.Connector)
	if err != nil {
		return err
	}

	op := fftypes.NewOperation(
		plugin,
		pool.Namespace,
		pool.TX.ID,
		"",
		fftypes.OpTypeTokenActivatePool)
	if err := am.database.InsertOperation(ctx, op); err != nil {
		return err
	}

	return plugin.ActivateTokenPool(ctx, op.ID, pool, event)
}

func (am *assetManager) GetTokenPools(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error) {
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, nil, err
	}
	return am.database.GetTokenPools(ctx, am.scopeNS(ns, filter))
}

func (am *assetManager) GetTokenPool(ctx context.Context, ns, connector, poolName string) (*fftypes.TokenPool, error) {
	if _, err := am.selectTokenPlugin(ctx, connector); err != nil {
		return nil, err
	}
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}
	if err := fftypes.ValidateFFNameFieldNoUUID(ctx, poolName, "name"); err != nil {
		return nil, err
	}
	pool, err := am.database.GetTokenPool(ctx, ns, poolName)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	return pool, nil
}

func (am *assetManager) GetTokenPoolByNameOrID(ctx context.Context, ns, poolNameOrID string) (*fftypes.TokenPool, error) {
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}

	var pool *fftypes.TokenPool

	poolID, err := fftypes.ParseUUID(ctx, poolNameOrID)
	if err != nil {
		if err := fftypes.ValidateFFNameField(ctx, poolNameOrID, "name"); err != nil {
			return nil, err
		}
		if pool, err = am.database.GetTokenPool(ctx, ns, poolNameOrID); err != nil {
			return nil, err
		}
	} else if pool, err = am.database.GetTokenPoolByID(ctx, poolID); err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	return pool, nil
}
