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
	"fmt"

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

type Handler interface {
	HandleDefinitionBroadcast(ctx context.Context, state *core.BatchState, msg *core.Message, data core.DataArray, tx *fftypes.UUID) (HandlerResult, error)
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

type definitionHandler struct {
	namespace  *core.Namespace
	multiparty bool
	database   database.Plugin
	blockchain blockchain.Plugin   // optional
	exchange   dataexchange.Plugin // optional
	data       data.Manager
	identity   identity.Manager
	assets     assets.Manager
	contracts  contracts.Manager // optional
	tokenNames map[string]string // mapping of token connector remote name => name
}

func newDefinitionHandler(ctx context.Context, ns *core.Namespace, multiparty bool, di database.Plugin, bi blockchain.Plugin, dx dataexchange.Plugin, dm data.Manager, im identity.Manager, am assets.Manager, cm contracts.Manager, tokenNames map[string]string) (*definitionHandler, error) {
	if di == nil || dm == nil || im == nil || am == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "DefinitionHandler")
	}
	return &definitionHandler{
		namespace:  ns,
		multiparty: multiparty,
		database:   di,
		blockchain: bi,
		exchange:   dx,
		data:       dm,
		identity:   im,
		assets:     am,
		contracts:  cm,
		tokenNames: tokenNames,
	}, nil
}

func (dh *definitionHandler) HandleDefinitionBroadcast(ctx context.Context, state *core.BatchState, msg *core.Message, data core.DataArray, tx *fftypes.UUID) (msgAction HandlerResult, err error) {
	l := log.L(ctx)
	l.Infof("Processing system definition '%s' [%s]", msg.Header.Tag, msg.Header.ID)
	switch msg.Header.Tag {
	case core.SystemTagDefineDatatype:
		return dh.handleDatatypeBroadcast(ctx, state, msg, data, tx)
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
		return HandlerResult{Action: ActionReject}, fmt.Errorf("unknown system tag '%s' for definition ID '%s'", msg.Header.Tag, msg.Header.ID)
	}
}

func (dh *definitionHandler) getSystemBroadcastPayload(ctx context.Context, msg *core.Message, data core.DataArray, res core.Definition) (valid bool) {
	l := log.L(ctx)
	if len(data) != 1 {
		l.Warnf("Unable to process system definition %s - expecting 1 attachment, found %d", msg.Header.ID, len(data))
		return false
	}
	err := json.Unmarshal(data[0].Value.Bytes(), &res)
	if err != nil {
		l.Warnf("Unable to process system definition %s - unmarshal failed: %s", msg.Header.ID, err)
		return false
	}
	res.SetBroadcastMessage(msg.Header.ID)
	return true
}
