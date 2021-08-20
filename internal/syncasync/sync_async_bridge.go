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

type requestType int

const (
	messageConfirm requestType = iota
	messageReply
)

type inflightRequest struct {
	id        *fftypes.UUID
	startTime time.Time
	response  chan inflightResponse
	reqType   requestType
}
type inflightResponse struct {
	id   *fftypes.UUID
	data interface{}
	err  error
}

type inflightRequestMap map[string]map[fftypes.UUID]*inflightRequest

type syncAsyncBridge struct {
	ctx         context.Context
	database    database.Plugin
	data        data.Manager
	sysevents   sysmessaging.SystemEvents
	sender      sysmessaging.MessageSender
	inflightMux sync.Mutex
	inflight    inflightRequestMap
}

func NewSyncAsyncBridge(ctx context.Context, di database.Plugin, dm data.Manager) Bridge {
	sa := &syncAsyncBridge{
		ctx:      log.WithLogField(ctx, "role", "sync-async-bridge"),
		database: di,
		data:     dm,
		inflight: make(inflightRequestMap),
	}
	return sa
}

func (sa *syncAsyncBridge) Init(sysevents sysmessaging.SystemEvents, sender sysmessaging.MessageSender) {
	sa.sysevents = sysevents
	sa.sender = sender
}

func (sa *syncAsyncBridge) addInFlight(ns string, reqType requestType) (*inflightRequest, error) {
	inflight := &inflightRequest{
		id:        fftypes.NewUUID(),
		startTime: time.Now(),
		response:  make(chan inflightResponse),
		reqType:   reqType,
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

func (sa *syncAsyncBridge) getInFlight(ns string, reqType requestType, id *fftypes.UUID) *inflightRequest {
	inflightNS := sa.inflight[ns]
	if inflightNS != nil && id != nil {
		inflight := inflightNS[*id]
		if inflight != nil && inflight.reqType == reqType {
			return inflight
		}
	}
	return nil
}

func (sa *syncAsyncBridge) removeInFlight(ns string, id *fftypes.UUID) {
	sa.inflightMux.Lock()
	defer func() {
		sa.inflightMux.Unlock()
	}()
	inflightNS := sa.inflight[ns]
	if inflightNS != nil {
		delete(inflightNS, *id)
	}
}

func (inflight *inflightRequest) msInflight() float64 {
	dur := time.Since(inflight.startTime)
	return float64(dur) / float64(time.Millisecond)
}

func (sa *syncAsyncBridge) eventCallback(event *fftypes.EventDelivery) error {
	sa.inflightMux.Lock()
	defer sa.inflightMux.Unlock()

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

	switch event.Type {
	case fftypes.EventTypeMessageConfirmed:
		// See if the CID marks this as a reply to an inflight event
		inflightReply := sa.getInFlight(event.Namespace, messageReply, msg.Header.CID)
		if inflightReply != nil {
			go sa.resolveReply(inflightReply, msg)
		}

		// See if this is a confirmation of the delivery of an inflight event
		inflight := sa.getInFlight(event.Namespace, messageConfirm, msg.Header.ID)
		if inflight != nil {
			go sa.resolveConfirmed(inflight, msg)
		}

	case fftypes.EventTypeMessageRejected:
		// See if this is a rejection of an inflight event
		inflight := sa.getInFlight(event.Namespace, messageConfirm, msg.Header.ID)
		if inflight != nil {
			go sa.resolveRejected(inflight, msg)
		}
	}

	return nil
}

func (sa *syncAsyncBridge) resolveReply(inflight *inflightRequest, msg *fftypes.Message) {
	log.L(sa.ctx).Debugf("Resolving reply request '%s' with message '%s'", inflight.id, msg.Header.ID)

	response := &fftypes.MessageInOut{Message: *msg}
	data, _, err := sa.data.GetMessageData(sa.ctx, msg, true)
	if err != nil {
		log.L(sa.ctx).Errorf("Failed to read response data for message '%s' on request '%s': %s", msg.Header.ID, inflight.id, err)
		return
	}
	response.SetInlineData(data)
	inflight.response <- inflightResponse{id: msg.Header.ID, data: response}
}

func (sa *syncAsyncBridge) resolveConfirmed(inflight *inflightRequest, msg *fftypes.Message) {
	log.L(sa.ctx).Debugf("Resolving confirmation request '%s' with message '%s'", inflight.id, msg.Header.ID)
	inflight.response <- inflightResponse{id: msg.Header.ID, data: msg}
}

func (sa *syncAsyncBridge) resolveRejected(inflight *inflightRequest, msg *fftypes.Message) {
	err := i18n.NewError(sa.ctx, i18n.MsgRejected, msg.Header.ID)
	log.L(sa.ctx).Errorf("Resolving confirmation request '%s' with error: %s", inflight.id, err)
	inflight.response <- inflightResponse{err: err}
}

func (sa *syncAsyncBridge) sendAndWait(ctx context.Context, ns string, unresolved *fftypes.MessageInOut, resolved *fftypes.Message, reqType requestType) (reply interface{}, err error) {
	if unresolved != nil {
		resolved = &unresolved.Message
	}

	inflight, err := sa.addInFlight(ns, reqType)
	if err != nil {
		return nil, err
	}
	var replyID *fftypes.UUID
	defer func() {
		sa.removeInFlight(ns, inflight.id)
		if replyID != nil {
			log.L(sa.ctx).Infof("Request '%s' resolved with message '%s' after %.2fms", inflight.id, replyID, inflight.msInflight())
		} else {
			log.L(sa.ctx).Infof("Request '%s' resolved with timeout after %.2fms", inflight.id, inflight.msInflight())
		}
	}()

	resolved.Header.ID = inflight.id
	_, err = sa.sender.SendMessageWithID(ctx, ns, unresolved, resolved, false)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, i18n.NewError(ctx, i18n.MsgRequestTimeout, inflight.id, inflight.msInflight())
	case reply := <-inflight.response:
		replyID = reply.id
		return reply.data, reply.err
	}
}

func (sa *syncAsyncBridge) RequestReply(ctx context.Context, ns string, unresolved *fftypes.MessageInOut) (*fftypes.MessageInOut, error) {
	if unresolved.Header.Group == nil && (unresolved.Group == nil || len(unresolved.Group.Members) == 0) {
		return nil, i18n.NewError(ctx, i18n.MsgRequestMustBePrivate)
	}
	if unresolved.Header.Tag == "" {
		return nil, i18n.NewError(ctx, i18n.MsgRequestReplyTagRequired)
	}
	if unresolved.Header.CID != nil {
		return nil, i18n.NewError(ctx, i18n.MsgRequestCannotHaveCID)
	}

	reply, err := sa.sendAndWait(ctx, ns, unresolved, nil, messageReply)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.MessageInOut), err
}

func (sa *syncAsyncBridge) SendConfirm(ctx context.Context, msg *fftypes.Message) (*fftypes.Message, error) {
	reply, err := sa.sendAndWait(ctx, msg.Header.Namespace, nil, msg, messageConfirm)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.Message), err
}
