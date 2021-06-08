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
	"testing"
)

func TestEventNotifier(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	en := newEventNotifier(ctx, "ut")
	var mySeq int64 = 1000000
	events := make(chan bool)
	go func() {
		defer close(events)
		for {
			err := en.waitNext(mySeq)
			if err != nil {
				return
			}
			events <- true
			mySeq++
		}
	}()

	en.newEvents <- 1000001
	<-events
	en.newEvents <- 1000002
	<-events

	cancel()
	<-events
}

func TestEventNotifierClosedChannel(t *testing.T) {
	en := newEventNotifier(context.Background(), "ut")
	var mySeq int64 = 1000000
	events := make(chan bool)
	go func() {
		defer close(events)
		for {
			err := en.waitNext(mySeq)
			if err != nil {
				return
			}
			events <- true
			mySeq++
		}
	}()

	en.newEvents <- 1000001
	<-events
	en.newEvents <- 1000002
	<-events

	close(en.newEvents)
	<-events
}
