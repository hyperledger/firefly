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

package defsender

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/pkg/core"
)

type Sender interface {
	core.Named

	CreateDefinition(ctx context.Context, def core.Definition, tag string, waitConfirm bool) (msg *core.Message, err error)
	CreateDefinitionWithIdentity(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (msg *core.Message, err error)
	CreateDatatype(ctx context.Context, datatype *core.Datatype, waitConfirm bool) (msg *core.Message, err error)
	CreateIdentityClaim(ctx context.Context, def *core.IdentityClaim, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (msg *core.Message, err error)
	CreateTokenPool(ctx context.Context, pool *core.TokenPoolAnnouncement, waitConfirm bool) (msg *core.Message, err error)
}

type definitionSender struct {
	ctx       context.Context
	namespace string
	broadcast broadcast.Manager
	identity  identity.Manager
	data      data.Manager
}

func NewDefinitionSender(ctx context.Context, ns string, bm broadcast.Manager, im identity.Manager, dm data.Manager) (Sender, error) {
	if bm == nil || im == nil || dm == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError)
	}
	return &definitionSender{
		ctx:       ctx,
		namespace: ns,
		broadcast: bm,
		identity:  im,
		data:      dm,
	}, nil
}

func (bm *definitionSender) Name() string {
	return "DefinitionSender"
}

func (bm *definitionSender) CreateDefinition(ctx context.Context, def core.Definition, tag string, waitConfirm bool) (msg *core.Message, err error) {
	return bm.CreateDefinitionWithIdentity(ctx, def, &core.SignerRef{ /* resolve to node default */ }, tag, waitConfirm)
}

func (bm *definitionSender) CreateDefinitionWithIdentity(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (msg *core.Message, err error) {

	err = bm.identity.ResolveInputSigningIdentity(ctx, signingIdentity)
	if err != nil {
		return nil, err
	}

	return bm.broadcastDefinitionCommon(ctx, def, signingIdentity, tag, waitConfirm)
}

// CreateIdentityClaim is a special form of CreateDefinition where the signing identity does not need to have been pre-registered
// The blockchain "key" will be normalized, but the "author" will pass through unchecked
func (bm *definitionSender) CreateIdentityClaim(ctx context.Context, def *core.IdentityClaim, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (msg *core.Message, err error) {

	signingIdentity.Key, err = bm.identity.NormalizeSigningKey(ctx, signingIdentity.Key, identity.KeyNormalizationBlockchainPlugin)
	if err != nil {
		return nil, err
	}

	return bm.broadcastDefinitionCommon(ctx, def, signingIdentity, tag, waitConfirm)
}

func (bm *definitionSender) broadcastDefinitionCommon(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (*core.Message, error) {

	b, err := json.Marshal(&def)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgSerializationFailed)
	}

	message := &core.MessageInOut{
		Message: core.Message{
			Header: core.MessageHeader{
				Namespace: bm.namespace,
				Type:      core.MessageTypeDefinition,
				SignerRef: *signingIdentity,
				Topics:    core.FFStringArray{def.Topic()},
				Tag:       tag,
				TxType:    core.TransactionTypeBatchPin,
			},
		},
		InlineData: core.InlineData{
			&core.DataRefOrValue{
				Value: fftypes.JSONAnyPtrBytes(b),
			},
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
