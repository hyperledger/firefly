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
	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func newTestDefinitionHandler(t *testing.T) (*definitionHandlers, *testDefinitionBatchState) {
	mdi := &databasemocks.Plugin{}
	mbi := &blockchainmocks.Plugin{}
	mdx := &dataexchangemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mam := &assetmocks.Manager{}
	mcm := &contractmocks.Manager{}
	mbi.On("VerifierType").Return(fftypes.VerifierTypeEthAddress).Maybe()
	dh, _ := NewDefinitionHandler(context.Background(), mdi, mbi, mdx, mdm, mim, mam, mcm)
	return dh.(*definitionHandlers), newTestDefinitionBatchState(t)
}

type testDefinitionBatchState struct {
	t                  *testing.T
	preFinalizers      []func(ctx context.Context) error
	finalizers         []func(ctx context.Context) error
	pendingConfirms    map[fftypes.UUID]*fftypes.Message
	confirmedDIDClaims []string
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

func (bs *testDefinitionBatchState) DIDClaimConfirmed(did string) {
	bs.confirmedDIDClaims = append(bs.confirmedDIDClaims, did)
}

func (bs *testDefinitionBatchState) assertNoFinalizers() {
	assert.Empty(bs.t, bs.preFinalizers)
	assert.Empty(bs.t, bs.finalizers)
}

func TestInitFail(t *testing.T) {
	_, err := NewDefinitionHandler(context.Background(), nil, nil, nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestHandleDefinitionBroadcastUnknown(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
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
	dh, _ := newTestDefinitionHandler(t)
	valid := dh.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "unknown",
		},
	}, fftypes.DataArray{}, nil)
	assert.False(t, valid)
}

func TestGetSystemBroadcastPayloadBadJSON(t *testing.T) {
	dh, _ := newTestDefinitionHandler(t)
	valid := dh.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: "unknown",
		},
	}, fftypes.DataArray{}, nil)
	assert.False(t, valid)
}

func TestActionEnum(t *testing.T) {
	assert.Equal(t, "confirm", fmt.Sprintf("%s", ActionConfirm))
	assert.Equal(t, "reject", fmt.Sprintf("%s", ActionReject))
	assert.Equal(t, "retry", fmt.Sprintf("%s", ActionRetry))
	assert.Equal(t, "wait", fmt.Sprintf("%s", ActionWait))
	assert.Equal(t, "unknown", fmt.Sprintf("%s", DefinitionMessageAction(999)))
}
