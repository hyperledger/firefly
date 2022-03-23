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

	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (t *transactionHelper) EnrichEvent(ctx context.Context, event *fftypes.Event) (*fftypes.EnrichedEvent, error) {
	e := &fftypes.EnrichedEvent{
		Event: *event,
	}

	switch event.Type {
	case fftypes.EventTypeTransactionSubmitted:
		tx, err := t.GetTransactionByIDCached(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Transaction = tx
	case fftypes.EventTypeMessageConfirmed, fftypes.EventTypeMessageRejected:
		msg, _, _, err := t.data.GetMessageWithDataCached(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Message = msg
	case fftypes.EventTypeBlockchainEventReceived:
		be, err := t.database.GetBlockchainEventByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.BlockchainEvent = be
	case fftypes.EventTypeContractAPIConfirmed:
		contractAPI, err := t.database.GetContractAPIByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.ContractAPI = contractAPI
	case fftypes.EventTypeContractInterfaceConfirmed:
		contractInterface, err := t.database.GetFFIByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.ContractInterface = contractInterface
	case fftypes.EventTypeDatatypeConfirmed:
		dt, err := t.database.GetDatatypeByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Datatype = dt
	case fftypes.EventTypeIdentityConfirmed, fftypes.EventTypeIdentityUpdated:
		identity, err := t.database.GetIdentityByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Identity = identity
	case fftypes.EventTypeNamespaceConfirmed:
		ns, err := t.database.GetNamespaceByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.NamespaceDetails = ns
	case fftypes.EventTypePoolConfirmed:
		tokenPool, err := t.database.GetTokenPoolByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.TokenPool = tokenPool
	case fftypes.EventTypeApprovalConfirmed, fftypes.EventTypeApprovalOpFailed:
		approval, err := t.database.GetTokenApproval(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.TokenApproval = approval
	case fftypes.EventTypeTransferConfirmed, fftypes.EventTypeTransferOpFailed:
		transfer, err := t.database.GetTokenTransfer(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.TokenTransfer = transfer
	}
	return e, nil
}
