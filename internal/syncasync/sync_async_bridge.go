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
	"github.com/hyperledger-labs/firefly/internal/events"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/privatemessaging"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

// Bridge translates synchronous (HTTP API) calls, into asynchronously sending a
// message and blocking until a correlating response is received, or we hit a timeout.
type Bridge interface {
	// Request performs a request/reply exchange taking a message as input, and returning a message as a response
	// The input message must have a tag, and a group, to be routed appropriately.
	RequestReply(ctx context.Context, ns string, request *fftypes.MessageInput) (reply *fftypes.MessageInput, err error)
}

type inflightRequest struct {
	id        *fftypes.UUID
	startTime time.Time
	namespace string
	response  chan *fftypes.MessageInput
}

type syncAsyncBridge struct {
	ctx         context.Context
	database    database.Plugin
	data        data.Manager
	events      events.EventManager
	messaging   privatemessaging.Manager
	inflightMux sync.Mutex
	inflight    map[string]map[fftypes.UUID]*inflightRequest
}

func NewSyncAsyncBridge(ctx context.Context, di database.Plugin, dm data.Manager, ei events.EventManager, pm privatemessaging.Manager) Bridge {
	sa := &syncAsyncBridge{
		ctx:       log.WithLogField(ctx, "role", "sync-async-bridge"),
		database:  di,
		data:      dm,
		events:    ei,
		messaging: pm,
		inflight:  make(map[string]map[fftypes.UUID]*inflightRequest),
	}
	return sa
}

func (sa *syncAsyncBridge) addInFlight(ns string) (*inflightRequest, error) {
	inflight := &inflightRequest{
		id:        fftypes.NewUUID(),
		namespace: ns,
		startTime: time.Now(),
		response:  make(chan *fftypes.MessageInput),
	}
	sa.inflightMux.Lock()
	defer func() {
		sa.inflightMux.Unlock()
		log.L(sa.ctx).Infof("RequestReply '%s' added", inflight.id)
	}()

	inflightNS := sa.inflight[ns]
	if inflightNS == nil {
		err := sa.events.AddSystemEventListener(ns, sa.eventCallback)
		if err != nil {
			return nil, err
		}
		inflightNS = make(map[fftypes.UUID]*inflightRequest)
		sa.inflight[ns] = inflightNS
	}
	inflightNS[*inflight.id] = inflight
	return inflight, nil
}

func (sa *syncAsyncBridge) removeInFlight(inflight *inflightRequest, reply *fftypes.MessageInput) {
	sa.inflightMux.Lock()
	defer func() {
		sa.inflightMux.Unlock()
		if reply != nil {
			log.L(sa.ctx).Infof("RequestReply '%s' resolved with message '%s' after %.2fms", inflight.id, reply.Header.ID, inflight.msInflight())
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
	if len(inflightNS) == 0 || event.Type != fftypes.EventTypeMessageConfirmed {
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
	if msg.Header.CID != nil {
		inflight := inflightNS[*msg.Header.CID]
		if inflight != nil {
			// No need to block while locked any further here - kick off the data lookup and resolve
			go sa.resolveInflight(inflight, msg)
		}
	}
	return nil
}

func (sa *syncAsyncBridge) resolveInflight(inflight *inflightRequest, msg *fftypes.Message) {

	log.L(sa.ctx).Debugf("Resolving RequestReply '%s' with message '%s'", inflight.id, msg.Header.ID)

	data, _, err := sa.data.GetMessageData(sa.ctx, msg, true)
	if err != nil {
		log.L(sa.ctx).Errorf("Failed to read response data for message '%s' on request '%s': %s", msg.Header.ID, inflight.id, err)
		return
	}
	response := &fftypes.MessageInput{Message: *msg}
	response.SetInlineData(data)

	// We deliver the full data in the response
	inflight.response <- response
}

func (sa *syncAsyncBridge) RequestReply(ctx context.Context, ns string, inRequest *fftypes.MessageInput) (reply *fftypes.MessageInput, err error) {

	if inRequest.Header.Tag == "" {
		return nil, i18n.NewError(ctx, i18n.MsgRequestReplyTagRequired)
	}
	if inRequest.Header.CID != nil {
		return nil, i18n.NewError(ctx, i18n.MsgRequestCannotHaveCID)
	}

	inflight, err := sa.addInFlight(ns)
	if err != nil {
		return nil, err
	}
	defer func() {
		sa.removeInFlight(inflight, reply)
	}()

	inRequest.Header.ID = inflight.id
	_, err = sa.messaging.SendMessageWithID(ctx, ns, inRequest)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, i18n.NewError(ctx, i18n.MsgRequestTimeout, inflight.id, inflight.msInflight())
	case reply = <-inflight.response:
		return reply, nil
	}
}
