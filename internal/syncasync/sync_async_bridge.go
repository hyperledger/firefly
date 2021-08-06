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

package syncasync

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/firefly/internal/data"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/sysmessaging"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

// Bridge translates synchronous (HTTP API) calls, into asynchronously sending a
// message and blocking until a correlating response is received, or we hit a timeout.
type Bridge interface {
	// Init is required as there's a bi-directional relationship between private messaging and syncasync bridge
	Init(sysevents sysmessaging.SystemEvents, sender sysmessaging.MessageSender)
	// Request performs a request/reply exchange taking a message as input, and returning a message as a response
	// The input message must have a tag, and a group, to be routed appropriately.
	RequestReply(ctx context.Context, ns string, request *fftypes.MessageInOut) (reply *fftypes.MessageInOut, err error)
	// SendConfirm blocks until the message is confirmed (or rejected), but does not look for a reply.
	SendConfirm(ctx context.Context, request *fftypes.Message) (reply *fftypes.Message, err error)
}

type inflightRequest struct {
	id           *fftypes.UUID
	startTime    time.Time
	namespace    string
	response     chan interface{}
	waitForReply bool
	withData     bool
}

type syncAsyncBridge struct {
	ctx         context.Context
	database    database.Plugin
	data        data.Manager
	sysevents   sysmessaging.SystemEvents
	sender      sysmessaging.MessageSender
	inflightMux sync.Mutex
	inflight    map[string]map[fftypes.UUID]*inflightRequest
}

func NewSyncAsyncBridge(ctx context.Context, di database.Plugin, dm data.Manager) Bridge {
	sa := &syncAsyncBridge{
		ctx:      log.WithLogField(ctx, "role", "sync-async-bridge"),
		database: di,
		data:     dm,
		inflight: make(map[string]map[fftypes.UUID]*inflightRequest),
	}
	return sa
}

func (sa *syncAsyncBridge) Init(sysevents sysmessaging.SystemEvents, sender sysmessaging.MessageSender) {
	sa.sysevents = sysevents
	sa.sender = sender
}

func (sa *syncAsyncBridge) addInFlight(ns string, waitForReply, withData bool) (*inflightRequest, error) {
	inflight := &inflightRequest{
		id:           fftypes.NewUUID(),
		namespace:    ns,
		startTime:    time.Now(),
		response:     make(chan interface{}),
		waitForReply: waitForReply,
		withData:     withData,
	}
	sa.inflightMux.Lock()
	defer func() {
		sa.inflightMux.Unlock()
		log.L(sa.ctx).Infof("RequestReply '%s' added", inflight.id)
	}()

	inflightNS := sa.inflight[ns]
	if inflightNS == nil {
		err := sa.sysevents.AddSystemEventListener(ns, sa.eventCallback)
		if err != nil {
			return nil, err
		}
		inflightNS = make(map[fftypes.UUID]*inflightRequest)
		sa.inflight[ns] = inflightNS
	}
	inflightNS[*inflight.id] = inflight
	return inflight, nil
}

func (sa *syncAsyncBridge) removeInFlight(inflight *inflightRequest, replyID *fftypes.UUID) {
	sa.inflightMux.Lock()
	defer func() {
		sa.inflightMux.Unlock()
		if replyID != nil {
			log.L(sa.ctx).Infof("RequestReply '%s' resolved with message '%s' after %.2fms", inflight.id, replyID, inflight.msInflight())
		} else {
			log.L(sa.ctx).Infof("RequestReply '%s' resolved with timeout after %.2fms", inflight.id, inflight.msInflight())
		}
	}()
	inflightNS := sa.inflight[inflight.namespace]
	if inflightNS != nil {
		delete(inflightNS, *inflight.id)
	}
}

func (inflight *inflightRequest) msInflight() float64 {
	dur := time.Since(inflight.startTime)
	return float64(dur) / float64(time.Millisecond)
}

