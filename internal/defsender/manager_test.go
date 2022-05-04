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

package defsender

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestDefinitionSender(t *testing.T) (*definitionSender, func()) {
	mbm := &broadcastmocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mdm := &datamocks.Manager{}

	ctx, cancel := context.WithCancel(context.Background())
	b, err := NewDefinitionSender(ctx, mbm, mim, mdm)
	assert.NoError(t, err)
	return b.(*definitionSender), cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewDefinitionSender(context.Background(), nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestName(t *testing.T) {
	bm, cancel := newTestDefinitionSender(t)
	defer cancel()
	assert.Equal(t, "DefinitionSender", bm.Name())
}

func TestBroadcastDefinitionAsNodeConfirm(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}

	mim.On("ResolveInputSigningIdentity", mock.Anything, "ff_system", mock.Anything).Return(nil)
	mbm.On("NewBroadcast", "ff_system", mock.Anything).Return(mms)
	mms.On("SendAndWait", mock.Anything).Return(nil)

	_, err := ds.BroadcastDefinitionAsNode(ds.ctx, fftypes.SystemNamespace, &fftypes.Namespace{}, fftypes.SystemTagDefineNamespace, true)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestBroadcastIdentityClaim(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}

	mim.On("NormalizeSigningKey", mock.Anything, "0x1234", identity.KeyNormalizationBlockchainPlugin).Return("", nil)
	mbm.On("NewBroadcast", "ff_system", mock.Anything).Return(mms)
	mms.On("SendAndWait", mock.Anything).Return(nil)

	_, err := ds.BroadcastIdentityClaim(ds.ctx, fftypes.SystemNamespace, &fftypes.IdentityClaim{
		Identity: &fftypes.Identity{},
	}, &fftypes.SignerRef{
		Key: "0x1234",
	}, fftypes.SystemTagDefineNamespace, true)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestBroadcastIdentityClaimFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)

	mim.On("NormalizeSigningKey", mock.Anything, "0x1234", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))

	_, err := ds.BroadcastIdentityClaim(ds.ctx, fftypes.SystemNamespace, &fftypes.IdentityClaim{
		Identity: &fftypes.Identity{},
	}, &fftypes.SignerRef{
		Key: "0x1234",
	}, fftypes.SystemTagDefineNamespace, true)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
}

func TestBroadcastDatatypeDefinitionAsNodeConfirm(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	ns := "customNamespace"

	mim.On("ResolveInputSigningIdentity", mock.Anything, ns, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", ns, mock.Anything).Return(mms)
	mms.On("SendAndWait", mock.Anything).Return(nil)

	_, err := ds.BroadcastDefinitionAsNode(ds.ctx, ns, &fftypes.Datatype{}, fftypes.SystemTagDefineNamespace, true)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestBroadcastDefinitionBadIdentity(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", mock.Anything, fftypes.SystemNamespace, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := ds.BroadcastDefinition(ds.ctx, fftypes.SystemNamespace, &fftypes.Namespace{}, &fftypes.SignerRef{
		Author: "wrong",
		Key:    "wrong",
	}, fftypes.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)
}
