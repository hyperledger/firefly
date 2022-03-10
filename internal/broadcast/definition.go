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

	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (bm *broadcastManager) BroadcastDefinitionAsNode(ctx context.Context, ns string, def fftypes.Definition, tag string, waitConfirm bool) (msg *fftypes.Message, err error) {
	return bm.BroadcastDefinition(ctx, ns, def, &fftypes.SignerRef{ /* resolve to node default */ }, tag, waitConfirm)
}

func (bm *broadcastManager) BroadcastDefinition(ctx context.Context, ns string, def fftypes.Definition, signingIdentity *fftypes.SignerRef, tag string, waitConfirm bool) (msg *fftypes.Message, err error) {

	err = bm.identity.ResolveInputSigningIdentity(ctx, ns, signingIdentity)
	if err != nil {
		return nil, err
	}

	return bm.broadcastDefinitionCommon(ctx, ns, def, signingIdentity, tag, waitConfirm)
}

// BroadcastIdentityClaim is a special form of BroadcastDefinitionAsNode where the signing identity does not need to have been pre-registered
// The blockchain "key" will be normalized, but the "author" will pass through unchecked
func (bm *broadcastManager) BroadcastIdentityClaim(ctx context.Context, ns string, def *fftypes.IdentityClaim, signingIdentity *fftypes.SignerRef, tag string, waitConfirm bool) (msg *fftypes.Message, err error) {

	signingIdentity.Key, err = bm.identity.NormalizeSigningKey(ctx, signingIdentity.Key, identity.KeyNormalizationBlockchainPlugin)
	if err != nil {
		return nil, err
	}

	return bm.broadcastDefinitionCommon(ctx, ns, def, signingIdentity, tag, waitConfirm)
}

func (bm *broadcastManager) broadcastDefinitionCommon(ctx context.Context, ns string, def fftypes.Definition, signingIdentity *fftypes.SignerRef, tag string, waitConfirm bool) (*fftypes.Message, error) {

	// Serialize it into a data object, as a piece of data we can write to a message
	d := &fftypes.Data{
		Validator: fftypes.ValidatorTypeSystemDefinition,
		ID:        fftypes.NewUUID(),
		Namespace: ns,
		Created:   fftypes.Now(),
	}
	b, err := json.Marshal(&def)
	if err == nil {
		d.Value = fftypes.JSONAnyPtrBytes(b)
		err = d.Seal(ctx, nil)
	}
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Create a broadcast message referring to the data
	newMsg := &data.NewMessage{
		Message: &fftypes.MessageInOut{
			Message: fftypes.Message{
				Header: fftypes.MessageHeader{
					Namespace: ns,
					Type:      fftypes.MessageTypeDefinition,
					SignerRef: *signingIdentity,
					Topics:    fftypes.FFStringArray{def.Topic()},
					Tag:       tag,
					TxType:    fftypes.TransactionTypeBatchPin,
				},
			},
		},
		ResolvedData: data.Resolved{
			NewData: fftypes.DataArray{d},
		},
	}

	// Broadcast the message
	sender := broadcastSender{
		mgr:       bm,
		namespace: ns,
		msg:       newMsg,
		resolved:  true,
	}
	sender.setDefaults()
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &newMsg.Message.Message, err
}
