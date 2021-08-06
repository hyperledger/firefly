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

package privatemessaging

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (pm *privateMessaging) SendMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error) {
	in.Header.ID = nil
	return pm.SendMessageWithID(ctx, ns, in, waitConfirm)
}

func (pm *privateMessaging) SendMessageWithID(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error) {
	in.Header.Namespace = ns
	in.Header.Type = fftypes.MessageTypePrivate
	if in.Header.Author == "" {
		in.Header.Author = pm.localOrgIdentity
	}
	if in.Header.TxType == "" {
		in.Header.TxType = fftypes.TransactionTypeBatchPin
	}

	sender, err := pm.identity.Resolve(ctx, in.Header.Author)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
	}

	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin
	err = pm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return pm.resolveAndSend(ctx, sender, in)
	})
	if err != nil {
		return nil, err
	}
	if waitConfirm {
		// TODO: wait for confirmation
	}
	// The broadcastMessage function modifies the input message to create all the refs
	return &in.Message, err
}

func (pm *privateMessaging) resolveAndSend(ctx context.Context, sender *fftypes.Identity, in *fftypes.MessageInOut) (err error) {
	// Resolve the member list into a group
	if err = pm.resolveReceipientList(ctx, sender, in); err != nil {
		return err
	}

	// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
	in.Message.Data, err = pm.data.ResolveInlineDataPrivate(ctx, in.Header.Namespace, in.InlineData)
	if err != nil {
		return err
	}

	// Seal the message
	if err := in.Message.Seal(ctx); err != nil {
		return err
	}

	if in.Message.Header.TxType == fftypes.TransactionTypeNone {
		in.Message.Confirmed = fftypes.Now()
		in.Message.Pending = false
	}

	// Store the message - this asynchronously triggers the next step in process
	if err = pm.database.InsertMessageLocal(ctx, &in.Message); err != nil {
		return err
	}

	if in.Message.Header.TxType == fftypes.TransactionTypeNone {
		err = pm.sendUnpinnedMessage(ctx, &in.Message)
		if err != nil {
			return err
		}

		// Emit a confirmation event locally immediately
		event := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, in.Message.Header.Namespace, in.Message.Header.ID)
		return pm.database.InsertEvent(ctx, event)
	}

	return nil
}

func (pm *privateMessaging) sendUnpinnedMessage(ctx context.Context, message *fftypes.Message) (err error) {

	// Retrieve the group
	group, nodes, err := pm.groupManager.getGroupNodes(ctx, message.Header.Group)
	if err != nil {
		return err
	}

	id, err := pm.identity.Resolve(ctx, message.Header.Author)
	if err == nil {
		err = pm.blockchain.VerifyIdentitySyntax(ctx, id)
	}
	if err != nil {
		log.L(ctx).Errorf("Invalid signing identity '%s': %s", message.Header.Author, err)
		return err
	}

	data, _, err := pm.data.GetMessageData(ctx, message, true)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: message,
		Data:    data,
		Group:   group,
	})
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	return pm.sendData(ctx, "message", message.Header.ID, message.Header.Group, message.Header.Namespace, nodes, payload, nil, data)
}
