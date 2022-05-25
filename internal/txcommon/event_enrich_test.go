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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEnrichMessageConfirmed(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdm.On("GetMessageWithDataCached", mock.Anything, ref1).Return(&core.Message{
		Header: core.MessageHeader{ID: ref1},
	}, nil, true, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeMessageConfirmed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.Message.Header.ID)
}

func TestEnrichMessageFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdm.On("GetMessageWithDataCached", mock.Anything, ref1).Return(nil, nil, false, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeMessageConfirmed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichMessageRejected(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdm.On("GetMessageWithDataCached", mock.Anything, ref1).Return(&core.Message{
		Header: core.MessageHeader{ID: ref1},
	}, nil, true, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeMessageRejected,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.Message.Header.ID)
}

func TestEnrichTxSubmitted(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetTransactionByID", mock.Anything, ref1).Return(&core.Transaction{
		ID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeTransactionSubmitted,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.Transaction.ID)
}

func TestEnrichTxFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetTransactionByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeTransactionSubmitted,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichBlockchainEventSubmitted(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetBlockchainEventByID", mock.Anything, ref1).Return(&core.BlockchainEvent{
		ID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeBlockchainEventReceived,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.BlockchainEvent.ID)
}

func TestEnrichBlockchainEventFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetBlockchainEventByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeBlockchainEventReceived,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichContractAPISubmitted(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetContractAPIByID", mock.Anything, ref1).Return(&core.ContractAPI{
		ID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeContractAPIConfirmed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.ContractAPI.ID)
}

func TestEnrichContractAPItFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetContractAPIByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeContractAPIConfirmed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichContractInterfaceSubmitted(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetFFIByID", mock.Anything, ref1).Return(&core.FFI{
		ID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeContractInterfaceConfirmed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.ContractInterface.ID)
}

func TestEnrichContractInterfacetFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetFFIByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeContractInterfaceConfirmed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichDatatypeConfirmed(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetDatatypeByID", mock.Anything, ref1).Return(&core.Datatype{
		ID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeDatatypeConfirmed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.Datatype.ID)
}

func TestEnrichDatatypeConfirmedFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetDatatypeByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeDatatypeConfirmed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichIdentityConfirmed(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetIdentityByID", mock.Anything, ref1).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID: ref1,
		},
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeIdentityConfirmed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.Identity.IdentityBase.ID)
}

func TestEnrichIdentityConfirmedFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetIdentityByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeIdentityConfirmed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichNamespaceConfirmed(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetNamespaceByID", mock.Anything, ref1).Return(&core.Namespace{
		ID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeNamespaceConfirmed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.NamespaceDetails.ID)
}

func TestEnrichNamespaceConfirmedFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetNamespaceByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeNamespaceConfirmed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichTokenPoolConfirmed(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetTokenPoolByID", mock.Anything, ref1).Return(&core.TokenPool{
		ID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypePoolConfirmed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.TokenPool.ID)
}

func TestEnrichTokenPoolConfirmedFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetTokenPoolByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypePoolConfirmed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichTokenApprovalConfirmed(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetTokenApprovalByID", mock.Anything, ref1).Return(&core.TokenApproval{
		LocalID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeApprovalConfirmed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.TokenApproval.LocalID)
}

func TestEnrichTokenApprovalFailed(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetOperationByID", mock.Anything, ref1).Return(&core.Operation{
		ID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeApprovalOpFailed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.Operation.ID)
}

func TestEnrichTokenApprovalConfirmedFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetTokenApprovalByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeApprovalConfirmed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichTokenTransferConfirmed(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetTokenTransferByID", mock.Anything, ref1).Return(&core.TokenTransfer{
		LocalID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeTransferConfirmed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.TokenTransfer.LocalID)
}

func TestEnrichTokenTransferFailed(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetOperationByID", mock.Anything, ref1).Return(&core.Operation{
		ID: ref1,
	}, nil)

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeTransferOpFailed,
		Reference: ref1,
	}

	enriched, err := txHelper.EnrichEvent(ctx, event)
	assert.NoError(t, err)
	assert.Equal(t, ref1, enriched.Operation.ID)
}

func TestEnrichTokenTransferConfirmedFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetTokenTransferByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeTransferConfirmed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}

func TestEnrichOperationFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	txHelper := NewTransactionHelper(mdi, mdm)
	ctx := context.Background()

	// Setup the IDs
	ref1 := fftypes.NewUUID()
	ev1 := fftypes.NewUUID()

	// Setup enrichment
	mdi.On("GetOperationByID", mock.Anything, ref1).Return(nil, fmt.Errorf("pop"))

	event := &core.Event{
		ID:        ev1,
		Type:      core.EventTypeApprovalOpFailed,
		Reference: ref1,
	}

	_, err := txHelper.EnrichEvent(ctx, event)
	assert.EqualError(t, err, "pop")
}
