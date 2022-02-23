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

package networkmap

import (
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterNodeOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.OrgKey, "0x23456")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeDescription, "Node 1")

	parentOrg := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", nm.ctx).Return(parentOrg, nil)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentOrg, nil)
	signerRef := &fftypes.SignerRef{Key: "0x23456"}
	mim.On("ResolveIdentitySigner", nm.ctx, parentOrg).Return(signerRef, nil)

	mdx := nm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("GetEndpointInfo", nm.ctx).Return(fftypes.JSONObject{
		"id":       "peer1",
		"endpoint": "details",
	}, nil)

	mockMsg := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastDefinition", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityClaim"),
		signerRef,
		fftypes.SystemTagIdentityClaim, true).Return(mockMsg, nil)

	node, err := nm.RegisterNode(nm.ctx, true)
	assert.NoError(t, err)
	assert.Equal(t, *mockMsg.Header.ID, *node.Messages.Claim)

}

func TestRegisterNodePeerInfoFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(config.OrgKey, "0x23456")
	config.Set(config.OrgName, "org1")
	config.Set(config.NodeDescription, "Node 1")

	parentOrg := testOrg("org1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", nm.ctx).Return(parentOrg, nil)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(parentOrg, nil)
	signerRef := &fftypes.SignerRef{Key: "0x23456"}
	mim.On("ResolveIdentitySigner", nm.ctx, parentOrg).Return(signerRef, nil)

	mdx := nm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("GetEndpointInfo", nm.ctx).Return(fftypes.JSONObject{}, fmt.Errorf("pop"))

	_, err := nm.RegisterNode(nm.ctx, true)
	assert.Regexp(t, "pop", err)

}

func TestRegisterNodeGetOwnerFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", nm.ctx).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterNode(nm.ctx, true)
	assert.Regexp(t, "pop", err)

}
