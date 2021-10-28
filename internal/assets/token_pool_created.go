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

func (am *assetManager) TokenPoolCreated(tk tokens.Plugin, pool *fftypes.TokenPool, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	announce := &fftypes.TokenPoolAnnouncement{
		TokenPool:    *pool,
		ProtocolTxID: protocolTxID,
	}
	pool = &announce.TokenPool

	var valid bool
	err := am.retry.Do(am.ctx, "persist token pool transaction", func(attempt int) (bool, error) {
		err := am.database.RunAsGroup(am.ctx, func(ctx context.Context) error {
			// Find a matching operation within this transaction
			fb := database.OperationQueryFactory.NewFilter(ctx)
			filter := fb.And(
				fb.Eq("tx", pool.TX.ID),
				fb.Eq("type", fftypes.OpTypeTokenCreatePool),
			)
			operations, _, err := am.database.GetOperations(ctx, filter)
			if err != nil || len(operations) == 0 {
				log.L(ctx).Debugf("Token pool transaction '%s' ignored, as it did not match an operation submitted by this node", pool.TX.ID)
				return nil
			}

			err = retrieveTokenPoolCreateInputs(ctx, operations[0], pool)
			if err != nil {
				log.L(ctx).Errorf("Error retrieving pool info from transaction '%s' (%s) - ignoring: %v", pool.TX.ID, err, operations[0].Input)
				return nil
			}

			// Update the transaction with the info received (but leave transaction as "pending").
			// At this point we are the only node in the network that knows about this transaction object.
			// Our local token connector has performed whatever actions it needs to perform, to give us
			// enough information to distribute to all other token connectors in the network.
			// (e.g. details of a newly created token instance or an existing one)
			transaction := &fftypes.Transaction{
				ID:     pool.TX.ID,
				Status: fftypes.OpStatusPending,
				Subject: fftypes.TransactionSubject{
					Namespace: pool.Namespace,
					Type:      pool.TX.Type,
					Signer:    pool.Key,
					Reference: pool.ID,
				},
				ProtocolID: protocolTxID,
				Info:       additionalInfo,
			}

			// Add a new operation for the announcement
			op := fftypes.NewTXOperation(
				tk,
				pool.Namespace,
				transaction.ID,
				"",
				fftypes.OpTypeTokenAnnouncePool,
				fftypes.OpStatusPending)

			valid, err = am.txhelper.PersistTransaction(ctx, transaction)
			if valid && err == nil {
				err = am.database.UpsertOperation(ctx, op, false)
			}
			return err
		})
		return err != nil, err
	})

	if !valid || err != nil {
		return err
	}

	// Announce the details of the new token pool
	_, err = am.broadcast.BroadcastTokenPool(am.ctx, pool.Namespace, announce, false)
	return err
}
