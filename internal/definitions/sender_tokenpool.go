// Copyright © 2023 Kaleido, Inc.
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

package definitions

import (
	"context"
	"errors"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

func (ds *definitionSender) PublishTokenPool(ctx context.Context, poolNameOrID, networkName string, waitConfirm bool) (*core.TokenPool, error) {
	if !ds.multiparty {
		return nil, i18n.NewError(ctx, coremsgs.MsgActionNotSupported)
	}
	pool, err := ds.assets.GetTokenPoolByNameOrID(ctx, poolNameOrID)
	if err != nil {
		return nil, err
	}
	if networkName != "" {
		pool.NetworkName = networkName
	}
	err = ds.publishTokenPool(ctx, &core.TokenPoolDefinition{Pool: pool}, waitConfirm)
	if err != nil {
		return nil, err
	}
	if !waitConfirm {
		err = ds.database.UpsertTokenPool(ctx, pool, database.UpsertOptimizationExisting)
	}
	return pool, err
}

func (ds *definitionSender) publishTokenPool(ctx context.Context, pool *core.TokenPoolDefinition, waitConfirm bool) error {
	// Map token connector name -> broadcast name
	if broadcastName, exists := ds.tokenBroadcastNames[pool.Pool.Connector]; exists {
		pool.Pool.Connector = broadcastName
	} else {
		log.L(ctx).Infof("Could not find broadcast name for token connector: %s", pool.Pool.Connector)
		return i18n.NewError(ctx, coremsgs.MsgInvalidConnectorName, broadcastName, "token")
	}

	if err := pool.Pool.Validate(ctx); err != nil {
		return err
	}

	// Prepare the pool definition to be serialized for broadcast
	localName := pool.Pool.Name
	pool.Pool.Name = ""
	pool.Pool.Namespace = ""
	pool.Pool.Published = true
	if pool.Pool.NetworkName == "" {
		pool.Pool.NetworkName = localName
	}

	msg, err := ds.sendDefinitionDefault(ctx, pool, core.SystemTagDefinePool, waitConfirm)
	if msg != nil {
		pool.Pool.Message = msg.Header.ID
	}

	pool.Pool.Name = localName
	pool.Pool.Namespace = ds.namespace
	return err
}

func (ds *definitionSender) DefineTokenPool(ctx context.Context, pool *core.TokenPoolDefinition, waitConfirm bool) error {
	if pool.Pool.Published {
		if !ds.multiparty {
			return i18n.NewError(ctx, coremsgs.MsgActionNotSupported)
		}
		return ds.publishTokenPool(ctx, pool, waitConfirm)
	}

	return fakeBatch(ctx, func(ctx context.Context, state *core.BatchState) (HandlerResult, error) {
		hr, err := ds.handler.handleTokenPoolDefinition(ctx, state, pool.Pool)
		if err != nil {
			if innerErr := errors.Unwrap(err); innerErr != nil {
				return hr, innerErr
			}
		}
		return hr, err
	})
}
