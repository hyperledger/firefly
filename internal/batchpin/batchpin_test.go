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

package batchpin

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger-labs/firefly/mocks/blockchainmocks"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestBatchPinSubmitter(t *testing.T) *batchPinSubmitter {
	mdi := &databasemocks.Plugin{}
	mii := &identitymocks.Plugin{}
	mbi := &blockchainmocks.Plugin{}
	mbi.On("Name").Return("ut").Maybe()
	return NewBatchPinSubmitter(mdi, mii, mbi).(*batchPinSubmitter)
}

func TestSubmitPinnedBatchOk(t *testing.T) {

	bp := newTestBatchPinSubmitter(t)
	ctx := context.Background()

	mii := bp.identity.(*identitymocks.Plugin)
	mbi := bp.blockchain.(*blockchainmocks.Plugin)
	mdi := bp.database.(*databasemocks.Plugin)

	identity := &fftypes.Identity{
		Identifier: "id1",
		OnChain:    "0x12345",
	}
	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "id1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	contexts := []*fftypes.Bytes32{}

	mii.On("Resolve", ctx, "id1").Return(identity, nil)
	mbi.On("VerifyIdentitySyntax", ctx, identity).Return(nil)
	mdi.On("UpsertTransaction", ctx, mock.Anything, false).Return(nil)
	mdi.On("UpsertOperation", ctx, mock.MatchedBy(func(op *fftypes.Operation) bool {
		assert.Equal(t, fftypes.OpTypeBlockchainBatchPin, op.Type)
		assert.Equal(t, "ut", op.Plugin)
		assert.Equal(t, *batch.Payload.TX.ID, *op.Transaction)
		return true
	}), false).Return(nil)
	mbi.On("SubmitBatchPin", ctx, (*fftypes.UUID)(nil), identity, mock.Anything).Return(nil)

	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.NoError(t, err)

}

func TestSubmitPinnedBatchOpFail(t *testing.T) {

	bp := newTestBatchPinSubmitter(t)
	ctx := context.Background()

	mii := bp.identity.(*identitymocks.Plugin)
	mbi := bp.blockchain.(*blockchainmocks.Plugin)
	mdi := bp.database.(*databasemocks.Plugin)

	identity := &fftypes.Identity{
		Identifier: "id1",
		OnChain:    "0x12345",
	}
	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "id1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	contexts := []*fftypes.Bytes32{}

	mii.On("Resolve", ctx, "id1").Return(identity, nil)
	mbi.On("VerifyIdentitySyntax", ctx, identity).Return(nil)
	mdi.On("UpsertTransaction", ctx, mock.Anything, false).Return(nil)
	mdi.On("UpsertOperation", ctx, mock.Anything, false).Return(fmt.Errorf("pop"))

	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.Regexp(t, "pop", err)

}

func TestSubmitPinnedBatchTxInsertFail(t *testing.T) {

	bp := newTestBatchPinSubmitter(t)
	ctx := context.Background()

	mii := bp.identity.(*identitymocks.Plugin)
	mbi := bp.blockchain.(*blockchainmocks.Plugin)
	mdi := bp.database.(*databasemocks.Plugin)

	identity := &fftypes.Identity{
		Identifier: "id1",
		OnChain:    "0x12345",
	}
	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "id1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	contexts := []*fftypes.Bytes32{}

	mii.On("Resolve", ctx, "id1").Return(identity, nil)
	mbi.On("VerifyIdentitySyntax", ctx, identity).Return(nil)
	mdi.On("UpsertTransaction", ctx, mock.Anything, false).Return(fmt.Errorf("pop"))

	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.Regexp(t, "pop", err)

}

func TestSubmitPinnedBatchTxBadIdentity(t *testing.T) {

	bp := newTestBatchPinSubmitter(t)
	ctx := context.Background()

	mii := bp.identity.(*identitymocks.Plugin)
	mbi := bp.blockchain.(*blockchainmocks.Plugin)

	identity := &fftypes.Identity{
		Identifier: "id1",
		OnChain:    "badness",
	}
	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "id1",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	contexts := []*fftypes.Bytes32{}

	mii.On("Resolve", ctx, "id1").Return(identity, nil)
	mbi.On("VerifyIdentitySyntax", ctx, identity).Return(fmt.Errorf("pop"))

	err := bp.SubmitPinnedBatch(ctx, batch, contexts)
	assert.Regexp(t, "pop", err)

}