func (sa *syncAsyncBridge) eventCallback(event *fftypes.EventDelivery) error {
	sa.inflightMux.Lock()
	defer sa.inflightMux.Unlock()

	// Find the right set of potential callbacks
	inflightNS := sa.inflight[event.Namespace]
	if len(inflightNS) == 0 || (event.Type != fftypes.EventTypeMessageConfirmed && event.Type != fftypes.EventTypeMessageRejected) {
		// No need to do any expensive lookups/matching - this could not be a match
		return nil
	}

	// Look up the message
	msg, err := sa.database.GetMessageByID(sa.ctx, event.Reference)
	if err != nil {
		return err
	}
	if msg == nil {
		// This should not happen...
		log.L(sa.ctx).Errorf("Unable to resolve message '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
		return nil // ..but we need to move on
	}

	// See if the CID in the message matches an inflight events
	if msg.Header.CID != nil && event.Type == fftypes.EventTypeMessageConfirmed {
		inflight := inflightNS[*msg.Header.CID]
		if inflight != nil && inflight.waitForReply {
			// No need to block while locked any further here - kick off the data lookup and resolve
			go sa.resolveInflight(inflight, msg, inflight.withData, nil)
			return nil
		}
	}

	// See if this is a confirmation of the delivery of a message that's in flight
	inflight := inflightNS[*msg.Header.ID]
	if inflight != nil && !inflight.waitForReply {
		var replyErr error
		if event.Type == fftypes.EventTypeMessageRejected {
			replyErr = i18n.NewError(sa.ctx, i18n.MsgRejected, msg.Header.ID)
		}
		go sa.resolveInflight(inflight, msg, inflight.withData, replyErr)
	}

	return nil
}

func (sa *syncAsyncBridge) resolveInflight(inflight *inflightRequest, msg *fftypes.Message, withData bool, err error) {

	if err != nil {
		log.L(sa.ctx).Errorf("Resolving RequestReply '%s' with error: %s", inflight.id, err)
		inflight.response <- err
		return
	}

	log.L(sa.ctx).Debugf("Resolving RequestReply '%s' with message '%s'", inflight.id, msg.Header.ID)

	if withData {
		response := &fftypes.MessageInOut{Message: *msg}
		data, _, err := sa.data.GetMessageData(sa.ctx, msg, true)
		if err != nil {
			log.L(sa.ctx).Errorf("Failed to read response data for message '%s' on request '%s': %s", msg.Header.ID, inflight.id, err)
			return
		}
		response.SetInlineData(data)
		// We deliver the full data in the response
		inflight.response <- response
	} else {
		inflight.response <- msg
	}

}

func (sa *syncAsyncBridge) sendAndWait(ctx context.Context, ns string, unresolved *fftypes.MessageInOut, resolved *fftypes.Message, waitForReply, withData bool) (reply interface{}, err error) {
	if unresolved != nil {
		resolved = &unresolved.Message
	}
	if resolved.Header.Tag == "" {
		return nil, i18n.NewError(ctx, i18n.MsgRequestReplyTagRequired)
	}
	if resolved.Header.CID != nil {
		return nil, i18n.NewError(ctx, i18n.MsgRequestCannotHaveCID)
	}

	inflight, err := sa.addInFlight(ns, waitForReply, withData)
	if err != nil {
		return nil, err
	}
	var replyID *fftypes.UUID
	defer func() {
		sa.removeInFlight(inflight, replyID)
	}()

	resolved.Header.ID = inflight.id
	_, err = sa.sender.SendMessageWithID(ctx, ns, unresolved, resolved, false)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, i18n.NewError(ctx, i18n.MsgRequestTimeout, inflight.id, inflight.msInflight())
	case reply = <-inflight.response:
		switch rt := reply.(type) {
		case *fftypes.MessageInOut:
			replyID = rt.Message.Header.ID
		case *fftypes.Message:
			replyID = rt.Header.ID
		case error:
			return nil, rt
		}
		return reply, nil
	}
}

func (sa *syncAsyncBridge) RequestReply(ctx context.Context, ns string, unresolved *fftypes.MessageInOut) (*fftypes.MessageInOut, error) {
	reply, err := sa.sendAndWait(ctx, ns, unresolved, nil,
		true, // wait for a reply
		true, // reply will be MessageInOut
	)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.MessageInOut), err
}

func (sa *syncAsyncBridge) SendConfirm(ctx context.Context, msg *fftypes.Message) (*fftypes.Message, error) {
	reply, err := sa.sendAndWait(ctx, msg.Header.Namespace, nil, msg,
		false, // wait for confirmation, but not a reply
		false, // do not include the full data in the reply
	)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.Message), err
}
