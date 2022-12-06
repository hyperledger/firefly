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
	"github.com/hyperledger/firefly/internal/assets"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/contracts"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
)

type Sender interface {
	core.Named

	ClaimIdentity(ctx context.Context, def *core.IdentityClaim, signingIdentity *core.SignerRef, parentSigner *core.SignerRef, waitConfirm bool) error
	UpdateIdentity(ctx context.Context, identity *core.Identity, def *core.IdentityUpdate, signingIdentity *core.SignerRef, waitConfirm bool) error
	DefineDatatype(ctx context.Context, datatype *core.Datatype, waitConfirm bool) error
	DefineTokenPool(ctx context.Context, pool *core.TokenPoolAnnouncement, waitConfirm bool) error
	DefineFFI(ctx context.Context, ffi *fftypes.FFI, waitConfirm bool) error
	DefineContractAPI(ctx context.Context, httpServerURL string, api *core.ContractAPI, waitConfirm bool) error
}

type definitionSender struct {
	ctx                 context.Context
	namespace           string
	multiparty          bool
	database            database.Plugin
	broadcast           broadcast.Manager // optional
	identity            identity.Manager
	data                data.Manager
	contracts           contracts.Manager // optional
	handler             *definitionHandler
	tokenBroadcastNames map[string]string // mapping of token connector name => remote name
}

// Definitions that get processed immediately will create a temporary batch state and then finalize it inline
func fakeBatch(ctx context.Context, handler func(context.Context, *core.BatchState) (HandlerResult, error)) (err error) {
	var state core.BatchState
	_, err = handler(ctx, &state)
	if err == nil {
		err = state.RunPreFinalize(ctx)
	}
	if err == nil {
		err = state.RunFinalize(ctx)
	}
	return err
}

func NewDefinitionSender(ctx context.Context, ns *core.Namespace, multiparty bool, di database.Plugin, bi blockchain.Plugin, dx dataexchange.Plugin, bm broadcast.Manager, im identity.Manager, dm data.Manager, am assets.Manager, cm contracts.Manager, tokenBroadcastNames map[string]string) (Sender, Handler, error) {
	if di == nil || im == nil || dm == nil {
		return nil, nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "DefinitionSender")
	}
	ds := &definitionSender{
		ctx:                 ctx,
		namespace:           ns.Name,
		multiparty:          multiparty,
		database:            di,
		broadcast:           bm,
		identity:            im,
		data:                dm,
		contracts:           cm,
		tokenBroadcastNames: tokenBroadcastNames,
	}
	dh, err := newDefinitionHandler(ctx, ns, multiparty, di, bi, dx, dm, im, am, cm, reverseMap(tokenBroadcastNames))
	ds.handler = dh
	return ds, dh, err
}

// reverseMap reverses the key/values of a given map
func reverseMap(orderedMap map[string]string) map[string]string {
	reverseMap := make(map[string]string, len(orderedMap))
	for k, v := range orderedMap {
		reverseMap[v] = k
	}
	return reverseMap
}

func (bm *definitionSender) Name() string {
	return "DefinitionSender"
}

func (bm *definitionSender) sendDefinitionDefault(ctx context.Context, def core.Definition, tag string, waitConfirm bool) (msg *core.Message, err error) {
	return bm.sendDefinition(ctx, def, &core.SignerRef{ /* resolve to node default */ }, tag, waitConfirm)
}

func (bm *definitionSender) sendDefinition(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (msg *core.Message, err error) {

	err = bm.identity.ResolveInputSigningIdentity(ctx, signingIdentity)
	if err != nil {
		return nil, err
	}

	return bm.sendDefinitionCommon(ctx, def, signingIdentity, tag, waitConfirm)
}

func (bm *definitionSender) sendDefinitionCommon(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (*core.Message, error) {

	b, err := json.Marshal(&def)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgSerializationFailed)
	}
	dataValue := fftypes.JSONAnyPtrBytes(b)
	message := &core.MessageInOut{
		Message: core.Message{
			Header: core.MessageHeader{
				Type:      core.MessageTypeDefinition,
				Topics:    fftypes.FFStringArray{def.Topic()},
				Tag:       tag,
				TxType:    core.TransactionTypeBatchPin,
				SignerRef: *signingIdentity,
			},
		},
		InlineData: core.InlineData{
			&core.DataRefOrValue{Value: dataValue},
		},
	}

	sender := bm.broadcast.NewBroadcast(message)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &message.Message, err
}
