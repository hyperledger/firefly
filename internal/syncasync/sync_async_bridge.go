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

package syncasync

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

// Bridge translates synchronous (HTTP API) calls, into asynchronously sending a
// message and blocking until a correlating response is received, or we hit a timeout.
type Bridge interface {
	// Init is required as there's a bi-directional relationship between sysmessaging and syncasync bridge
	Init(sysevents sysmessaging.SystemEvents)

	// The following "WaitFor*" methods all wait for a particular type of event callback, and block until it is received.
	// To use them, invoke the appropriate method, and pass a "send" callback that is expected to trigger the relevant event.

	// WaitForReply waits for a reply to the message with the supplied ID
	WaitForReply(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.MessageInOut, error)
	// WaitForMessage waits for a message with the supplied ID
	WaitForMessage(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.Message, error)
	// WaitForIdentity waits for an identity with the supplied ID
	WaitForIdentity(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.Identity, error)
	// WaitForTokenPool waits for a token pool with the supplied ID
	WaitForTokenPool(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenPool, error)
	// WaitForTokenTransfer waits for a token transfer with the supplied ID
	WaitForTokenTransfer(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenTransfer, error)
	// WaitForTokenTransfer waits for a token approval with the supplied ID
	WaitForTokenApproval(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenApproval, error)
}

type RequestSender func(ctx context.Context) error

type requestType int

