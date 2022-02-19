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

package privatemessaging

import (
	"context"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/database"
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
	if pm.metrics.IsMetricsEnabled() {
		pm.metrics.MessageSubmitted(&in.Message)
	}
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
	return pm.syncasync.WaitForReply(ctx, ns, in.Header.ID, message.Send)
}

// sendMethod is the specific operation requested of the messageSender.
// To minimize duplication and group database operations, there is a single internal flow with subtle differences for each method.
type messageSender struct {
	mgr       *privateMessaging
	namespace string
	msg       *fftypes.MessageInOut
	resolved  bool
}

type sendMethod int

const (
	// methodPrepare requests that the message be validated and sealed, but not sent (i.e. no database writes are performed)
	methodPrepare sendMethod = iota
	// methodSend requests that the message be sent and pinned to the blockchain, but does not wait for confirmation
	methodSend
	// methodSendAndWait requests that the message be sent and waits until it is pinned and confirmed by the blockchain
	methodSendAndWait
)

func (s *messageSender) Prepare(ctx context.Context) error {
	return s.resolveAndSend(ctx, methodPrepare)
}

func (s *messageSender) Send(ctx context.Context) error {
	return s.resolveAndSend(ctx, methodSend)
}

func (s *messageSender) SendAndWait(ctx context.Context) error {
	return s.resolveAndSend(ctx, methodSendAndWait)
}

func (s *messageSender) setDefaults() {
	s.msg.Header.ID = fftypes.NewUUID()
	s.msg.Header.Namespace = s.namespace
	s.msg.State = fftypes.MessageStateReady
	if s.msg.Header.Type == "" {
		s.msg.Header.Type = fftypes.MessageTypePrivate
	}
	switch s.msg.Header.TxType {
	case fftypes.TransactionTypeUnpinned, fftypes.TransactionTypeNone:
		// "unpinned" used to be called "none" (before we introduced batching + a TX on unppinned sends)
		s.msg.Header.TxType = fftypes.TransactionTypeUnpinned
	default:
		// the only other valid option is "batch_pin"
		s.msg.Header.TxType = fftypes.TransactionTypeBatchPin
	}
}

func (s *messageSender) resolveAndSend(ctx context.Context, method sendMethod) error {
	sent := false

	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin)
	err := s.mgr.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		if !s.resolved {
			if err := s.resolve(ctx); err != nil {
				return err
			}
			msgSizeEstimate := s.msg.EstimateSize(true)
			if msgSizeEstimate > s.mgr.maxBatchPayloadLength {
				return i18n.NewError(ctx, i18n.MsgTooLargePrivate, float64(msgSizeEstimate)/1024, float64(s.mgr.maxBatchPayloadLength)/1024)
			}
			s.resolved = true
		}

		// If we aren't waiting for blockchain confirmation, insert the local message immediately within the same DB transaction.
		if method != methodSendAndWait {
			err = s.sendInternal(ctx, method)
			sent = true
		}
		return err
	})

	if err != nil || sent {
		return err
	}

	return s.sendInternal(ctx, method)
}

func (s *messageSender) resolve(ctx context.Context) error {
	// Resolve the sending identity
	if err := s.mgr.identity.ResolveInputIdentity(ctx, &s.msg.Header.SignerRef); err != nil {
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

func (s *messageSender) sendInternal(ctx context.Context, method sendMethod) error {
	if method == methodSendAndWait {
		// Pass it to the sync-async handler to wait for the confirmation to come back in.
		// NOTE: Our caller makes sure we are not in a RunAsGroup (which would be bad)
		out, err := s.mgr.syncasync.WaitForMessage(ctx, s.namespace, s.msg.Header.ID, s.Send)
		if out != nil {
			s.msg.Message = *out
		}
		return err
	}

	// Seal the message
	if err := s.msg.Seal(ctx); err != nil {
		return err
	}
	if method == methodPrepare {
		return nil
	}

	// Store the message - this asynchronously triggers the next step in process
	if err := s.mgr.database.UpsertMessage(ctx, &s.msg.Message, database.UpsertOptimizationNew); err != nil {
		return err
	}
	log.L(ctx).Infof("Sent private message %s:%s sequence=%d", s.msg.Header.Namespace, s.msg.Header.ID, s.msg.Sequence)

	return nil
}
