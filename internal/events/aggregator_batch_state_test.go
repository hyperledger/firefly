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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFlushPinsFail(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("UpdatePins", ag.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	bs.MarkMessageDispatched(ag.ctx, fftypes.NewUUID(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}, 0)

	err := bs.flushPins(ag.ctx)
	assert.Regexp(t, "pop", err)
}

func TestSetContextBlockedByNoState(t *testing.T) {
	ag, cancel := newTestAggregator()
	defer cancel()
	bs := newBatchState(ag)

	unmaskedContext := fftypes.NewRandB32()
	bs.SetContextBlockedBy(ag.ctx, *unmaskedContext, 10)

	ready, err := bs.CheckUnmaskedContextReady(ag.ctx, unmaskedContext, &fftypes.Message{}, "topic1", 1)
	assert.NoError(t, err)
	assert.False(t, ready)
}
