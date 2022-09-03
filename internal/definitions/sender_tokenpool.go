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

package definitions

import (
	"context"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

func (bm *definitionSender) DefineTokenPool(ctx context.Context, pool *core.TokenPoolAnnouncement, waitConfirm bool) error {
	// Map token connector name -> broadcast name
	if broadcastName, exists := bm.tokenBroadcastNames[pool.Pool.Connector]; exists {
		pool.Pool.Connector = broadcastName
	} else {
		log.L(ctx).Infof("Could not find broadcast name for token connector: %s", pool.Pool.Connector)
		return i18n.NewError(ctx, coremsgs.MsgInvalidConnectorName, broadcastName, "token")
	}

	if bm.multiparty {
		if err := pool.Pool.Validate(ctx); err != nil {
			return err
		}

		pool.Pool.Namespace = ""
		msg, err := bm.sendDefinitionDefault(ctx, pool, core.SystemTagDefinePool, waitConfirm)
		if msg != nil {
			pool.Pool.Message = msg.Header.ID
		}
		pool.Pool.Namespace = bm.namespace
		return err
	}

	return fakeBatch(ctx, func(ctx context.Context, state *core.BatchState) (HandlerResult, error) {
		return bm.handler.handleTokenPoolDefinition(ctx, state, pool.Pool)
	})
}
