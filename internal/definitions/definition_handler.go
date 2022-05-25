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
	"encoding/json"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/assets"
	"github.com/hyperledger/firefly/internal/contracts"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
)

type DefinitionHandler interface {
	HandleDefinitionBroadcast(ctx context.Context, state DefinitionBatchState, msg *core.Message, data core.DataArray, tx *fftypes.UUID) (HandlerResult, error)
}

type HandlerResult struct {
	Action           DefinitionMessageAction
	CustomCorrelator *fftypes.UUID
}

// DefinitionMessageAction is the action to be taken on an individual definition message
type DefinitionMessageAction int

const (
	// ActionReject the message was successfully processed, but was malformed/invalid and should be marked as rejected
	ActionReject DefinitionMessageAction = iota

	// ActionConfirm the message was valid and should be confirmed
	ActionConfirm

	// ActionRetry a recoverable error was encountered - batch should be halted and then re-processed from the start
	ActionRetry

	// ActionWait the message is still awaiting further pieces for aggregation and should be held in pending state
	ActionWait
)

func (dma DefinitionMessageAction) String() string {
	switch dma {
	case ActionReject:
		return "reject"
	case ActionConfirm:
		return "confirm"
	case ActionRetry:
		return "retry"
	case ActionWait:
		return "wait"
	default:
		return "unknown"
	}
}

// DefinitionBatchState tracks the state between definition handlers that run in-line on the pin processing route in the
// aggregator as part of a batch of pins. They might have complex API calls, and interdependencies, that need to be managed via this state.
// The actions to be taken at the end of a definition batch.
// See further notes on "batchState" in the event aggregator
type DefinitionBatchState interface {
	// PreFinalize may perform a blocking action (possibly to an external connector) that should execute outside database RunAsGroup
	AddPreFinalize(func(ctx context.Context) error)

	// Finalize may perform final, non-idempotent database operations (such as inserting Events)
	AddFinalize(func(ctx context.Context) error)

	// GetPendingConfirm returns a map of messages are that pending confirmation after already being processed in this batch
	GetPendingConfirm() map[fftypes.UUID]*core.Message

	// Notify of a DID claim locking in, so a rewind gets queued for it to go back and process any dependent child identities/messages
	DIDClaimConfirmed(did string)
}

type definitionHandlers struct {
	database   database.Plugin
	blockchain blockchain.Plugin
	exchange   dataexchange.Plugin
	data       data.Manager
	identity   identity.Manager
	assets     assets.Manager
	contracts  contracts.Manager
}

func NewDefinitionHandler(ctx context.Context, di database.Plugin, bi blockchain.Plugin, dx dataexchange.Plugin, dm data.Manager, im identity.Manager, am assets.Manager, cm contracts.Manager) (DefinitionHandler, error) {
	if di == nil || bi == nil || dx == nil || dm == nil || im == nil || am == nil || cm == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "DefinitionHandler")
	}
	return &definitionHandlers{
		database:   di,
		blockchain: bi,
		exchange:   dx,
		data:       dm,
		identity:   im,
		assets:     am,
		contracts:  cm,
	}, nil
}

func (dh *definitionHandlers) HandleDefinitionBroadcast(ctx context.Context, state DefinitionBatchState, msg *core.Message, data core.DataArray, tx *fftypes.UUID) (msgAction HandlerResult, err error) {
	l := log.L(ctx)
	l.Infof("Processing system definition broadcast '%s' [%s]", msg.Header.Tag, msg.Header.ID)
	switch msg.Header.Tag {
	case core.SystemTagDefineDatatype:
		return dh.handleDatatypeBroadcast(ctx, state, msg, data, tx)
	case core.SystemTagDefineNamespace:
		return dh.handleNamespaceBroadcast(ctx, state, msg, data, tx)
	case core.DeprecatedSystemTagDefineOrganization:
		return dh.handleDeprecatedOrganizationBroadcast(ctx, state, msg, data)
	case core.DeprecatedSystemTagDefineNode:
		return dh.handleDeprecatedNodeBroadcast(ctx, state, msg, data)
	case core.SystemTagIdentityClaim:
		return dh.handleIdentityClaimBroadcast(ctx, state, msg, data, nil)
	case core.SystemTagIdentityVerification:
		return dh.handleIdentityVerificationBroadcast(ctx, state, msg, data)
	case core.SystemTagIdentityUpdate:
		return dh.handleIdentityUpdateBroadcast(ctx, state, msg, data)
	case core.SystemTagDefinePool:
		return dh.handleTokenPoolBroadcast(ctx, state, msg, data)
	case core.SystemTagDefineFFI:
		return dh.handleFFIBroadcast(ctx, state, msg, data, tx)
	case core.SystemTagDefineContractAPI:
		return dh.handleContractAPIBroadcast(ctx, state, msg, data, tx)
	default:
		l.Warnf("Unknown SystemTag '%s' for definition ID '%s'", msg.Header.Tag, msg.Header.ID)
		return HandlerResult{Action: ActionReject}, nil
	}
}

func (dh *definitionHandlers) getSystemBroadcastPayload(ctx context.Context, msg *core.Message, data core.DataArray, res core.Definition) (valid bool) {
	l := log.L(ctx)
	if len(data) != 1 {
		l.Warnf("Unable to process system broadcast %s - expecting 1 attachment, found %d", msg.Header.ID, len(data))
		return false
	}
	err := json.Unmarshal(data[0].Value.Bytes(), &res)
	if err != nil {
		l.Warnf("Unable to process system broadcast %s - unmarshal failed: %s", msg.Header.ID, err)
		return false
	}
	res.SetBroadcastMessage(msg.Header.ID)
	return true
}
