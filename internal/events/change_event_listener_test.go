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
	"testing"

	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestChangeEventListenerDispatch(t *testing.T) {
	subID := fftypes.NewUUID()
	sub := &subscription{
		dispatcherElection: make(chan bool, 1),
		definition: &fftypes.Subscription{
			SubscriptionRef: fftypes.SubscriptionRef{
				ID: subID, Namespace: "ns1", Name: "sub1",
			},
		},
	}

	ed, cancel := newTestEventDispatcher(sub)
	defer cancel()

	ed.cel.addDispatcher(*subID, ed)

	go ed.cel.changeEventListener()
	ed.cel.changeEvents <- &fftypes.ChangeEvent{
		Collection: "ut",
	}
	ce := <-ed.changeEvents
	assert.Equal(t, "ut", ce.Collection)
}
