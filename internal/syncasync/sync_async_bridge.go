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
	RequestReply(ctx context.Context, ns string, request *fftypes.MessageInOut) (*fftypes.MessageInOut, error)
	// SendConfirm blocks until the message is confirmed (or rejected), but does not look for a reply.
	SendConfirm(ctx context.Context, request *fftypes.Message) (*fftypes.Message, error)
	// SendConfirmTokenPool blocks until the token pool is confirmed (or rejected)
	SendConfirmTokenPool(ctx context.Context, ns string, send RequestSender) (*fftypes.TokenPool, error)
}

type RequestSender func(requestID *fftypes.UUID) error

type requestType int

const (
	messageConfirm requestType = iota
	messageReply
	tokenPoolConfirm
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
	if len(inflightNS) == 0 {
		// No need to do any expensive lookups/matching - this could not be a match
		return nil
	}

	getMessage := func() (*fftypes.Message, error) {
		msg, err := sa.database.GetMessageByID(sa.ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		if msg == nil {
			// This should not happen (but we need to move on)
			log.L(sa.ctx).Errorf("Unable to resolve message '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
		}
		return msg, nil
	}

	getPool := func() (*fftypes.TokenPool, error) {
		pool, err := sa.database.GetTokenPoolByID(sa.ctx, event.Reference)
		if err != nil {
			return nil, err
		}
		if pool == nil {
			// This should not happen (but we need to move on)
			log.L(sa.ctx).Errorf("Unable to resolve token pool '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
		}
		return pool, nil
	}

	switch event.Type {
	case fftypes.EventTypeMessageConfirmed:
		msg, err := getMessage()
		if err != nil || msg == nil {
			return err
		}
		// See if the CID marks this as a reply to an inflight message
		inflightReply := sa.getInFlight(event.Namespace, messageReply, msg.Header.CID)
		if inflightReply != nil {
			go sa.resolveReply(inflightReply, msg)
		}

		// See if this is a confirmation of the delivery of an inflight message
		inflight := sa.getInFlight(event.Namespace, messageConfirm, msg.Header.ID)
		if inflight != nil {
			go sa.resolveConfirmed(inflight, msg)
		}

	case fftypes.EventTypeMessageRejected:
		msg, err := getMessage()
		if err != nil || msg == nil {
			return err
		}
		// See if this is a rejection of an inflight message
		inflight := sa.getInFlight(event.Namespace, messageConfirm, msg.Header.ID)
		if inflight != nil {
			go sa.resolveRejected(inflight, msg)
		}

	case fftypes.EventTypePoolConfirmed:
		pool, err := getPool()
		if err != nil || pool == nil {
			return err
		}
		// See if this is a confirmation of an inflight token pool
		inflight := sa.getInFlight(event.Namespace, tokenPoolConfirm, pool.ID)
		if inflight != nil {
			go sa.resolveConfirmedTokenPool(inflight, pool)
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

func (sa *syncAsyncBridge) resolveConfirmedTokenPool(inflight *inflightRequest, pool *fftypes.TokenPool) {
	log.L(sa.ctx).Debugf("Resolving confirmation request '%s' with pool '%s'", inflight.id, pool.ID)
	inflight.response <- inflightResponse{id: pool.ID, data: pool}
}

func (sa *syncAsyncBridge) sendAndWait(ctx context.Context, ns string, reqType requestType, send RequestSender) (interface{}, error) {
	inflight, err := sa.addInFlight(ns, reqType)
	if err != nil {
		return nil, err
	}
	log.L(sa.ctx).Infof("Inflight request '%s' added", inflight.id)
	var replyID *fftypes.UUID
	defer func() {
		sa.removeInFlight(ns, inflight.id)
		if replyID != nil {
			log.L(sa.ctx).Infof("Inflight request '%s' resolved with reply '%s' after %.2fms", inflight.id, replyID, inflight.msInflight())
		} else {
			log.L(sa.ctx).Infof("Inflight request '%s' resolved with timeout after %.2fms", inflight.id, inflight.msInflight())
		}
	}()

	err = send(inflight.id)
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
	if unresolved.Header.Tag == "" {
		return nil, i18n.NewError(ctx, i18n.MsgRequestReplyTagRequired)
	}
	if unresolved.Header.CID != nil {
		return nil, i18n.NewError(ctx, i18n.MsgRequestCannotHaveCID)
	}

	reply, err := sa.sendAndWait(ctx, ns, messageReply, func(requestID *fftypes.UUID) error {
		_, err := sa.sender.SendMessageWithID(ctx, ns, requestID, unresolved, &unresolved.Message, false)
		return err
	})
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.MessageInOut), err
}

func (sa *syncAsyncBridge) SendConfirm(ctx context.Context, msg *fftypes.Message) (*fftypes.Message, error) {
	reply, err := sa.sendAndWait(ctx, msg.Header.Namespace, messageConfirm, func(requestID *fftypes.UUID) error {
		_, err := sa.sender.SendMessageWithID(ctx, msg.Header.Namespace, requestID, nil, msg, false)
		return err
	})
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.Message), err
}

func (sa *syncAsyncBridge) SendConfirmTokenPool(ctx context.Context, ns string, send RequestSender) (*fftypes.TokenPool, error) {
	reply, err := sa.sendAndWait(ctx, ns, tokenPoolConfirm, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.TokenPool), err
}
