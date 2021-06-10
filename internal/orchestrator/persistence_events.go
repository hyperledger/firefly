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

package orchestrator

import (
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (or *orchestrator) MessageCreated(sequence int64) {
	or.batch.NewMessages() <- sequence
}

func (or *orchestrator) PinCreated(sequence int64) {
	or.events.NewPins() <- sequence
}

func (or *orchestrator) EventCreated(sequence int64) {
	or.events.NewEvents() <- sequence
}

func (or *orchestrator) SubscriptionCreated(id *fftypes.UUID) {
	or.events.NewSubscriptions() <- id
}

func (or *orchestrator) SubscriptionDeleted(id *fftypes.UUID) {
	or.events.DeletedSubscriptions() <- id
}
