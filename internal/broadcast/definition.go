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

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
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

func (bm *broadcastManager) broadcastDefinitionCommon(ctx context.Context, ns string, def fftypes.Definition, signingIdentity *fftypes.SignerRef, tag string, waitConfirm bool) (msg *fftypes.Message, err error) {

	// Serialize it into a data object, as a piece of data we can write to a message
	data := &fftypes.Data{
		Validator: fftypes.ValidatorTypeSystemDefinition,
		ID:        fftypes.NewUUID(),
		Namespace: ns,
		Created:   fftypes.Now(),
	}
	b, err := json.Marshal(&def)
	if err == nil {
		data.Value = fftypes.JSONAnyPtrBytes(b)
		err = data.Seal(ctx, nil)
	}
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Write as data to the local store
	if err = bm.database.UpsertData(ctx, data, database.UpsertOptimizationNew); err != nil {
		return nil, err
	}

	// Create a broadcast message referring to the data
	in := &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Namespace: ns,
				Type:      fftypes.MessageTypeDefinition,
				SignerRef: *signingIdentity,
				Topics:    fftypes.FFStringArray{def.Topic()},
				Tag:       tag,
				TxType:    fftypes.TransactionTypeBatchPin,
			},
			Data: fftypes.DataRefs{
				{ID: data.ID, Hash: data.Hash},
			},
		},
	}

	// Broadcast the message
	sender := broadcastSender{
		mgr:       bm,
		namespace: ns,
		msg:       in,
		resolved:  true,
	}
	sender.setDefaults()
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &in.Message, err
}
