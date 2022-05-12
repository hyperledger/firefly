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
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDefinitionHandler struct {
	err error
}

func (dh *mockDefinitionHandler) HandleDefinition(ctx context.Context, state *core.BatchState, msg *core.Message, data *core.Data) error {
	return dh.err
}

func newTestDefinitionSender(t *testing.T) (*definitionSender, func()) {
	mbm := &broadcastmocks.Manager{}
	mim := &identitymanagermocks.Manager{}
	mdm := &datamocks.Manager{}

	ctx, cancel := context.WithCancel(context.Background())
	b, err := NewDefinitionSender(ctx, "ns1", false, mbm, mim, mdm)
	assert.NoError(t, err)
	return b.(*definitionSender), cancel
}

func TestInitFail(t *testing.T) {
	_, err := NewDefinitionSender(context.Background(), "", false, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestName(t *testing.T) {
	bm, cancel := newTestDefinitionSender(t)
	defer cancel()
	assert.Equal(t, "DefinitionSender", bm.Name())
}

func TestCreateDefinitionConfirm(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}

	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("SendAndWait", mock.Anything).Return(nil)

	ds.multiparty = true
	_, err := ds.CreateDefinition(ds.ctx, &core.Namespace{}, core.SystemTagDefineNamespace, true)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestCreateIdentityClaim(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}

	mim.On("NormalizeSigningKey", mock.Anything, "0x1234", identity.KeyNormalizationBlockchainPlugin).Return("", nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("SendAndWait", mock.Anything).Return(nil)

	ds.multiparty = true

	_, err := ds.CreateIdentityClaim(ds.ctx, &core.IdentityClaim{
		Identity: &core.Identity{},
	}, &core.SignerRef{
		Key: "0x1234",
	}, core.SystemTagDefineNamespace, true)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestCreateIdentityClaimFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)

	mim.On("NormalizeSigningKey", mock.Anything, "0x1234", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))

	_, err := ds.CreateIdentityClaim(ds.ctx, &core.IdentityClaim{
		Identity: &core.Identity{},
	}, &core.SignerRef{
		Key: "0x1234",
	}, core.SystemTagDefineNamespace, true)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
}

func TestCreateDatatypeDefinitionAsNodeConfirm(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}

	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", mock.Anything).Return(mms)
	mms.On("SendAndWait", mock.Anything).Return(nil)

	ds.multiparty = true

	_, err := ds.CreateDefinition(ds.ctx, &core.Datatype{}, core.SystemTagDefineNamespace, true)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestCreateDefinitionBadIdentity(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := ds.CreateDefinitionWithIdentity(ds.ctx, &core.Namespace{}, &core.SignerRef{
		Author: "wrong",
		Key:    "wrong",
	}, core.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)
}

func TestCreateDefinitionLocal(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mdh := &mockDefinitionHandler{}
	ds.Init(mdh)

	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)

	_, err := ds.CreateDefinition(ds.ctx, &core.Datatype{}, core.SystemTagDefineDatatype, true)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestCreateDefinitionLocalError(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}
	mdh := &mockDefinitionHandler{err: fmt.Errorf("pop")}
	ds.Init(mdh)

	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)

	_, err := ds.CreateDefinition(ds.ctx, &core.Datatype{}, core.SystemTagDefineDatatype, true)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}