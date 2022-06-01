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
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/namespacemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testOrg(name string) *core.Identity {
	i := &core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      core.IdentityTypeOrg,
			Namespace: "ns1",
			Name:      name,
		},
		IdentityProfile: core.IdentityProfile{
			Description: "desc",
			Profile: fftypes.JSONObject{
				"some": "profiledata",
			},
		},
		Messages: core.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	i.DID, _ = i.GenerateDID(context.Background())
	return i
}

func TestRegisterNodeOrgOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(coreconfig.NodeDescription, "Node 1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerBlockchainKey", nm.ctx, "ns1").Return(&core.VerifierRef{
		Value: "0x12345",
	}, nil)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*core.Identity")).Return(nil, false, nil)

	mdm := nm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", nm.ctx, "ns1").Return(nil)

	mockMsg := &core.Message{Header: core.MessageHeader{ID: fftypes.NewUUID()}}
	mbm := nm.broadcast.(*broadcastmocks.Manager)
	mbm.On("BroadcastIdentityClaim", nm.ctx,
		"ns1",
		mock.AnythingOfType("*core.IdentityClaim"),
		mock.MatchedBy(func(sr *core.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		core.SystemTagIdentityClaim, false).Return(mockMsg, nil)

	mns := nm.namespace.(*namespacemocks.Manager)
	mns.On("GetMultipartyConfig", "ns1", coreconfig.OrgName).Return("org1")
	mns.On("GetMultipartyConfig", "ns1", coreconfig.OrgDescription).Return("")

	org, err := nm.RegisterNodeOrganization(nm.ctx, "ns1", false)
	assert.NoError(t, err)
	assert.Equal(t, *mockMsg.Header.ID, *org.Messages.Claim)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mns.AssertExpectations(t)
}

func TestRegisterNodeOrgNoName(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(coreconfig.NodeDescription, "")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerBlockchainKey", nm.ctx, "ns1").Return(&core.VerifierRef{
		Value: "0x12345",
	}, nil)

	mns := nm.namespace.(*namespacemocks.Manager)
	mns.On("GetMultipartyConfig", "ns1", coreconfig.OrgName).Return("")
	mns.On("GetMultipartyConfig", "ns1", coreconfig.OrgDescription).Return("")

	_, err := nm.RegisterNodeOrganization(nm.ctx, "ns1", false)
	assert.Regexp(t, "FF10216", err)

	mim.AssertExpectations(t)
	mns.AssertExpectations(t)
}

func TestRegisterNodeGetOwnerBlockchainKeyFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(coreconfig.OrgName, "")
	config.Set(coreconfig.NodeDescription, "")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerBlockchainKey", nm.ctx, "ns1").Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterNodeOrganization(nm.ctx, "ns1", false)
	assert.Regexp(t, "pop", err)

}
