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

func (ds *definitionSender) PublishTokenPool(ctx context.Context, poolNameOrID, networkName string, waitConfirm bool) (pool *core.TokenPool, err error) {
	if !ds.multiparty {
		return nil, i18n.NewError(ctx, coremsgs.MsgActionNotSupported)
	}

	var sender *sendWrapper
	err = ds.database.RunAsGroup(ctx, func(ctx context.Context) error {
		if pool, err = ds.assets.GetTokenPoolByNameOrID(ctx, poolNameOrID); err != nil {
			return err
		}
		if networkName != "" {
			pool.NetworkName = networkName
		}

		sender = ds.getTokenPoolSender(ctx, &core.TokenPoolDefinition{Pool: pool})
		if sender.err != nil {
			return sender.err
		}
		if !waitConfirm {
			if err = sender.sender.Prepare(ctx); err != nil {
				return err
			}
			if err = ds.database.UpsertTokenPool(ctx, pool, database.UpsertOptimizationExisting); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	_, err = sender.send(ctx, waitConfirm)
	return pool, err
}

func (ds *definitionSender) getTokenPoolSender(ctx context.Context, pool *core.TokenPoolDefinition) *sendWrapper {
	// Map token connector name -> broadcast name
	if broadcastName, exists := ds.tokenBroadcastNames[pool.Pool.Connector]; exists {
		pool.Pool.Connector = broadcastName
	} else {
		log.L(ctx).Infof("Could not find broadcast name for token connector: %s", pool.Pool.Connector)
		return wrapSendError(i18n.NewError(ctx, coremsgs.MsgInvalidConnectorName, broadcastName, "token"))
	}

	if err := pool.Pool.Validate(ctx); err != nil {
		return wrapSendError(err)
	}

	// Prepare the pool definition to be serialized for broadcast
	localName := pool.Pool.Name
	pool.Pool.Name = ""
	pool.Pool.Namespace = ""
	pool.Pool.Published = true
	if pool.Pool.NetworkName == "" {
		pool.Pool.NetworkName = localName
	}

	sender := ds.getSenderDefault(ctx, pool, core.SystemTagDefinePool)
	if sender.message != nil {
		pool.Pool.Message = sender.message.Header.ID
	}

	pool.Pool.Name = localName
	pool.Pool.Namespace = ds.namespace
	return sender
}

func (ds *definitionSender) DefineTokenPool(ctx context.Context, pool *core.TokenPoolDefinition, waitConfirm bool) error {
	if pool.Pool.Published {
		if !ds.multiparty {
			return i18n.NewError(ctx, coremsgs.MsgActionNotSupported)
		}
		_, err := ds.getTokenPoolSender(ctx, pool).send(ctx, waitConfirm)
		return err
	}

	pool.Pool.NetworkName = ""

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
