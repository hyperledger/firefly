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

	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

// Bridge translates synchronous (HTTP API) calls, into asynchronously sending a
// message and blocking until a correlating response is received, or we hit a timeout.
type Bridge interface {
	// Init is required as there's a bi-directional relationship between sysmessaging and syncasync bridge
	Init(sysevents sysmessaging.SystemEvents)
	// Request performs a request/reply exchange taking a message as input, and returning a message as a response
	// The input message must have a tag, and a group, to be routed appropriately.
	RequestReply(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.MessageInOut, error)
	// SendConfirm blocks until the message is confirmed (or rejected), but does not look for a reply.
	SendConfirm(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.Message, error)
	// SendConfirmTokenPool blocks until the token pool is confirmed (or rejected)
	SendConfirmTokenPool(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenPool, error)
	// SendConfirmTokenTransfer blocks until the token transfer is confirmed
	SendConfirmTokenTransfer(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenTransfer, error)
}

type RequestSender func() error

type requestType int

const (
	messageConfirm requestType = iota
	messageReply
	tokenPoolConfirm
	tokenTransferConfirm
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

func (sa *syncAsyncBridge) Init(sysevents sysmessaging.SystemEvents) {
	sa.sysevents = sysevents
}

func (sa *syncAsyncBridge) addInFlight(ns string, id *fftypes.UUID, reqType requestType) (*inflightRequest, error) {
	inflight := &inflightRequest{
		id:        id,
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

func (sa *syncAsyncBridge) getMessageFromEvent(event *fftypes.EventDelivery) (*fftypes.Message, error) {
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

func (sa *syncAsyncBridge) getPoolFromEvent(event *fftypes.EventDelivery) (*fftypes.TokenPool, error) {
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

func (sa *syncAsyncBridge) getTransferFromEvent(event *fftypes.EventDelivery) (*fftypes.TokenTransfer, error) {
	transfer, err := sa.database.GetTokenTransfer(sa.ctx, event.Reference)
	if err != nil {
		return nil, err
	}
	if transfer == nil {
		// This should not happen (but we need to move on)
		log.L(sa.ctx).Errorf("Unable to resolve token transfer '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
	}
	return transfer, nil
}

func (sa *syncAsyncBridge) eventCallback(event *fftypes.EventDelivery) error {
	sa.inflightMux.Lock()
	defer sa.inflightMux.Unlock()

	inflightNS := sa.inflight[event.Namespace]
	if len(inflightNS) == 0 {
		// No need to do any expensive lookups/matching - this could not be a match
		return nil
	}

	switch event.Type {
	case fftypes.EventTypeMessageConfirmed:
		msg, err := sa.getMessageFromEvent(event)
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
		msg, err := sa.getMessageFromEvent(event)
		if err != nil || msg == nil {
			return err
		}
		// See if this is a rejection of an inflight message
		inflight := sa.getInFlight(event.Namespace, messageConfirm, msg.Header.ID)
		if inflight != nil {
			go sa.resolveRejected(inflight, msg)
		}

	case fftypes.EventTypePoolConfirmed:
		pool, err := sa.getPoolFromEvent(event)
		if err != nil || pool == nil {
			return err
		}
		// See if this is a confirmation of an inflight token pool
		inflight := sa.getInFlight(event.Namespace, tokenPoolConfirm, pool.ID)
		if inflight != nil {
			go sa.resolveConfirmedTokenPool(inflight, pool)
		}

	case fftypes.EventTypePoolRejected:
		pool, err := sa.getPoolFromEvent(event)
		if err != nil || pool == nil {
			return err
		}
		// See if this is a rejection of an inflight token pool
		inflight := sa.getInFlight(event.Namespace, tokenPoolConfirm, pool.ID)
		if inflight != nil {
			go sa.resolveRejectedTokenPool(inflight, pool)
		}

	case fftypes.EventTypeTransferConfirmed:
		transfer, err := sa.getTransferFromEvent(event)
		if err != nil || transfer == nil {
			return err
		}
		// See if this is a confirmation of an inflight token transfer
		inflight := sa.getInFlight(event.Namespace, tokenTransferConfirm, transfer.LocalID)
		if inflight != nil {
			go sa.resolveConfirmedTokenTransfer(inflight, transfer)
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
	log.L(sa.ctx).Debugf("Resolving message confirmation request '%s' with ID '%s'", inflight.id, msg.Header.ID)
	inflight.response <- inflightResponse{id: msg.Header.ID, data: msg}
}

func (sa *syncAsyncBridge) resolveRejected(inflight *inflightRequest, msg *fftypes.Message) {
	err := i18n.NewError(sa.ctx, i18n.MsgRejected, msg.Header.ID)
	log.L(sa.ctx).Errorf("Resolving message confirmation request '%s' with error: %s", inflight.id, err)
	inflight.response <- inflightResponse{err: err}
}

func (sa *syncAsyncBridge) resolveConfirmedTokenPool(inflight *inflightRequest, pool *fftypes.TokenPool) {
	log.L(sa.ctx).Debugf("Resolving token pool confirmation request '%s' with ID '%s'", inflight.id, pool.ID)
	inflight.response <- inflightResponse{id: pool.ID, data: pool}
}

func (sa *syncAsyncBridge) resolveRejectedTokenPool(inflight *inflightRequest, pool *fftypes.TokenPool) {
	err := i18n.NewError(sa.ctx, i18n.MsgTokenPoolRejected, pool.ID)
	log.L(sa.ctx).Errorf("Resolving token pool confirmation request '%s' with error '%s'", inflight.id, err)
	inflight.response <- inflightResponse{err: err}
}

func (sa *syncAsyncBridge) resolveConfirmedTokenTransfer(inflight *inflightRequest, transfer *fftypes.TokenTransfer) {
	log.L(sa.ctx).Debugf("Resolving token transfer confirmation request '%s' with ID '%s'", inflight.id, transfer.LocalID)
	inflight.response <- inflightResponse{id: transfer.LocalID, data: transfer}
}

func (sa *syncAsyncBridge) sendAndWait(ctx context.Context, ns string, id *fftypes.UUID, reqType requestType, send RequestSender) (interface{}, error) {
	inflight, err := sa.addInFlight(ns, id, reqType)
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

	err = send()
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

func (sa *syncAsyncBridge) RequestReply(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.MessageInOut, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, messageReply, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.MessageInOut), err
}

func (sa *syncAsyncBridge) SendConfirm(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.Message, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, messageConfirm, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.Message), err
}

func (sa *syncAsyncBridge) SendConfirmTokenPool(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenPool, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, tokenPoolConfirm, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.TokenPool), err
}

func (sa *syncAsyncBridge) SendConfirmTokenTransfer(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenTransfer, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, tokenTransferConfirm, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.TokenTransfer), err
}
