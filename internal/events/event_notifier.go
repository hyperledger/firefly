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

package events

import (
	"context"
	"sync"

	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
)

type eventNotifier struct {
	ctx            context.Context
	newEvents      chan int64
	latestSequence int64
	cond           *sync.Cond
	closed         bool
}

func newEventNotifier(ctx context.Context) *eventNotifier {
	mux := &sync.Mutex{}
	en := &eventNotifier{
		ctx:            ctx,
		newEvents:      make(chan int64),
		latestSequence: -1,
		cond:           sync.NewCond(mux),
	}
	go en.newEventLoop()
	return en
}

func (en *eventNotifier) waitNext(lastSequence int64) error {
	log.L(en.ctx).Tracef("Next notification %d", lastSequence)
	en.cond.L.Lock()
	closed := en.closed
	for en.latestSequence <= lastSequence && !en.closed {
		en.cond.Wait()
	}
	en.cond.L.Unlock()
	if closed {
		return i18n.NewError(en.ctx, i18n.MsgEventListenerClosing)
	}
	return nil
}

func (en *eventNotifier) close() {
	en.cond.L.Lock()
	en.closed = true
	en.cond.Broadcast()
	en.cond.L.Unlock()
}

func (en *eventNotifier) newEventLoop() {
	l := log.L(en.ctx)
	defer en.close()
	for {
		select {
		case <-en.ctx.Done():
			l.Debugf("New event notifier loop ending (context cancelled)")
			return
		case seq, ok := <-en.newEvents:
			if !ok {
				l.Debugf("New event notifier loop ending (closed channel)")
				return
			}
			log.L(en.ctx).Tracef("Notifying %d", seq)
			en.cond.L.Lock()
			en.latestSequence = seq
			en.cond.Broadcast()
			en.cond.L.Unlock()
		}
	}
}
