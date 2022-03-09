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
		tx, err := t.database.GetTransactionByID(ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		e.Transaction = tx
	case fftypes.EventTypeMessageConfirmed, fftypes.EventTypeMessageRejected:
		msg, err := t.database.GetMessageByID(ctx, event.Reference)
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
	}
	return e, nil
}
