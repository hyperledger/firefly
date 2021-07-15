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

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

type changeEventListener struct {
	ctx          context.Context
	changeEvents chan *fftypes.ChangeEvent
	dispatchers  map[fftypes.UUID]*eventDispatcher
	mux          sync.Mutex
}

func newChangeEventListener(ctx context.Context) *changeEventListener {
	return &changeEventListener{
		ctx:          ctx,
		changeEvents: make(chan *fftypes.ChangeEvent, config.GetInt(config.EventDBEventsBufferSize)),
		dispatchers:  make(map[fftypes.UUID]*eventDispatcher),
	}
}

func (cel *changeEventListener) dispatch(ce *fftypes.ChangeEvent) {
	// Take a copy of the dispatcher list, so we don't hold a lock while calling a dispatcher
	cel.mux.Lock()
	dispatchers := make([]*eventDispatcher, 0, len(cel.dispatchers))
	for _, d := range cel.dispatchers {
		dispatchers = append(dispatchers, d)
	}
	cel.mux.Unlock()

	for _, d := range dispatchers {
		d.dispatchChangeEvent(ce)
	}
}

func (cel *changeEventListener) addDispatcher(uuid fftypes.UUID, dispatcher *eventDispatcher) {
	cel.mux.Lock()
	cel.dispatchers[uuid] = dispatcher
	cel.mux.Unlock()
}

func (cel *changeEventListener) removeDispatcher(uuid fftypes.UUID) {
	cel.mux.Lock()
	delete(cel.dispatchers, uuid)
	cel.mux.Unlock()
}

func (cel *changeEventListener) changeEventListener() {
	for {
		select {
		case ce := <-cel.changeEvents:
			cel.dispatch(ce)
		case <-cel.ctx.Done():
			return
		}
	}
}