const (
	messageConfirm requestType = iota
	messageReply
	identityConfirm
	tokenPoolConfirm
	tokenTransferConfirm
	tokenApproveConfirm
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
	if id == nil {
		return nil
	}
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

func (sa *syncAsyncBridge) getMessageFromEvent(event *fftypes.EventDelivery) (msg *fftypes.Message, err error) {
	if msg, err = sa.database.GetMessageByID(sa.ctx, event.Reference); err != nil {
		return nil, err
	}
	if msg == nil {
		// This should not happen (but we need to move on)
		log.L(sa.ctx).Errorf("Unable to resolve message '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
	}
	return msg, nil
}

func (sa *syncAsyncBridge) getIdentityFromEvent(event *fftypes.EventDelivery) (identity *fftypes.Identity, err error) {
	if identity, err = sa.database.GetIdentityByID(sa.ctx, event.Reference); err != nil {
		return nil, err
	}
	if identity == nil {
		// This should not happen (but we need to move on)
		log.L(sa.ctx).Errorf("Unable to resolve identity '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
	}
	return identity, nil
}

func (sa *syncAsyncBridge) getPoolFromEvent(event *fftypes.EventDelivery) (pool *fftypes.TokenPool, err error) {
	if pool, err = sa.database.GetTokenPoolByID(sa.ctx, event.Reference); err != nil {
		return nil, err
	}
	if pool == nil {
		// This should not happen (but we need to move on)
		log.L(sa.ctx).Errorf("Unable to resolve token pool '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
	}
	return pool, nil
}

func (sa *syncAsyncBridge) getPoolFromMessage(msg *fftypes.Message) (*fftypes.TokenPool, error) {
	if len(msg.Data) > 0 {
		data, err := sa.database.GetDataByID(sa.ctx, msg.Data[0].ID, true)
		if err != nil || data == nil {
			return nil, err
		}
		var pool fftypes.TokenPoolAnnouncement
		if err := json.Unmarshal(data.Value.Bytes(), &pool); err == nil {
			return pool.Pool, nil
		}
	}
	return nil, nil
}

func (sa *syncAsyncBridge) getTransferFromEvent(event *fftypes.EventDelivery) (transfer *fftypes.TokenTransfer, err error) {
	if transfer, err = sa.database.GetTokenTransfer(sa.ctx, event.Reference); err != nil {
		return nil, err
	}
	if transfer == nil {
		// This should not happen (but we need to move on)
		log.L(sa.ctx).Errorf("Unable to resolve token transfer '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
	}
	return transfer, nil
}

func (sa *syncAsyncBridge) getApprovalFromEvent(event *fftypes.EventDelivery) (approval *fftypes.TokenApproval, err error) {
	if approval, err = sa.database.GetTokenApproval(sa.ctx, event.Reference); err != nil {
		return nil, err
	}

	if approval == nil {
		// This should not happen (but we need to move on)
		log.L(sa.ctx).Errorf("Unable to resolve token approval '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
	}
	return approval, nil
}

func (sa *syncAsyncBridge) getOperationFromEvent(event *fftypes.EventDelivery) (op *fftypes.Operation, err error) {
	if op, err = sa.database.GetOperationByID(sa.ctx, event.Reference); err != nil {
		return nil, err
	}
	if op == nil {
		// This should not happen (but we need to move on)
		log.L(sa.ctx).Errorf("Unable to resolve operation '%s' for %s event '%s'", event.Reference, event.Type, event.ID)
	}
	return op, nil
}

func (sa *syncAsyncBridge) handleMessageConfirmedEvent(event *fftypes.EventDelivery) error {

	// See if the CID marks this as a reply to an inflight message
	inflight := sa.getInFlight(event.Namespace, messageConfirm, event.Reference)
	inflightReply := sa.getInFlight(event.Namespace, messageReply, event.Correlator)

	if inflightReply == nil && inflight == nil {
		return nil
	}

	msg, err := sa.getMessageFromEvent(event)
	if err != nil || msg == nil {
		return err
	}

	if inflightReply != nil {
		go sa.resolveReply(inflightReply, msg)
	}
	if inflight != nil {
		go sa.resolveConfirmed(inflight, msg)
	}

	return nil
}

func (sa *syncAsyncBridge) handleMessageRejectedEvent(event *fftypes.EventDelivery) error {

	// See if this is a rejection of an inflight message
	inflight := sa.getInFlight(event.Namespace, messageConfirm, event.Reference)
	inflightPool := sa.getInFlight(event.Namespace, tokenPoolConfirm, event.Correlator)

	if inflight == nil && inflightPool == nil {
		return nil
	}

	msg, err := sa.getMessageFromEvent(event)
	if err != nil || msg == nil {
		return err
	}
	if inflight != nil {
		go sa.resolveRejected(inflight, msg.Header.ID)
	}

	// See if this is a rejection of an inflight token pool
	if inflightPool != nil {
		if pool, err := sa.getPoolFromMessage(msg); err != nil {
			return err
		} else if pool != nil {
			go sa.resolveRejectedTokenPool(inflightPool, pool.ID)
		}
	}

	return nil
}

func (sa *syncAsyncBridge) handleIdentityConfirmedEvent(event *fftypes.EventDelivery) error {
	// See if the CID marks this as a reply to an inflight identity
	inflightReply := sa.getInFlight(event.Namespace, identityConfirm, event.Reference)
	if inflightReply == nil {
		return nil
	}

	identity, err := sa.getIdentityFromEvent(event)
	if err != nil || identity == nil {
		return err
	}

	go sa.resolveIdentity(inflightReply, identity)

	return nil
}

func (sa *syncAsyncBridge) handlePoolConfirmedEvent(event *fftypes.EventDelivery) error {
	// See if this is a confirmation of an inflight token pool
	inflight := sa.getInFlight(event.Namespace, tokenPoolConfirm, event.Reference)
	if inflight == nil {
		return nil
	}

	pool, err := sa.getPoolFromEvent(event)
	if err != nil || pool == nil {
		return err
	}

	go sa.resolveConfirmedTokenPool(inflight, pool)

	return nil
}

func (sa *syncAsyncBridge) handleTransferConfirmedEvent(event *fftypes.EventDelivery) error {
	// See if this is a confirmation of an inflight token transfer
	inflight := sa.getInFlight(event.Namespace, tokenTransferConfirm, event.Reference)
	if inflight == nil {
		return nil
	}

	transfer, err := sa.getTransferFromEvent(event)
	if err != nil || transfer == nil {
		return err
	}

	go sa.resolveConfirmedTokenTransfer(inflight, transfer)

	return nil
}

func (sa *syncAsyncBridge) handleApprovalConfirmedEvent(event *fftypes.EventDelivery) error {

	// See if this is a confirmation of an inflight token approval
	inflight := sa.getInFlight(event.Namespace, tokenApproveConfirm, event.Reference)
	if inflight == nil {
		return nil
	}

	approval, err := sa.getApprovalFromEvent(event)
	if err != nil || approval == nil {
		return err
	}

	go sa.resolveConfirmedTokenApproval(inflight, approval)

	return nil
}

func (sa *syncAsyncBridge) handleTransferOpFailedEvent(event *fftypes.EventDelivery) error {
	// See if this is a failure of an inflight token transfer operation
	inflight := sa.getInFlight(event.Namespace, tokenTransferConfirm, event.Correlator)
	if inflight == nil {
		return nil
	}

	op, err := sa.getOperationFromEvent(event)
	if err != nil || op == nil {
		return err
	}
	// Extract the LocalID of the transfer
	var transfer fftypes.TokenTransfer
	if err := txcommon.RetrieveTokenTransferInputs(sa.ctx, op, &transfer); err != nil {
		log.L(sa.ctx).Warnf("Failed to extract token transfer inputs for operation '%s': %s", op.ID, err)
	}

	go sa.resolveFailedTokenTransfer(inflight, transfer.LocalID)

	return nil
}

func (sa *syncAsyncBridge) handleApprovalOpFailedEvent(event *fftypes.EventDelivery) error {
	// See if this is a failure of an inflight token transfer operation
	inflight := sa.getInFlight(event.Namespace, tokenApproveConfirm, event.Correlator)
	if inflight == nil {
		return nil
	}

	op, err := sa.getOperationFromEvent(event)
	if err != nil || op == nil {
		return err
	}
	// Extract the LocalID of the transfer
	var approval fftypes.TokenApproval
	if err := txcommon.RetrieveTokenApprovalInputs(sa.ctx, op, &approval); err != nil {
		log.L(sa.ctx).Warnf("Failed to extract token approval inputs for operation '%s': %s", op.ID, err)
	}

	go sa.resolveFailedTokenTransfer(inflight, approval.LocalID)

	return nil
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
		return sa.handleMessageConfirmedEvent(event)

	case fftypes.EventTypeMessageRejected:
		return sa.handleMessageRejectedEvent(event)

	case fftypes.EventTypePoolConfirmed:
		return sa.handlePoolConfirmedEvent(event)

	case fftypes.EventTypeTransferConfirmed:
		return sa.handleTransferConfirmedEvent(event)

	case fftypes.EventTypeApprovalConfirmed:
		return sa.handleApprovalConfirmedEvent(event)

	case fftypes.EventTypeTransferOpFailed:
		return sa.handleTransferOpFailedEvent(event)

	case fftypes.EventTypeApprovalOpFailed:
		return sa.handleApprovalOpFailedEvent(event)

	case fftypes.EventTypeIdentityConfirmed:
		return sa.handleIdentityConfirmedEvent(event)
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

func (sa *syncAsyncBridge) resolveRejected(inflight *inflightRequest, msgID *fftypes.UUID) {
	err := i18n.NewError(sa.ctx, i18n.MsgRejected, msgID)
	log.L(sa.ctx).Errorf("Resolving message confirmation request '%s' with error: %s", inflight.id, err)
	inflight.response <- inflightResponse{err: err}
}

func (sa *syncAsyncBridge) resolveIdentity(inflight *inflightRequest, identity *fftypes.Identity) {
	log.L(sa.ctx).Debugf("Resolving identity creation '%s' with ID '%s'", inflight.id, identity.ID)
	inflight.response <- inflightResponse{id: identity.ID, data: identity}
}

func (sa *syncAsyncBridge) resolveConfirmedTokenPool(inflight *inflightRequest, pool *fftypes.TokenPool) {
	log.L(sa.ctx).Debugf("Resolving token pool confirmation request '%s' with ID '%s'", inflight.id, pool.ID)
	inflight.response <- inflightResponse{id: pool.ID, data: pool}
}

func (sa *syncAsyncBridge) resolveRejectedTokenPool(inflight *inflightRequest, poolID *fftypes.UUID) {
	err := i18n.NewError(sa.ctx, i18n.MsgTokenPoolRejected, poolID)
	log.L(sa.ctx).Errorf("Resolving token pool confirmation request '%s' with error '%s'", inflight.id, err)
	inflight.response <- inflightResponse{err: err}
}

func (sa *syncAsyncBridge) resolveConfirmedTokenTransfer(inflight *inflightRequest, transfer *fftypes.TokenTransfer) {
	log.L(sa.ctx).Debugf("Resolving token transfer confirmation request '%s' with ID '%s'", inflight.id, transfer.LocalID)
	inflight.response <- inflightResponse{id: transfer.LocalID, data: transfer}
}

func (sa *syncAsyncBridge) resolveConfirmedTokenApproval(inflight *inflightRequest, approval *fftypes.TokenApproval) {
	log.L(sa.ctx).Debugf("Resolving token approval confirmation request '%s' with ID '%s'", inflight.id, approval.LocalID)
	inflight.response <- inflightResponse{id: approval.LocalID, data: approval}
}

func (sa *syncAsyncBridge) resolveFailedTokenTransfer(inflight *inflightRequest, transferID *fftypes.UUID) {
	err := i18n.NewError(sa.ctx, i18n.MsgTokenTransferFailed, transferID)
	log.L(sa.ctx).Debugf("Resolving token transfer confirmation request '%s' with error '%s'", inflight.id, err)
	inflight.response <- inflightResponse{err: err}
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

	err = send(ctx)
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

func (sa *syncAsyncBridge) WaitForReply(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.MessageInOut, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, messageReply, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.MessageInOut), err
}

func (sa *syncAsyncBridge) WaitForMessage(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.Message, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, messageConfirm, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.Message), err
}

func (sa *syncAsyncBridge) WaitForIdentity(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.Identity, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, identityConfirm, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.Identity), err
}

func (sa *syncAsyncBridge) WaitForTokenPool(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenPool, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, tokenPoolConfirm, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.TokenPool), err
}

func (sa *syncAsyncBridge) WaitForTokenTransfer(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenTransfer, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, tokenTransferConfirm, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.TokenTransfer), err
}

func (sa *syncAsyncBridge) WaitForTokenApproval(ctx context.Context, ns string, id *fftypes.UUID, send RequestSender) (*fftypes.TokenApproval, error) {
	reply, err := sa.sendAndWait(ctx, ns, id, tokenApproveConfirm, send)
	if err != nil {
		return nil, err
	}
	return reply.(*fftypes.TokenApproval), err
}
