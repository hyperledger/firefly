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

package events

import (
	"fmt"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (em *eventManager) TxSubmissionUpdate(bi blockchain.Plugin, tx string, txState fftypes.OpStatus, protocolTxID, errorMessage string, additionalInfo fftypes.JSONObject) error {

	// Find a matching operation, for this plugin, with the specified ID.
	// We retry a few times, as there's an outside possibility of the event arriving before we're finished persisting the operation itself
	var operations []*fftypes.Operation
	fb := database.OperationQueryFactory.NewFilter(em.ctx)
	filter := fb.And(
		fb.Eq("tx", tx),
		fb.Eq("plugin", bi.Name()),
	)
	err := em.retry.Do(em.ctx, fmt.Sprintf("correlate tx %s", tx), func(attempt int) (retry bool, err error) {
		operations, _, err = em.database.GetOperations(em.ctx, filter)
		if err == nil && len(operations) == 0 {
			err = i18n.NewError(em.ctx, i18n.Msg404NotFound)
		}
		return (err != nil && attempt <= em.opCorrelationRetries), err
	})
	if err != nil {
		log.L(em.ctx).Warnf("Failed to correlate tracking ID '%s' with a submitted operation", tx)
		return nil
	}

	update := database.OperationQueryFactory.NewUpdate(em.ctx).
		Set("status", txState).
		Set("error", errorMessage).
		Set("info", additionalInfo)
	for _, op := range operations {
		if err := em.database.UpdateOperation(em.ctx, op.ID, update); err != nil {
			return err
		}
	}

	return nil
}
