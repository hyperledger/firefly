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
		err = pm.resolveMessage(ctx, sender, in)
		if err == nil && !waitConfirm {
			// We can safely optimize the send into the same DB transaction
			out, err = pm.sendOrWaitMessage(ctx, in, false)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	if waitConfirm {
		// perform the send and wait for the confirmation after closing the original DB transaction
		out, err = pm.sendOrWaitMessage(ctx, in, true)
	}
	return out, err
}

func (pm *privateMessaging) resolveMessage(ctx context.Context, sender *fftypes.Identity, in *fftypes.MessageInOut) (err error) {
	// Resolve the member list into a group
	if err = pm.resolveReceipientList(ctx, sender, in); err != nil {
		return err
	}

	// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
	in.Message.Data, err = pm.data.ResolveInlineDataPrivate(ctx, in.Header.Namespace, in.InlineData)
	return err
}

func (pm *privateMessaging) sendOrWaitMessage(ctx context.Context, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error) {

	immediateConfirm := in.Message.Header.TxType == fftypes.TransactionTypeNone

	if immediateConfirm {
		in.Message.Confirmed = fftypes.Now()
		in.Message.Pending = false
	}

	if immediateConfirm || !waitConfirm {

		// Seal the message
		if err := in.Message.Seal(ctx); err != nil {
			return nil, err
		}

		// Store the message - this asynchronously triggers the next step in process
		if err = pm.database.InsertMessageLocal(ctx, &in.Message); err != nil {
			return nil, err
		}

		if immediateConfirm {
			err = pm.sendUnpinnedMessage(ctx, &in.Message)
			if err != nil {
				return nil, err
			}

			// Emit a confirmation event locally immediately
			event := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, in.Message.Header.Namespace, in.Message.Header.ID)
			err = pm.database.InsertEvent(ctx, event)
		}

		return &in.Message, err

	}

	// Pass it to the sync-async handler to wait for the confirmation to come back in.
	// NOTE: Our caller makes sure we are not in a RunAsGroup (which would be bad)
	return pm.syncasync.SendConfirm(ctx, in.Header.Namespace, in)

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
