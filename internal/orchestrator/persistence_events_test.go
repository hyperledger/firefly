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

package orchestrator

import (
	"testing"

	"github.com/kaleido-io/firefly/mocks/batchmocks"
	"github.com/kaleido-io/firefly/mocks/eventmocks"
)

func TestMessageCreated(t *testing.T) {
	mb := &batchmocks.BatchManager{}
	o := &orchestrator{
		batch: mb,
	}
	c := make(chan int64, 1)
	mb.On("NewMessages").Return((chan<- int64)(c))
	o.MessageCreated(12345)
}

func TestEventCreated(t *testing.T) {
	mem := &eventmocks.EventManager{}
	o := &orchestrator{
		events: mem,
	}
	c := make(chan int64, 1)
	mem.On("NewEvents").Return((chan<- int64)(c))
	o.EventCreated(12345)
}
