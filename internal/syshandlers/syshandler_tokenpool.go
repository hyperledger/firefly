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

	// Create the pool in pending state
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

func (sh *systemHandlers) rejectPool(ctx context.Context, pool *fftypes.TokenPool) error {
	event := fftypes.NewEvent(fftypes.EventTypePoolRejected, pool.Namespace, pool.ID)
	err := sh.database.InsertEvent(ctx, event)
	return err
}

func (sh *systemHandlers) handleTokenPoolBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (SystemBroadcastAction, error) {
	var announce fftypes.TokenPoolAnnouncement
	if valid := sh.getSystemBroadcastPayload(ctx, msg, data, &announce); !valid {
		return ActionReject, nil
	}

	pool := announce.Pool
	pool.Message = msg.Header.ID

	if err := pool.Validate(ctx); err != nil {
		log.L(ctx).Warnf("Token pool '%s' rejected - validate failed: %s", pool.ID, err)
		return ActionReject, sh.rejectPool(ctx, pool)
	}

	// Check if pool has already been confirmed on chain (and confirm the message if so)
	if existingPool, err := sh.database.GetTokenPoolByID(ctx, pool.ID); err != nil {
		return ActionRetry, err
	} else if existingPool != nil && existingPool.State == fftypes.TokenPoolStateConfirmed {
		return ActionConfirm, nil
	}

	if valid, err := sh.persistTokenPool(ctx, &announce); err != nil {
		return ActionRetry, err
	} else if !valid {
		return ActionReject, sh.rejectPool(ctx, pool)
	}

	if err := sh.assets.ActivateTokenPool(ctx, pool, announce.TX); err != nil {
		log.L(ctx).Errorf("Failed to activate token pool '%s': %s", pool.ID, err)
		return ActionRetry, err
	}

	// Message will remain unconfirmed until pool confirmation triggers a rewind
	return ActionWait, nil
}
