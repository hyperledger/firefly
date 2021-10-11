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

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

func (am *assetManager) TokensTransferred(tk tokens.Plugin, transfer *fftypes.TokenTransfer, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	return am.retry.Do(am.ctx, "persist token transfer", func(attempt int) (bool, error) {
		err := am.database.RunAsGroup(am.ctx, func(ctx context.Context) error {
			pool, err := am.database.GetTokenPoolByProtocolID(ctx, transfer.PoolProtocolID)
			if err != nil {
				return err
			}
			if pool == nil {
				log.L(ctx).Warnf("Token transfer received for unknown pool '%s' - ignoring: %s", transfer.PoolProtocolID, protocolTxID)
				return nil
			}
			if err := am.database.UpsertTokenTransfer(ctx, transfer); err != nil {
				log.L(ctx).Errorf("Failed to record token transfer '%s': %s", transfer.ProtocolID, err)
				return err
			}

			balance := &fftypes.TokenBalanceChange{
				PoolProtocolID: transfer.PoolProtocolID,
				TokenIndex:     transfer.TokenIndex,
			}
			if transfer.Type != fftypes.TokenTransferTypeMint {
				balance.Identity = transfer.From
				balance.Amount.Int().Neg(transfer.Amount.Int())
				if err := am.database.AddTokenAccountBalance(ctx, balance); err != nil {
					log.L(ctx).Errorf("Failed to update account '%s' for token transfer '%s': %s", balance.Identity, transfer.ProtocolID, err)
					return err
				}
			}

			if transfer.Type != fftypes.TokenTransferTypeBurn {
				balance.Identity = transfer.To
				balance.Amount.Int().Set(transfer.Amount.Int())
				if err := am.database.AddTokenAccountBalance(ctx, balance); err != nil {
					log.L(ctx).Errorf("Failed to update account '%s for token transfer '%s': %s", balance.Identity, transfer.ProtocolID, err)
					return err
				}
			}

			log.L(ctx).Infof("Token transfer recorded id=%s author=%s", transfer.ProtocolID, signingIdentity)
			event := fftypes.NewEvent(fftypes.EventTypeTransferConfirmed, pool.Namespace, transfer.LocalID)
			return am.database.InsertEvent(ctx, event)
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}
