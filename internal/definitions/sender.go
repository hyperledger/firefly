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
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/contracts"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

type Sender interface {
	core.Named

	Init(handler Handler)
	CreateDefinition(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (msg *core.Message, err error)
	DefineDatatype(ctx context.Context, datatype *core.Datatype, waitConfirm bool) error
	DefineIdentity(ctx context.Context, def *core.IdentityClaim, signingIdentity *core.SignerRef, parentSigner *core.SignerRef, tag string, waitConfirm bool) error
	DefineTokenPool(ctx context.Context, pool *core.TokenPoolAnnouncement, waitConfirm bool) error
	DefineFFI(ctx context.Context, ffi *core.FFI, waitConfirm bool) error
	DefineContractAPI(ctx context.Context, httpServerURL string, api *core.ContractAPI, waitConfirm bool) error
}

type definitionSender struct {
	ctx        context.Context
	namespace  string
	multiparty bool
	database   database.Plugin
	broadcast  broadcast.Manager // optional
	identity   identity.Manager
	data       data.Manager
	contracts  contracts.Manager
	handler    *definitionHandler
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

func NewDefinitionSender(ctx context.Context, ns string, multiparty bool, di database.Plugin, bm broadcast.Manager, im identity.Manager, dm data.Manager, cm contracts.Manager) (Sender, error) {
	if di == nil || im == nil || dm == nil || cm == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError)
	}
	return &definitionSender{
		ctx:        ctx,
		namespace:  ns,
		multiparty: multiparty,
		database:   di,
		broadcast:  bm,
		identity:   im,
		data:       dm,
		contracts:  cm,
	}, nil
}

func (bm *definitionSender) Name() string {
	return "DefinitionSender"
}

func (bm *definitionSender) Init(handler Handler) {
	bm.handler = handler.(*definitionHandler)
}

func (bm *definitionSender) createDefinitionDefault(ctx context.Context, def core.Definition, tag string, waitConfirm bool) (msg *core.Message, err error) {
	return bm.CreateDefinition(ctx, def, &core.SignerRef{ /* resolve to node default */ }, tag, waitConfirm)
}

func (bm *definitionSender) CreateDefinition(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (msg *core.Message, err error) {

	if bm.multiparty {
		err = bm.identity.ResolveInputSigningIdentity(ctx, signingIdentity)
		if err != nil {
			return nil, err
		}
	}

	return bm.createDefinitionCommon(ctx, def, signingIdentity, tag, waitConfirm)
}

func (bm *definitionSender) createDefinitionCommon(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (*core.Message, error) {

	b, err := json.Marshal(&def)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgSerializationFailed)
	}
	dataValue := fftypes.JSONAnyPtrBytes(b)
	message := &core.MessageInOut{
		Message: core.Message{
			Header: core.MessageHeader{
				Namespace: bm.namespace,
				Type:      core.MessageTypeDefinition,
				Topics:    core.FFStringArray{def.Topic()},
				Tag:       tag,
				TxType:    core.TransactionTypeBatchPin,
			},
		},
	}
	if signingIdentity != nil {
		message.Header.SignerRef = *signingIdentity
	}

	message.InlineData = core.InlineData{
		&core.DataRefOrValue{Value: dataValue},
	}
	sender := bm.broadcast.NewBroadcast(message)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &message.Message, err
}