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

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (pm *privateMessaging) SendMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error) {
	if err := pm.prepareMessage(ctx, ns, in); err != nil {
		return nil, err
	}
	return pm.resolveAndSend(ctx, in, nil, waitConfirm)
}

func (pm *privateMessaging) RequestReply(ctx context.Context, ns string, in *fftypes.MessageInOut) (*fftypes.MessageInOut, error) {
	if in.Header.Tag == "" {
		return nil, i18n.NewError(ctx, i18n.MsgRequestReplyTagRequired)
	}
	if in.Header.CID != nil {
		return nil, i18n.NewError(ctx, i18n.MsgRequestCannotHaveCID)
	}
	if err := pm.prepareMessage(ctx, ns, in); err != nil {
		return nil, err
	}
	return pm.syncasync.RequestReply(ctx, ns, in.Header.ID, func() error {
		_, err := pm.resolveAndSend(ctx, in, &in.Message, false)
		return err
	})
}

func (pm *privateMessaging) prepareMessage(ctx context.Context, ns string, msg *fftypes.MessageInOut) error {
	msg.Header.ID = fftypes.NewUUID()
	msg.Header.Namespace = ns
	if msg.Header.Type == "" {
		msg.Header.Type = fftypes.MessageTypePrivate
	}
	if msg.Header.TxType == "" {
		msg.Header.TxType = fftypes.TransactionTypeBatchPin
	}

	// Resolve the sending identity
	if err := pm.identity.ResolveInputIdentity(ctx, &msg.Header.Identity); err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
	}

	return nil
}

func (pm *privateMessaging) resolveAndSend(ctx context.Context, unresolved *fftypes.MessageInOut, resolved *fftypes.Message, waitConfirm bool) (out *fftypes.Message, err error) {
	if unresolved != nil {
		resolved = &unresolved.Message
	}

	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin)
	sent := false
	err = pm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		if unresolved != nil {
			err = pm.resolveMessage(ctx, unresolved)
			if err != nil {
				return err
			}
		}

		// If we aren't waiting for blockchain confirmation, insert the local message immediately within the same DB transaction.
		if !waitConfirm {
			out, err = pm.sendMessageAsync(ctx, resolved)
			sent = true
		}
		return err
	})

	if err != nil || sent {
		return out, err
	}

	return pm.sendMessageCommon(ctx, resolved, waitConfirm)
}

func (pm *privateMessaging) resolveMessage(ctx context.Context, in *fftypes.MessageInOut) (err error) {
	// Resolve the member list into a group
	if err = pm.resolveReceipientList(ctx, in); err != nil {
		return err
	}

	// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
	in.Message.Data, err = pm.data.ResolveInlineDataPrivate(ctx, in.Header.Namespace, in.InlineData)
	return err
}

func (pm *privateMessaging) sendMessageCommon(ctx context.Context, msg *fftypes.Message, waitConfirm bool) (*fftypes.Message, error) {
	if waitConfirm && msg.Header.TxType != fftypes.TransactionTypeNone {
		return pm.sendMessageSync(ctx, msg)
	}
	return pm.sendMessageAsync(ctx, msg)
}

func (pm *privateMessaging) sendMessageAsync(ctx context.Context, msg *fftypes.Message) (*fftypes.Message, error) {
	immediateConfirm := msg.Header.TxType == fftypes.TransactionTypeNone

	// Seal the message
	if err := msg.Seal(ctx); err != nil {
		return nil, err
	}

	if immediateConfirm {
		msg.Confirmed = fftypes.Now()
		msg.Pending = false
		// msg.Header.Key = "" // there is no on-chain signing assurance with this message
	}

	// Store the message - this asynchronously triggers the next step in process
	if err := pm.database.InsertMessageLocal(ctx, msg); err != nil {
		return nil, err
	}

	if immediateConfirm {
		if err := pm.sendUnpinnedMessage(ctx, msg); err != nil {
			return nil, err
		}

		// Emit a confirmation event locally immediately
		event := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, msg.Header.Namespace, msg.Header.ID)
		if err := pm.database.InsertEvent(ctx, event); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func (pm *privateMessaging) sendMessageSync(ctx context.Context, msg *fftypes.Message) (*fftypes.Message, error) {
	// Pass it to the sync-async handler to wait for the confirmation to come back in.
	// NOTE: Our caller makes sure we are not in a RunAsGroup (which would be bad)
	return pm.syncasync.SendConfirm(ctx, msg.Header.Namespace, msg.Header.ID, func() error {
		_, err := pm.resolveAndSend(ctx, nil, msg, false)
		return err
	})
}

func (pm *privateMessaging) sendUnpinnedMessage(ctx context.Context, message *fftypes.Message) (err error) {

	// Retrieve the group
	group, nodes, err := pm.groupManager.getGroupNodes(ctx, message.Header.Group)
	if err != nil {
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
