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

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/eventsmocks"
	"github.com/kaleido-io/firefly/mocks/publicstoragemocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEventManager(t *testing.T) (*eventManager, func()) {
	config.Reset()
	ctx, cancel := context.WithCancel(context.Background())
	mdi := &databasemocks.Plugin{}
	mpi := &publicstoragemocks.Plugin{}
	met := &eventsmocks.Plugin{}
	met.On("Name").Return("ut").Maybe()
	em, err := NewEventManager(ctx, mpi, mdi)
	assert.NoError(t, err)
	return em.(*eventManager), cancel
}

func TestStartStop(t *testing.T) {
	em, cancel := newTestEventManager(t)
	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetOffset", mock.Anything, fftypes.OffsetTypeAggregator, fftypes.SystemNamespace, aggregatorOffsetName).Return(&fftypes.Offset{
		Type:      fftypes.OffsetTypeAggregator,
		Namespace: fftypes.SystemNamespace,
		Name:      aggregatorOffsetName,
		Current:   12345,
	}, nil)
	mdi.On("GetEvents", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Event{}, nil)
	mdi.On("GetSubscriptions", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Subscription{}, nil)
	assert.NoError(t, em.Start())
	em.NewEvents() <- 12345
	cancel()
	em.WaitStop()
}

func TestStartStopBadDependencies(t *testing.T) {
	_, err := NewEventManager(context.Background(), nil, nil)
	assert.Regexp(t, "FF10128", err)

}

func TestStartStopBadTransports(t *testing.T) {
	config.Set(config.EventTransportsEnabled, []string{"wrongun"})
	defer config.Reset()
	mdi := &databasemocks.Plugin{}
	mpi := &publicstoragemocks.Plugin{}
	_, err := NewEventManager(context.Background(), mpi, mdi)
	assert.Regexp(t, "FF10172", err)

}
