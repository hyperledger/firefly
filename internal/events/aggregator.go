// Copyright © 2021 Kaleido, Inc.
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

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/log"
)

type aggregator struct {
	ctx       context.Context
	newEvents chan *uuid.UUID
}

func newAggregator(ctx context.Context) *aggregator {
	ag := &aggregator{
		ctx:       log.WithLogField(ctx, "role", "aggregator"),
		newEvents: make(chan *uuid.UUID),
	}
	go ag.eventLoop()
	return ag
}

func (ag *aggregator) eventLoop() {
	l := log.L(ag.ctx)
	for {
		select {
		case ev := <-ag.newEvents:
			l.Infof("New event: %s", ev)
		case <-ag.ctx.Done():
			l.Debugf("Aggregator existing (context canceled)")
			return
		}
	}
}
