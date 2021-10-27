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
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (pm *privateMessaging) NewMessage(ns string, in *fftypes.MessageInOut) sysmessaging.MessageSender {
	message := &messageSender{
		mgr:       pm,
		namespace: ns,
		msg:       in,
	}
	message.setDefaults()
	return message
}

func (pm *privateMessaging) SendMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error) {
	message := pm.NewMessage(ns, in)
	if waitConfirm {
		err = message.SendAndWait(ctx)
	} else {
		err = message.Send(ctx)
	}
	return &in.Message, err
}

func (pm *privateMessaging) RequestReply(ctx context.Context, ns string, in *fftypes.MessageInOut) (*fftypes.MessageInOut, error) {
	if in.Header.Tag == "" {
		return nil, i18n.NewError(ctx, i18n.MsgRequestReplyTagRequired)
	}
	if in.Header.CID != nil {
		return nil, i18n.NewError(ctx, i18n.MsgRequestCannotHaveCID)
	}
	message := pm.NewMessage(ns, in)
	return pm.syncasync.RequestReply(ctx, ns, in.Header.ID, message.Send)
}

type messageSender struct {
	mgr          *privateMessaging
	namespace    string
	msg          *fftypes.MessageInOut
	resolved     bool
	sendCallback sysmessaging.BeforeSendCallback
}

func (s *messageSender) Send(ctx context.Context) error {
	return s.resolveAndSend(ctx, false)
}

func (s *messageSender) SendAndWait(ctx context.Context) error {
	return s.resolveAndSend(ctx, true)
}

func (s *messageSender) BeforeSend(cb sysmessaging.BeforeSendCallback) sysmessaging.MessageSender {
	s.sendCallback = cb
	return s
}

func (s *messageSender) setDefaults() {
	s.msg.Header.ID = fftypes.NewUUID()
	s.msg.Header.Namespace = s.namespace
	s.msg.State = fftypes.MessageStateReady
	if s.msg.Header.Type == "" {
		s.msg.Header.Type = fftypes.MessageTypePrivate
	}
	if s.msg.Header.TxType == "" {
		s.msg.Header.TxType = fftypes.TransactionTypeBatchPin
	}
}

func (s *messageSender) resolveAndSend(ctx context.Context, waitConfirm bool) error {
	sent := false

	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin)
	err := s.mgr.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		if !s.resolved {
			if err := s.resolveMessage(ctx); err != nil {
				return err
			}
			s.resolved = true
		}

		// If we aren't waiting for blockchain confirmation, insert the local message immediately within the same DB transaction.
		if !waitConfirm {
			err = s.sendInternal(ctx, waitConfirm)
			sent = true
		}
		return err
	})

	if err != nil || sent {
		return err
	}

	return s.sendInternal(ctx, waitConfirm)
}

func (s *messageSender) resolveMessage(ctx context.Context) error {
	// Resolve the sending identity
	if err := s.mgr.identity.ResolveInputIdentity(ctx, &s.msg.Header.Identity); err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
	}

	// Resolve the member list into a group
	if err := s.mgr.resolveRecipientList(ctx, s.msg); err != nil {
		return err
	}

	// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
	dataRefs, err := s.mgr.data.ResolveInlineDataPrivate(ctx, s.namespace, s.msg.InlineData)
	s.msg.Message.Data = dataRefs
	return err
}

func (s *messageSender) sendInternal(ctx context.Context, waitConfirm bool) error {
	immediateConfirm := s.msg.Header.TxType == fftypes.TransactionTypeNone

	if waitConfirm && !immediateConfirm {
		// Pass it to the sync-async handler to wait for the confirmation to come back in.
		// NOTE: Our caller makes sure we are not in a RunAsGroup (which would be bad)
		out, err := s.mgr.syncasync.SendConfirm(ctx, s.namespace, s.msg.Header.ID, s.Send)
		if out != nil {
			s.msg.Message = *out
		}
		return err
	}

	// Seal the message
	if err := s.msg.Seal(ctx); err != nil {
		return err
	}
	if s.sendCallback != nil {
		if err := s.sendCallback(ctx); err != nil {
			return err
		}
	}

	if immediateConfirm {
		s.msg.Confirmed = fftypes.Now()
		// msg.Header.Key = "" // there is no on-chain signing assurance with this message
	}

	// Store the message - this asynchronously triggers the next step in process
	if err := s.mgr.database.UpsertMessage(ctx, &s.msg.Message, false, false); err != nil {
		return err
	}

	if immediateConfirm {
		if err := s.sendUnpinned(ctx); err != nil {
			return err
		}

		// Emit a confirmation event locally immediately
		event := fftypes.NewEvent(fftypes.EventTypeMessageConfirmed, s.namespace, s.msg.Header.ID)
		if err := s.mgr.database.InsertEvent(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

func (s *messageSender) sendUnpinned(ctx context.Context) (err error) {
	// Retrieve the group
	group, nodes, err := s.mgr.groupManager.getGroupNodes(ctx, s.msg.Header.Group)
	if err != nil {
		return err
	}

	data, _, err := s.mgr.data.GetMessageData(ctx, &s.msg.Message, true)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: &s.msg.Message,
		Data:    data,
		Group:   group,
	})
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	return s.mgr.sendData(ctx, "message", s.msg.Header.ID, s.msg.Header.Group, s.namespace, nodes, payload, nil, data)
}
