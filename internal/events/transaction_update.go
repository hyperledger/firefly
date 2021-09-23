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
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (em *eventManager) TxSubmissionUpdate(plugin fftypes.Named, tx string, txState fftypes.OpStatus, errorMessage string, opOutput fftypes.JSONObject) error {

	// Find a matching operation, for this plugin, with the specified ID.
	fb := database.OperationQueryFactory.NewFilter(em.ctx)
	filter := fb.And(
		fb.Eq("tx", tx),
		fb.Eq("plugin", plugin.Name()),
	)
	operations, _, err := em.database.GetOperations(em.ctx, filter)
	if err == nil && len(operations) == 0 {
		err = i18n.NewError(em.ctx, i18n.Msg404NotFound)
	}
	if err != nil {
		log.L(em.ctx).Debugf("TX '%s' submission update ignored, as it was not submitted by this node", tx)
		return nil
	}

	update := database.OperationQueryFactory.NewUpdate(em.ctx).
		Set("status", txState).
		Set("error", errorMessage).
		Set("output", opOutput)
	for _, op := range operations {
		if err := em.database.UpdateOperation(em.ctx, op.ID, update); err != nil {
			return err
		}
	}

	return nil
}
