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

package events

import (
	"context"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

func (em *eventManager) loadApprovalOperation(ctx context.Context, tx *fftypes.UUID, approval *fftypes.TokenApproval) error {
	approval.LocalID = nil

	// find a matching operation within the transaction
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", tx),
		fb.Eq("type", fftypes.OpTypeTokenApproval),
	)
	operations, _, err := em.database.GetOperations(ctx, filter)
	if err != nil {
		return err
	}
	if len(operations) > 0 {
		if origApproval, err := txcommon.RetrieveTokenApprovalInputs(ctx, operations[0]); err != nil {
			log.L(ctx).Warnf("Failed to read operation inputs for token approval '%s': %s", approval.ProtocolID, err)
		} else if origApproval != nil {
			approval.LocalID = origApproval.LocalID
		}
	}

	if approval.LocalID == nil {
		approval.LocalID = fftypes.NewUUID()
	}
	return nil
}

func (em *eventManager) persistTokenApproval(ctx context.Context, approval *tokens.TokenApproval) (valid bool, err error) {
	pool, err := em.database.GetTokenPoolByProtocolID(ctx, approval.Connector, approval.PoolProtocolID)
	if err != nil {
		return false, err
	}
	if pool == nil {
		log.L(ctx).Infof("Token approval received for unknown pool '%s' - ignoring: %s", approval.PoolProtocolID, approval.Event.ProtocolID)
		return false, nil
	}
	approval.Namespace = pool.Namespace
	approval.Pool = pool.ID

	if approval.TX.ID != nil {
		if err := em.loadApprovalOperation(ctx, approval.TX.ID, &approval.TokenApproval); err != nil {
			return false, err
		}

		if valid, err := em.txHelper.PersistTransaction(ctx, approval.Namespace, approval.TX.ID, approval.TX.Type, approval.Event.BlockchainTXID); err != nil || !valid {
			return valid, err
		}

		if existing, err := em.database.GetTokenApproval(ctx, approval.LocalID); err != nil {
			return false, err
		} else if existing != nil {
			approval.LocalID = fftypes.NewUUID()
		}
	} else {
		approval.LocalID = fftypes.NewUUID()
	}

	chainEvent := buildBlockchainEvent(approval.Namespace, nil, &approval.Event, &approval.TX)
	approval.BlockchainEvent = chainEvent.ID
	if err := em.persistBlockchainEvent(ctx, chainEvent); err != nil {
		return false, err
	}

	if err := em.database.UpsertTokenApproval(ctx, &approval.TokenApproval); err != nil {
		log.L(ctx).Errorf("Failed to record token approval '%s': %s", approval.ProtocolID, err)
		return false, err
	}
	log.L(ctx).Infof("Token approval recorded id=%s author=%s", approval.ProtocolID, approval.Key)
	return true, nil
}

func (em *eventManager) TokensApproved(ti tokens.Plugin, approval *tokens.TokenApproval) error {
	err := em.retry.Do(em.ctx, "persist token approval", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			if valid, err := em.persistTokenApproval(ctx, approval); !valid || err != nil {
				return err
			}

			event := fftypes.NewEvent(fftypes.EventTypeApprovalConfirmed, approval.Namespace, approval.LocalID, approval.TX.ID, "")
			return em.database.InsertEvent(ctx, event)
		})
		return err != nil, err // retry indefinitely (until context closes)
	})

	return err
}
