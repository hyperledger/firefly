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

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (dh *definitionHandlers) persistTokenPool(ctx context.Context, announce *fftypes.TokenPoolAnnouncement) (valid bool, err error) {
	pool := announce.Pool

	// Create the pool in pending state
	pool.State = fftypes.TokenPoolStatePending
	err = dh.database.UpsertTokenPool(ctx, pool)
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

func (dh *definitionHandlers) handleTokenPoolBroadcast(ctx context.Context, state DefinitionBatchState, msg *fftypes.Message, data []*fftypes.Data) (HandlerResult, error) {
	var announce fftypes.TokenPoolAnnouncement
	if valid := dh.getSystemBroadcastPayload(ctx, msg, data, &announce); !valid {
		return HandlerResult{Action: ActionReject}, nil
	}

	pool := announce.Pool
	pool.Message = msg.Header.ID

	// Set an event correlator, so that if we reject then the sync-async bridge action can know
	// from the event (without downloading and parsing the msg)
	correlator := pool.ID

	if err := pool.Validate(ctx); err != nil {
		log.L(ctx).Warnf("Token pool '%s' rejected - validate failed: %s", pool.ID, err)
		return HandlerResult{Action: ActionReject, CustomCorrelator: correlator}, nil
	}

	// Check if pool has already been confirmed on chain (and confirm the message if so)
	if existingPool, err := dh.database.GetTokenPoolByID(ctx, pool.ID); err != nil {
		return HandlerResult{Action: ActionRetry}, err
	} else if existingPool != nil && existingPool.State == fftypes.TokenPoolStateConfirmed {
		return HandlerResult{Action: ActionConfirm, CustomCorrelator: correlator}, nil
	}

	if valid, err := dh.persistTokenPool(ctx, &announce); err != nil {
		return HandlerResult{Action: ActionRetry}, err
	} else if !valid {
		return HandlerResult{Action: ActionReject, CustomCorrelator: correlator}, nil
	}

	// Message will remain unconfirmed, but plugin will be notified to activate the pool
	// This will ultimately trigger a pool creation event and a rewind
	state.AddPreFinalize(func(ctx context.Context) error {
		if err := dh.assets.ActivateTokenPool(ctx, pool, announce.Event); err != nil {
			log.L(ctx).Errorf("Failed to activate token pool '%s': %s", pool.ID, err)
			return err
		}
		return nil
	})
	return HandlerResult{Action: ActionWait, CustomCorrelator: correlator}, nil
}
