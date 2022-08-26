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
	"github.com/hyperledger/firefly/pkg/database"
)

func (dh *definitionHandler) handleTokenPoolBroadcast(ctx context.Context, state *core.BatchState, msg *core.Message, data core.DataArray) (HandlerResult, error) {
	var announce core.TokenPoolAnnouncement
	if valid := dh.getSystemBroadcastPayload(ctx, msg, data, &announce); !valid {
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedBadPayload, "token pool", msg.Header.ID)
	}

	pool := announce.Pool
	// Map remote connector name -> local name
	if localName, ok := dh.tokenNames[pool.Connector]; ok {
		pool.Connector = localName
	} else {
		log.L(ctx).Infof("Could not find local name for token connector: %s", pool.Connector)
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgInvalidConnectorName, pool.Connector, "token")
	}

	pool.Message = msg.Header.ID
	return dh.handleTokenPoolDefinition(ctx, state, pool)
}

func (dh *definitionHandler) handleTokenPoolDefinition(ctx context.Context, state *core.BatchState, pool *core.TokenPool) (HandlerResult, error) {
	// Set an event correlator, so that if we reject then the sync-async bridge action can know
	// from the event (without downloading and parsing the msg)
	correlator := pool.ID

	pool.Namespace = dh.namespace.Name
	if err := pool.Validate(ctx); err != nil {
		return HandlerResult{Action: ActionReject, CustomCorrelator: correlator}, i18n.NewError(ctx, coremsgs.MsgDefRejectedValidateFail, "token pool", pool.ID, err)
	}

	// Check if pool has already been confirmed on chain (and confirm the message if so)
	if existingPool, err := dh.database.GetTokenPoolByID(ctx, dh.namespace.Name, pool.ID); err != nil {
		return HandlerResult{Action: ActionRetry}, err
	} else if existingPool != nil && existingPool.State == core.TokenPoolStateConfirmed {
		return HandlerResult{Action: ActionConfirm, CustomCorrelator: correlator}, nil
	}

	// Create the pool in pending state
	pool.State = core.TokenPoolStatePending
	if err := dh.database.UpsertTokenPool(ctx, pool); err != nil {
		if err == database.IDMismatch {
			return HandlerResult{Action: ActionReject, CustomCorrelator: correlator}, i18n.NewError(ctx, coremsgs.MsgDefRejectedIDMismatch, "token pool", pool.ID)
		}
		return HandlerResult{Action: ActionRetry}, err
	}

	// Message will remain unconfirmed, but plugin will be notified to activate the pool
	// This will ultimately trigger a pool creation event and a rewind
	state.AddPreFinalize(func(ctx context.Context) error {
		if err := dh.assets.ActivateTokenPool(ctx, pool); err != nil {
			log.L(ctx).Errorf("Failed to activate token pool '%s': %s", pool.ID, err)
			return err
		}
		return nil
	})
	return HandlerResult{Action: ActionWait, CustomCorrelator: correlator}, nil
}
