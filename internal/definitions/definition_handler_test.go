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

package definitions

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestDefinitionHandlers(t *testing.T) *definitionHandlers {
	mdi := &databasemocks.Plugin{}
	mdx := &dataexchangemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mbm := &broadcastmocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	mam := &assetmocks.Manager{}
	return NewDefinitionHandlers(mdi, mdx, mdm, mbm, mpm, mam).(*definitionHandlers)
}

func TestHandleSystemBroadcastUnknown(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "unknown",
		},
	}, []*fftypes.Data{})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestGetSystemBroadcastPayloadMissingData(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	valid := dh.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "unknown",
		},
	}, []*fftypes.Data{}, nil)
	assert.False(t, valid)
}

func TestGetSystemBroadcastPayloadBadJSON(t *testing.T) {
	dh := newTestDefinitionHandlers(t)
	valid := dh.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "unknown",
		},
	}, []*fftypes.Data{}, nil)
	assert.False(t, valid)
}

func TestPrivateMessagingPassthroughs(t *testing.T) {
	ctx := context.Background()

	dh := newTestDefinitionHandlers(t)
	mpm := dh.messaging.(*privatemessagingmocks.Manager)
	mpm.On("GetGroupByID", ctx, mock.Anything).Return(nil, nil)
	mpm.On("GetGroups", ctx, mock.Anything).Return(nil, nil, nil)
	mpm.On("ResolveInitGroup", ctx, mock.Anything).Return(nil, nil)
	mpm.On("EnsureLocalGroup", ctx, mock.Anything).Return(false, nil)

	_, _ = dh.GetGroupByID(ctx, fftypes.NewUUID().String())
	_, _, _ = dh.GetGroups(ctx, nil)
	_, _ = dh.ResolveInitGroup(ctx, nil)
	_, _ = dh.EnsureLocalGroup(ctx, nil)

	mpm.AssertExpectations(t)

}
