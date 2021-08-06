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

package syshandlers

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/firefly/mocks/broadcastmocks"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger-labs/firefly/mocks/datamocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymocks"
	"github.com/hyperledger-labs/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestSystemHandlers(t *testing.T) *systemHandlers {
	mdi := &databasemocks.Plugin{}
	mii := &identitymocks.Plugin{}
	mdx := &dataexchangemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mbm := &broadcastmocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	return NewSystemHandlers(mdi, mii, mdx, mdm, mbm, mpm).(*systemHandlers)
}

func TestHandleSystemBroadcastUnknown(t *testing.T) {
	sh := newTestSystemHandlers(t)
	valid, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "uknown",
		},
	}, []*fftypes.Data{})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestGetSystemBroadcastPayloadMissingData(t *testing.T) {
	sh := newTestSystemHandlers(t)
	valid := sh.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "uknown",
		},
	}, []*fftypes.Data{}, nil)
	assert.False(t, valid)
}

func TestGetSystemBroadcastPayloadBadJSON(t *testing.T) {
	sh := newTestSystemHandlers(t)
	valid := sh.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "uknown",
		},
	}, []*fftypes.Data{}, nil)
	assert.False(t, valid)
}

func TestPrivateMessagingPassthroughs(t *testing.T) {
	ctx := context.Background()

	sh := newTestSystemHandlers(t)
	mpm := sh.messaging.(*privatemessagingmocks.Manager)
	mpm.On("GetGroupByID", ctx, mock.Anything).Return(nil, nil)
	mpm.On("GetGroups", ctx, mock.Anything).Return(nil, nil)
	mpm.On("ResolveInitGroup", ctx, mock.Anything).Return(nil, nil)
	mpm.On("EnsureLocalGroup", ctx, mock.Anything).Return(false, nil)
	mpm.On("SendMessageWithID", ctx, "ns1", mock.Anything, mock.Anything).Return(nil, nil)

	_, _ = sh.GetGroupByID(ctx, fftypes.NewUUID().String())
	_, _ = sh.GetGroups(ctx, nil)
	_, _ = sh.ResolveInitGroup(ctx, nil)
	_, _ = sh.EnsureLocalGroup(ctx, nil)
	_, _ = sh.SendMessageWithID(ctx, "ns1", nil, false)

	mpm.AssertExpectations(t)

}
