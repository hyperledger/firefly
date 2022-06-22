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

package txcommon

import (
	"context"

	"github.com/hyperledger/firefly/pkg/core"
)

func (t *transactionHelper) EnrichEvent(ctx context.Context, event *core.Event) (*core.EnrichedEvent, error) {
	e := &core.EnrichedEvent{
		Event: *event,
	}

	switch event.Type {
	case core.EventTypeTransactionSubmitted:
		tx, err := t.GetTransactionByIDCached(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Transaction = tx
	case core.EventTypeMessageConfirmed, core.EventTypeMessageRejected:
		msg, _, _, err := t.data.GetMessageWithDataCached(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Message = msg
	case core.EventTypeBlockchainEventReceived:
		be, err := t.GetBlockchainEventByIDCached(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.BlockchainEvent = be
	case core.EventTypeContractAPIConfirmed:
		contractAPI, err := t.database.GetContractAPIByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.ContractAPI = contractAPI
	case core.EventTypeContractInterfaceConfirmed:
		contractInterface, err := t.database.GetFFIByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.ContractInterface = contractInterface
	case core.EventTypeDatatypeConfirmed:
		dt, err := t.database.GetDatatypeByID(ctx, t.namespace, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Datatype = dt
	case core.EventTypeIdentityConfirmed, core.EventTypeIdentityUpdated:
		identity, err := t.database.GetIdentityByID(ctx, t.namespace, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Identity = identity
	case core.EventTypeNamespaceConfirmed:
		ns, err := t.database.GetNamespaceByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.NamespaceDetails = ns
	case core.EventTypePoolConfirmed:
		tokenPool, err := t.database.GetTokenPoolByID(ctx, t.namespace, event.Reference)
		if err != nil {
			return nil, err
		}
		e.TokenPool = tokenPool
	case core.EventTypeApprovalConfirmed:
		approval, err := t.database.GetTokenApprovalByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.TokenApproval = approval
	case core.EventTypeTransferConfirmed:
		transfer, err := t.database.GetTokenTransferByID(ctx, t.namespace, event.Reference)
		if err != nil {
			return nil, err
		}
		e.TokenTransfer = transfer
	case core.EventTypeApprovalOpFailed, core.EventTypeTransferOpFailed, core.EventTypeBlockchainInvokeOpFailed, core.EventTypePoolOpFailed, core.EventTypeBlockchainInvokeOpSucceeded:
		operation, err := t.database.GetOperationByID(ctx, t.namespace, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Operation = operation
	}
	return e, nil
}
