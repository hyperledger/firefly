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

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (bm *broadcastManager) NewBroadcast(ns string, in *fftypes.MessageInOut) sysmessaging.MessageSender {
	broadcast := &broadcastSender{
		mgr:       bm,
		namespace: ns,
		msg:       in,
	}
	broadcast.setDefaults()
	return broadcast
}

func (bm *broadcastManager) BroadcastMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error) {
	broadcast := bm.NewBroadcast(ns, in)
	if waitConfirm {
		err = broadcast.SendAndWait(ctx)
	} else {
		err = broadcast.Send(ctx)
	}
	return &in.Message, err
}

type broadcastSender struct {
	mgr       *broadcastManager
	namespace string
	msg       *fftypes.MessageInOut
	resolved  bool
}

type sendMethod int

const (
	methodPrepare sendMethod = iota
	methodSend
	methodSendAndWait
)

func (s *broadcastSender) Prepare(ctx context.Context) error {
	return s.resolveAndSend(ctx, methodPrepare)
}

func (s *broadcastSender) Send(ctx context.Context) error {
	return s.resolveAndSend(ctx, methodSend)
}

func (s *broadcastSender) SendAndWait(ctx context.Context) error {
	return s.resolveAndSend(ctx, methodSendAndWait)
}

func (s *broadcastSender) setDefaults() {
	s.msg.Header.ID = fftypes.NewUUID()
	s.msg.Header.Namespace = s.namespace
	if s.msg.Header.Type == "" {
		s.msg.Header.Type = fftypes.MessageTypeBroadcast
	}
	if s.msg.Header.TxType == "" {
		s.msg.Header.TxType = fftypes.TransactionTypeBatchPin
	}
}

func (s *broadcastSender) resolveAndSend(ctx context.Context, method sendMethod) error {
	sent := false

	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin)
	var dataToPublish []*fftypes.DataAndBlob
	err := s.mgr.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		if !s.resolved {
			if dataToPublish, err = s.resolve(ctx); err != nil {
				return err
			}
			s.resolved = true
		}

		// For the simple case where we have no data to publish and aren't waiting for blockchain confirmation,
		// insert the local message immediately within the same DB transaction.
		// Otherwise, break out of the DB transaction (since those operations could take multiple seconds).
		if len(dataToPublish) == 0 && method != methodSendAndWait {
			sent = true
			return s.sendInternal(ctx, method)
		}
		return nil
	})

	if err != nil || sent {
		return err
	}

	// Perform deferred processing
	if len(dataToPublish) > 0 {
		if err := s.mgr.publishBlobs(ctx, dataToPublish); err != nil {
			return err
		}
	}
	return s.sendInternal(ctx, method)
}

func (s *broadcastSender) resolve(ctx context.Context) ([]*fftypes.DataAndBlob, error) {
	// Resolve the sending identity
	if !s.isRootOrgBroadcast(ctx) {
		if err := s.mgr.identity.ResolveInputIdentity(ctx, &s.msg.Header.Identity); err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
		}
	}

	// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
	dataRefs, dataToPublish, err := s.mgr.data.ResolveInlineDataBroadcast(ctx, s.namespace, s.msg.InlineData)
	s.msg.Message.Data = dataRefs
	return dataToPublish, err
}

func (s *broadcastSender) sendInternal(ctx context.Context, method sendMethod) (err error) {
	if method == methodSendAndWait {
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
	return s.mgr.database.InsertMessageLocal(ctx, &s.msg.Message)
}

func (s *broadcastSender) isRootOrgBroadcast(ctx context.Context) bool {
	// Look into message to see if it contains a data item that is a root organization definition
	if s.msg.Header.Type == fftypes.MessageTypeDefinition {
		messageData, ok, err := s.mgr.data.GetMessageData(ctx, &s.msg.Message, true)
		if ok && err == nil {
			if len(messageData) > 0 {
				dataItem := messageData[0]
				if dataItem.Validator == fftypes.MessageTypeDefinition {
					var org *fftypes.Organization
					err := json.Unmarshal(dataItem.Value, &org)
					if err != nil {
						return false
					}
					if org != nil && org.Name != "" && org.ID != nil && org.Parent == "" {
						return true
					}
				}
			}
		}
	}
	return false
}
