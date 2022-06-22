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

package broadcast

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/pkg/core"
)

func (bm *broadcastManager) BroadcastDefinitionAsNode(ctx context.Context, def core.Definition, tag string, waitConfirm bool) (msg *core.Message, err error) {
	return bm.BroadcastDefinition(ctx, def, &core.SignerRef{ /* resolve to node default */ }, tag, waitConfirm)
}

func (bm *broadcastManager) BroadcastDefinition(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (msg *core.Message, err error) {

	err = bm.identity.ResolveInputSigningIdentity(ctx, signingIdentity)
	if err != nil {
		return nil, err
	}

	return bm.broadcastDefinitionCommon(ctx, def, signingIdentity, tag, waitConfirm)
}

// BroadcastIdentityClaim is a special form of BroadcastDefinitionAsNode where the signing identity does not need to have been pre-registered
// The blockchain "key" will be normalized, but the "author" will pass through unchecked
func (bm *broadcastManager) BroadcastIdentityClaim(ctx context.Context, def *core.IdentityClaim, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (msg *core.Message, err error) {

	signingIdentity.Key, err = bm.identity.NormalizeSigningKey(ctx, signingIdentity.Key, identity.KeyNormalizationBlockchainPlugin)
	if err != nil {
		return nil, err
	}

	return bm.broadcastDefinitionCommon(ctx, def, signingIdentity, tag, waitConfirm)
}

func (bm *broadcastManager) broadcastDefinitionCommon(ctx context.Context, def core.Definition, signingIdentity *core.SignerRef, tag string, waitConfirm bool) (*core.Message, error) {

	// Serialize it into a data object, as a piece of data we can write to a message
	d := &core.Data{
		Validator: core.ValidatorTypeSystemDefinition,
		ID:        fftypes.NewUUID(),
		Namespace: bm.namespace,
		Created:   fftypes.Now(),
	}
	b, err := json.Marshal(&def)
	if err == nil {
		d.Value = fftypes.JSONAnyPtrBytes(b)
		err = d.Seal(ctx, nil)
	}
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgSerializationFailed)
	}

	// Create a broadcast message referring to the data
	newMsg := &data.NewMessage{
		Message: &core.MessageInOut{
			Message: core.Message{
				Header: core.MessageHeader{
					Namespace: bm.namespace,
					Type:      core.MessageTypeDefinition,
					SignerRef: *signingIdentity,
					Topics:    core.FFStringArray{def.Topic()},
					Tag:       tag,
					TxType:    core.TransactionTypeBatchPin,
				},
				Data: core.DataRefs{
					{ID: d.ID, Hash: d.Hash, ValueSize: d.ValueSize},
				},
			},
		},
		NewData: core.DataArray{d},
		AllData: core.DataArray{d},
	}

	// Broadcast the message
	sender := broadcastSender{
		mgr:      bm,
		msg:      newMsg,
		resolved: true,
	}
	sender.setDefaults()
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &newMsg.Message.Message, err
}
