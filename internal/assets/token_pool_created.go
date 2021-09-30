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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

func (am *assetManager) persistTokenPoolTransaction(ctx context.Context, pool *fftypes.TokenPool, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) (valid bool, err error) {
	if pool.ID == nil || pool.TX.ID == nil {
		log.L(ctx).Errorf("Invalid token pool '%s'. Missing ID (%v) or transaction ID (%v)", pool.ID, pool.ID, pool.TX.ID)
		return false, nil // this is not retryable
	}
	return am.txhelper.PersistTransaction(ctx, &fftypes.Transaction{
		ID: pool.TX.ID,
		Subject: fftypes.TransactionSubject{
			Namespace: pool.Namespace,
			Type:      pool.TX.Type,
			Signer:    signingIdentity,
			Reference: pool.ID,
		},
		ProtocolID: protocolTxID,
		Info:       additionalInfo,
	})
}

func (am *assetManager) persistTokenPool(ctx context.Context, pool *fftypes.TokenPool) (valid bool, err error) {
	l := log.L(ctx)
	if err := fftypes.ValidateFFNameField(ctx, pool.Name, "name"); err != nil {
		l.Errorf("Invalid token pool '%s' - invalid name '%s': %a", pool.ID, pool.Name, err)
		return false, nil // This is not retryable
	}
	err = am.database.UpsertTokenPool(ctx, pool)
	if err != nil {
		if err == database.IDMismatch {
			log.L(ctx).Errorf("Invalid token pool '%s'. ID mismatch with existing record", pool.ID)
			return false, nil // This is not retryable
		}
		l.Errorf("Failed to insert token pool '%s': %s", pool.ID, err)
		return false, err // a persistence failure here is considered retryable (so returned)
	}
	return true, nil
}

func (am *assetManager) TokenPoolCreated(tk tokens.Plugin, pool *fftypes.TokenPool, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	return am.retry.Do(am.ctx, "persist token pool", func(attempt int) (bool, error) {
		err := am.database.RunAsGroup(am.ctx, func(ctx context.Context) error {
			valid, err := am.persistTokenPoolTransaction(ctx, pool, signingIdentity, protocolTxID, additionalInfo)
			if valid && err == nil {
				valid, err = am.persistTokenPool(ctx, pool)
			}
			if err != nil {
				return err
			}
			if !valid {
				log.L(ctx).Warnf("Token pool rejected id=%s author=%s", pool.ID, signingIdentity)
				event := fftypes.NewEvent(fftypes.EventTypePoolRejected, pool.Namespace, pool.ID)
				return am.database.InsertEvent(ctx, event)
			}
			log.L(ctx).Infof("Token pool created id=%s author=%s", pool.ID, signingIdentity)
			event := fftypes.NewEvent(fftypes.EventTypePoolConfirmed, pool.Namespace, pool.ID)
			return am.database.InsertEvent(ctx, event)
		})
		return err != nil, err // retry indefinitely (until context closes)
	})
}
