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

package broadcast

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (bm *broadcastManager) BroadcastDefinitionAsNode(ctx context.Context, def fftypes.Definition, tag fftypes.SystemTag, waitConfirm bool) (msg *fftypes.Message, err error) {
	return bm.BroadcastDefinition(ctx, def, &fftypes.Identity{ /* resolve to node default */ }, tag, waitConfirm)
}

func (bm *broadcastManager) BroadcastDefinition(ctx context.Context, def fftypes.Definition, signingIdentity *fftypes.Identity, tag fftypes.SystemTag, waitConfirm bool) (msg *fftypes.Message, err error) {

	err = bm.identity.ResolveInputIdentity(ctx, signingIdentity)
	if err != nil {
		return nil, err
	}

	// Ensure the broadcast message is nil on the sending side - only set on receiving side
	def.SetBroadcastMessage(nil)

	// Serialize it into a data object, as a piece of data we can write to a message
	data := &fftypes.Data{
		Validator: fftypes.ValidatorTypeSystemDefinition,
		ID:        fftypes.NewUUID(),
		Namespace: fftypes.SystemNamespace,
		Created:   fftypes.Now(),
	}
	data.Value, err = json.Marshal(&def)
	if err == nil {
		err = data.Seal(ctx)
	}
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Write as data to the local store
	if err = bm.database.UpsertData(ctx, data, true, false /* we just generated the ID, so it is new */); err != nil {
		return nil, err
	}

	// Create a broadcast message referring to the data
	msg = &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: fftypes.SystemNamespace,
			Type:      fftypes.MessageTypeDefinition,
			Identity:  *signingIdentity,
			Topics:    fftypes.FFNameArray{def.Topic()},
			Tag:       string(tag),
			TxType:    fftypes.TransactionTypeBatchPin,
		},
		Data: fftypes.DataRefs{
			{ID: data.ID, Hash: data.Hash},
		},
	}

	// Broadcast the message
	return bm.broadcastMessageCommon(ctx, msg, waitConfirm)
}
