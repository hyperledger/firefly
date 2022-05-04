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

	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/defsendermocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/config"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testOrg(name string) *fftypes.Identity {
	i := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.IdentityTypeOrg,
			Namespace: fftypes.SystemNamespace,
			Name:      name,
		},
		IdentityProfile: fftypes.IdentityProfile{
			Description: "desc",
			Profile: fftypes.JSONObject{
				"some": "profiledata",
			},
		},
		Messages: fftypes.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	i.DID, _ = i.GenerateDID(context.Background())
	return i
}

func TestRegisterNodeOrgOk(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(coreconfig.OrgName, "org1")
	config.Set(coreconfig.NodeDescription, "Node 1")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerBlockchainKey", nm.ctx).Return(&fftypes.VerifierRef{
		Value: "0x12345",
	}, nil)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(nil, false, nil)

	mockMsg := &fftypes.Message{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}}
	mds := nm.defsender.(*defsendermocks.Sender)
	mds.On("BroadcastIdentityClaim", nm.ctx,
		fftypes.SystemNamespace,
		mock.AnythingOfType("*fftypes.IdentityClaim"),
		mock.MatchedBy(func(sr *fftypes.SignerRef) bool {
			return sr.Key == "0x12345"
		}),
		fftypes.SystemTagIdentityClaim, false).Return(mockMsg, nil)

	org, err := nm.RegisterNodeOrganization(nm.ctx, false)
	assert.NoError(t, err)
	assert.Equal(t, *mockMsg.Header.ID, *org.Messages.Claim)

}

func TestRegisterNodeOrgNoName(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(coreconfig.OrgName, "")
	config.Set(coreconfig.NodeDescription, "")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerBlockchainKey", nm.ctx).Return(&fftypes.VerifierRef{
		Value: "0x12345",
	}, nil)
	mim.On("VerifyIdentityChain", nm.ctx, mock.AnythingOfType("*fftypes.Identity")).Return(nil, false, nil)

	_, err := nm.RegisterNodeOrganization(nm.ctx, false)
	assert.Regexp(t, "FF10216", err)

}

func TestRegisterNodeGetOwnerBlockchainKeyFail(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	config.Set(coreconfig.OrgName, "")
	config.Set(coreconfig.NodeDescription, "")

	mim := nm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerBlockchainKey", nm.ctx).Return(nil, fmt.Errorf("pop"))

	_, err := nm.RegisterNodeOrganization(nm.ctx, false)
	assert.Regexp(t, "pop", err)

}
