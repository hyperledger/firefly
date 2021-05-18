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
	"testing"

	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/publicstoragemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEventManager() (em *eventManager, cancel func()) {
	ctx, cancel := context.WithCancel(context.Background())
	mpi := &publicstoragemocks.Plugin{}
	mdi := &databasemocks.Plugin{}
	em = NewEventManager(ctx, mpi, mdi).(*eventManager)
	return
}

func TestStartStop(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName).Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeAggregator,
		Namespace: fftypes.SystemNamespace,
		Name:      aggregatorOffsetName,
		Current:   12345,
	}, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	mpi := &publicstoragemocks.Plugin{}
	em := NewEventManager(ctx, mpi, mdi)
	assert.NoError(t, em.Start())
	em.NewEvents() <- fftypes.NewUUID()
	cancel()
	em.WaitStop()
}
