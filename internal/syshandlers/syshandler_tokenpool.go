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

package syshandlers

import (
	"context"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (sh *systemHandlers) confirmPoolAnnounceOp(ctx context.Context, pool *fftypes.TokenPool) error {
	// Find a matching operation within this transaction
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", pool.TX.ID),
		fb.Eq("type", fftypes.OpTypeTokenAnnouncePool),
	)
	if operations, _, err := sh.database.GetOperations(ctx, filter); err != nil {
		return err
	} else if len(operations) > 0 {
		op := operations[0]
		update := database.OperationQueryFactory.NewUpdate(ctx).
			Set("status", fftypes.OpStatusSucceeded).
			Set("output", fftypes.JSONObject{"message": pool.Message})
		if err := sh.database.UpdateOperation(ctx, op.ID, update); err != nil {
			return err
		}
	}
	return nil
}

func (sh *systemHandlers) persistTokenPool(ctx context.Context, announce *fftypes.TokenPoolAnnouncement) (valid bool, err error) {
	pool := announce.Pool

	// Mark announce operation (if any) completed
	if err := sh.confirmPoolAnnounceOp(ctx, pool); err != nil {
		return false, err // retryable
	}

	// Verify pool has not already been created
	if existingPool, err := sh.database.GetTokenPoolByID(ctx, pool.ID); err != nil {
		return false, err // retryable
	} else if existingPool != nil {
		log.L(ctx).Warnf("Token pool '%s' already exists - ignoring", pool.ID)
		return false, nil // not retryable
	}

	// Create the pool in unconfirmed state
	pool.State = fftypes.TokenPoolStatePending
	err = sh.database.UpsertTokenPool(ctx, pool)
	if err != nil {
		if err == database.IDMismatch {
			log.L(ctx).Errorf("Invalid token pool '%s'. ID mismatch with existing record", pool.ID)
			return false, nil // not retryable
		}
		log.L(ctx).Errorf("Failed to insert token pool '%s': %s", pool.ID, err)
		return false, err // retryable
	}

	return true, nil
}

func (sh *systemHandlers) handleTokenPoolBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	var announce fftypes.TokenPoolAnnouncement
	if valid = sh.getSystemBroadcastPayload(ctx, msg, data, &announce); !valid {
		return false, nil // not retryable
	}

	pool := announce.Pool
	pool.Message = msg.Header.ID
	if err = pool.Validate(ctx); err != nil {
		log.L(ctx).Warnf("Token pool '%s' rejected - validate failed: %s", pool.ID, err)
		err = nil // not retryable
		valid = false
	} else {
		valid, err = sh.persistTokenPool(ctx, &announce) // only returns retryable errors
		if err != nil {
			log.L(ctx).Warnf("Token pool '%s' rejected - failed to write: %s", pool.ID, err)
		}
	}

	if err != nil {
		return false, err
	} else if !valid {
		event := fftypes.NewEvent(fftypes.EventTypePoolRejected, pool.Namespace, pool.ID)
		err = sh.database.InsertEvent(ctx, event)
		return err != nil, err
	}

	if err = sh.assets.ActivateTokenPool(ctx, pool, announce.TX); err != nil {
		log.L(ctx).Errorf("Failed to activate token pool '%s': %s", pool.ID, err)
		return false, err // retryable
	}
	return true, nil
}
