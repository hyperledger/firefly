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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestDefinitionHandlers(t *testing.T) (*definitionHandlers, *testDefinitionBatchState) {
	mdi := &databasemocks.Plugin{}
	mbi := &blockchainmocks.Plugin{}
	mdx := &dataexchangemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mbm := &broadcastmocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	mam := &assetmocks.Manager{}
	mcm := &contractmocks.Manager{}
	mbi.On("VerifierType").Return(fftypes.VerifierTypeEthAddress).Maybe()
	return NewDefinitionHandlers(mdi, mbi, mdx, mdm, mim, mbm, mpm, mam, mcm).(*definitionHandlers), newTestDefinitionBatchState(t)
}

type testDefinitionBatchState struct {
	t               *testing.T
	preFinalizers   []func(ctx context.Context) error
	finalizers      []func(ctx context.Context) error
	pendingConfirms map[fftypes.UUID]*fftypes.Message
}

func newTestDefinitionBatchState(t *testing.T) *testDefinitionBatchState {
	return &testDefinitionBatchState{
		t:               t,
		pendingConfirms: make(map[fftypes.UUID]*fftypes.Message),
	}
}

func (bs *testDefinitionBatchState) AddPreFinalize(pf func(ctx context.Context) error) {
	bs.preFinalizers = append(bs.preFinalizers, pf)
}

func (bs *testDefinitionBatchState) AddFinalize(pf func(ctx context.Context) error) {
	bs.finalizers = append(bs.finalizers, pf)
}

func (bs *testDefinitionBatchState) GetPendingConfirm() map[fftypes.UUID]*fftypes.Message {
	return bs.pendingConfirms
}

func (bs *testDefinitionBatchState) assertNoFinalizers() {
	assert.Empty(bs.t, bs.preFinalizers)
	assert.Empty(bs.t, bs.finalizers)
}

func TestHandleDefinitionBroadcastUnknown(t *testing.T) {
	dh, bs := newTestDefinitionHandlers(t)
	action, err := dh.HandleDefinitionBroadcast(context.Background(), bs, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "unknown",
		},
	}, fftypes.DataArray{}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)
	bs.assertNoFinalizers()
}

func TestGetSystemBroadcastPayloadMissingData(t *testing.T) {
	dh, _ := newTestDefinitionHandlers(t)
	valid := dh.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "unknown",
		},
	}, fftypes.DataArray{}, nil)
	assert.False(t, valid)
}

func TestGetSystemBroadcastPayloadBadJSON(t *testing.T) {
	dh, _ := newTestDefinitionHandlers(t)
	valid := dh.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "unknown",
		},
	}, fftypes.DataArray{}, nil)
	assert.False(t, valid)
}

func TestPrivateMessagingPassthroughs(t *testing.T) {
	ctx := context.Background()

	dh, _ := newTestDefinitionHandlers(t)
	mpm := dh.messaging.(*privatemessagingmocks.Manager)
	mpm.On("GetGroupByID", ctx, mock.Anything).Return(nil, nil)
	mpm.On("GetGroupsNS", ctx, "ns1", mock.Anything).Return(nil, nil, nil)
	mpm.On("ResolveInitGroup", ctx, mock.Anything).Return(nil, nil)
	mpm.On("EnsureLocalGroup", ctx, mock.Anything).Return(false, nil)

	_, _ = dh.GetGroupByID(ctx, fftypes.NewUUID().String())
	_, _, _ = dh.GetGroupsNS(ctx, "ns1", nil)
	_, _ = dh.ResolveInitGroup(ctx, nil)
	_, _ = dh.EnsureLocalGroup(ctx, nil)

	mpm.AssertExpectations(t)

}

func TestActionEnum(t *testing.T) {
	assert.Equal(t, "confirm", fmt.Sprintf("%s", ActionConfirm))
	assert.Equal(t, "reject", fmt.Sprintf("%s", ActionReject))
	assert.Equal(t, "retry", fmt.Sprintf("%s", ActionRetry))
	assert.Equal(t, "wait", fmt.Sprintf("%s", ActionWait))
	assert.Equal(t, "unknown", fmt.Sprintf("%s", DefinitionMessageAction(999)))
}
