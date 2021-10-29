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
	"context"
	"fmt"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func addTokenPoolCreateInputs(op *fftypes.Operation, pool *fftypes.TokenPool) {
	op.Input = fftypes.JSONObject{
		"id":        pool.ID.String(),
		"namespace": pool.Namespace,
		"name":      pool.Name,
		"config":    pool.Config,
	}
}

func retrieveTokenPoolCreateInputs(ctx context.Context, op *fftypes.Operation, pool *fftypes.TokenPool) (err error) {
	input := &op.Input
	pool.ID, err = fftypes.ParseUUID(ctx, input.GetString("id"))
	if err != nil {
		return err
	}
	pool.Namespace = input.GetString("namespace")
	pool.Name = input.GetString("name")
	if pool.Namespace == "" || pool.Name == "" {
		return fmt.Errorf("namespace or name missing from inputs")
	}
	pool.Config = input.GetObject("config")
	return nil
}

func (am *assetManager) CreateTokenPool(ctx context.Context, ns string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	if err := am.data.VerifyNamespaceExists(ctx, ns); err != nil {
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

func (am *assetManager) CreateTokenPoolByType(ctx context.Context, ns string, connector string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	if err := am.data.VerifyNamespaceExists(ctx, ns); err != nil {
		return nil, err
	}
	pool.ID = fftypes.NewUUID()
	pool.Namespace = ns
	pool.Connector = connector

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

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: pool.Namespace,
			Type:      fftypes.TransactionTypeTokenPool,
			Signer:    pool.Key,
			Reference: pool.ID,
		},
		Created: fftypes.Now(),
		Status:  fftypes.OpStatusPending,
	}
	tx.Hash = tx.Subject.Hash()
	pool.TX.ID = tx.ID
	pool.TX.Type = tx.Subject.Type

	op := fftypes.NewTXOperation(
		plugin,
		pool.Namespace,
		tx.ID,
		"",
		fftypes.OpTypeTokenCreatePool,
		fftypes.OpStatusPending)
	addTokenPoolCreateInputs(op, pool)

	err = am.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		err = am.database.UpsertTransaction(ctx, tx, false /* should be new, or idempotent replay */)
		if err == nil {
			err = am.database.UpsertOperation(ctx, op, false)
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	return pool, plugin.CreateTokenPool(ctx, op.ID, pool)
}

func (am *assetManager) GetTokenPools(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error) {
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, nil, err
	}
	return am.database.GetTokenPools(ctx, am.scopeNS(ns, filter))
}

func (am *assetManager) GetTokenPoolsByType(ctx context.Context, ns string, connector string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error) {
	if _, err := am.selectTokenPlugin(ctx, connector); err != nil {
		return nil, nil, err
	}
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
	if err := fftypes.ValidateFFNameField(ctx, poolName, "name"); err != nil {
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

func (am *assetManager) ValidateTokenPoolTx(ctx context.Context, pool *fftypes.TokenPool, protocolTxID string) error {
	// TODO: validate that the given token pool was created with the given protocolTxId
	return nil
}
