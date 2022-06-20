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

package broadcast

import (
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastDefinitionAsNodeConfirm(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	msa := bm.syncasync.(*syncasyncmocks.Bridge)
	mim := bm.identity.(*identitymanagermocks.Manager)

	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	msa.On("WaitForMessage", bm.ctx, "ns1", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := bm.BroadcastDefinitionAsNode(bm.ctx, "ns1", &core.Namespace{}, core.SystemTagDefineNamespace, true)
	assert.EqualError(t, err, "pop")

	msa.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBroadcastIdentityClaim(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	msa := bm.syncasync.(*syncasyncmocks.Bridge)
	mim := bm.identity.(*identitymanagermocks.Manager)

	mim.On("NormalizeSigningKey", mock.Anything, "0x1234", identity.KeyNormalizationBlockchainPlugin).Return("", nil)
	msa.On("WaitForMessage", bm.ctx, "ns1", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := bm.BroadcastIdentityClaim(bm.ctx, "ns1", &core.IdentityClaim{
		Identity: &core.Identity{},
	}, &core.SignerRef{
		Key: "0x1234",
	}, core.SystemTagDefineNamespace, true)
	assert.EqualError(t, err, "pop")

	msa.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBroadcastIdentityClaimFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mim := bm.identity.(*identitymanagermocks.Manager)

	mim.On("NormalizeSigningKey", mock.Anything, "0x1234", identity.KeyNormalizationBlockchainPlugin).Return("", fmt.Errorf("pop"))

	_, err := bm.BroadcastIdentityClaim(bm.ctx, "ns1", &core.IdentityClaim{
		Identity: &core.Identity{},
	}, &core.SignerRef{
		Key: "0x1234",
	}, core.SystemTagDefineNamespace, true)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
}

func TestBroadcastDatatypeDefinitionAsNodeConfirm(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	msa := bm.syncasync.(*syncasyncmocks.Bridge)
	mim := bm.identity.(*identitymanagermocks.Manager)
	ns := "customNamespace"

	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(nil)
	msa.On("WaitForMessage", bm.ctx, ns, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := bm.BroadcastDefinitionAsNode(bm.ctx, ns, &core.Datatype{}, core.SystemTagDefineNamespace, true)
	assert.EqualError(t, err, "pop")

	msa.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBroadcastDefinitionBadIdentity(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mim := bm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := bm.BroadcastDefinition(bm.ctx, "ns1", &core.Namespace{}, &core.SignerRef{
		Author: "wrong",
		Key:    "wrong",
	}, core.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)
}
