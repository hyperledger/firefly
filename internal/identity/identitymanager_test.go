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

package identity

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func newTestIdentityManager(t *testing.T) (context.Context, *identityManager) {

	mdi := &databasemocks.Plugin{}
	mii := &identitymocks.Plugin{}
	mbi := &blockchainmocks.Plugin{}

	config.Reset()

	ctx := context.Background()
	im, err := NewIdentityManager(ctx, mdi, mii, mbi)
	assert.NoError(t, err)
	return ctx, im.(*identityManager)
}

func TestNewIdentityManagerMissingDeps(t *testing.T) {
	_, err := NewIdentityManager(context.Background(), nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestResolveInputIdentityBlankBlank(t *testing.T) {

	identity := &fftypes.Identity{}
	org := &fftypes.Organization{
		ID:       fftypes.NewUUID(),
		Name:     "org1",
		Identity: "0x12345",
	}

	ctx, im := newTestIdentityManager(t)
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", ctx, "org1").Return(org, nil).Once()

	config.Set(config.OrgName, "org1")

	err := im.ResolveInputIdentity(ctx, identity)
	assert.NoError(t, err)
	assert.Equal(t, "0x12345", identity.Key)
	assert.Equal(t, fmt.Sprintf("did:firefly:org/%s", org.ID), identity.Author)

	// Cached result (note once above)
	err = im.ResolveInputIdentity(ctx, &fftypes.Identity{})
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestResolveInputIdentityBlankShortKeyNameResolved(t *testing.T) {

	identity := &fftypes.Identity{
		Key: "org1key",
	}
	org := &fftypes.Organization{
		ID:       fftypes.NewUUID(),
		Name:     "org1",
		Identity: "0x12345",
	}

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "org1key").Return("0x12345", nil).Once()
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", ctx, "0x12345").Return(org, nil).Once()

	config.Set(config.OrgName, "org1")

	err := im.ResolveInputIdentity(ctx, identity)
	assert.NoError(t, err)
	assert.Equal(t, "0x12345", identity.Key)
	assert.Equal(t, fmt.Sprintf("did:firefly:org/%s", org.ID), identity.Author)
	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputIdentityBlankShortKeyNameUnresolved(t *testing.T) {

	identity := &fftypes.Identity{
		Key: "org1key",
	}
	org := &fftypes.Organization{
		ID:       fftypes.NewUUID(),
		Name:     "org1",
		Identity: "0x12345",
	}

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "org1key").Return("0x12345", nil).Once()
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", ctx, "0x12345").Return(nil, nil).Once()
	mdi.On("GetOrganizationByName", ctx, "org1").Return(org, nil).Once()

	config.Set(config.OrgName, "org1")

	err := im.ResolveInputIdentity(ctx, identity)
	assert.NoError(t, err)
	assert.Equal(t, "0x12345", identity.Key)
	assert.Equal(t, fmt.Sprintf("did:firefly:org/%s", org.ID), identity.Author)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputIdentityBlankShortKeyNameFail(t *testing.T) {

	identity := &fftypes.Identity{
		Key: "org1key",
	}

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "org1key").Return("0x12345", nil).Once()
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", ctx, "0x12345").Return(nil, fmt.Errorf("pop")).Once()

	config.Set(config.OrgName, "org1")

	err := im.ResolveInputIdentity(ctx, identity)
	assert.Regexp(t, "pop", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputIdentityOrgIdShortKeyName(t *testing.T) {

	identity := &fftypes.Identity{
		Key:    "org1key",
		Author: "org1",
	}
	org := &fftypes.Organization{
		ID:       fftypes.NewUUID(),
		Name:     "org1",
		Identity: "0x12345",
	}

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "org1key").Return("0x12345", nil).Once()
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", ctx, "org1").Return(org, nil).Once()

	err := im.ResolveInputIdentity(ctx, identity)
	assert.NoError(t, err)
	assert.Equal(t, "0x12345", identity.Key)
	assert.Equal(t, fmt.Sprintf("did:firefly:org/%s", org.ID), identity.Author)

	// Cached result (note once on mocks above)
	err = im.ResolveInputIdentity(ctx, &fftypes.Identity{
		Key:    "org1key",
		Author: "org1",
	})
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestResolveInputIdentityOrgKeyMismatch(t *testing.T) {

	identity := &fftypes.Identity{
		Key:    "org1key",
		Author: "org1",
	}
	org := &fftypes.Organization{
		ID:       fftypes.NewUUID(),
		Name:     "org1",
		Identity: "0x222222",
	}

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "org1key").Return("0x111111", nil).Once()
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", ctx, "org1").Return(org, nil).Once()

	err := im.ResolveInputIdentity(ctx, identity)
	assert.Regexp(t, "FF10279", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestResolveInputIdentityResolveKeyFail(t *testing.T) {

	identity := &fftypes.Identity{
		Key: "org1key",
	}

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "org1key").Return("", fmt.Errorf("pop"))

	err := im.ResolveInputIdentity(ctx, identity)
	assert.Regexp(t, err, "pop")
	mbi.AssertExpectations(t)
}

func TestResolveInputIdentityBadOrgDID(t *testing.T) {

	identity := &fftypes.Identity{
		Author: "did:firefly:org/!NoUUIDHere!",
	}

	ctx, im := newTestIdentityManager(t)

	err := im.ResolveInputIdentity(ctx, identity)
	assert.Regexp(t, "FF10142", err)
}

func TestResolveInputIdentityOrgLookupByDIDFail(t *testing.T) {

	orgId := fftypes.NewUUID()
	identity := &fftypes.Identity{
		Author: fmt.Sprintf("did:firefly:org/%s", orgId),
	}

	ctx, im := newTestIdentityManager(t)
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByID", ctx, orgId).Return(nil, fmt.Errorf("pop"))

	err := im.ResolveInputIdentity(ctx, identity)
	assert.Regexp(t, "pop", err)
	mdi.AssertExpectations(t)
}

func TestResolveInputIdentityOrgLookupByDIDNotFound(t *testing.T) {

	orgId := fftypes.NewUUID()
	identity := &fftypes.Identity{
		Author: fmt.Sprintf("did:firefly:org/%s", orgId),
	}

	ctx, im := newTestIdentityManager(t)
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByID", ctx, orgId).Return(nil, nil)

	err := im.ResolveInputIdentity(ctx, identity)
	assert.Regexp(t, "FF10277", err)
	mdi.AssertExpectations(t)
}

func TestResolveInputIdentityOrgLookupByNameFail(t *testing.T) {

	identity := &fftypes.Identity{
		Author: "org1",
	}

	ctx, im := newTestIdentityManager(t)
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", ctx, "org1").Return(nil, fmt.Errorf("pop"))

	err := im.ResolveInputIdentity(ctx, identity)
	assert.Regexp(t, "pop", err)
	mdi.AssertExpectations(t)
}

func TestResolveInputIdentityOrgLookupByNameNotFound(t *testing.T) {

	identity := &fftypes.Identity{
		Author: "org1",
	}

	ctx, im := newTestIdentityManager(t)
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", ctx, "org1").Return(nil, nil)

	err := im.ResolveInputIdentity(ctx, identity)
	assert.Regexp(t, "FF10278", err)
	mdi.AssertExpectations(t)
}

func TestResolveSigningKeyIdentityBadSigningKey(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "badness").Return("", fmt.Errorf("pop"))

	_, err := im.ResolveSigningKeyIdentity(ctx, "badness")
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
}

func TestResolveSigningKeyIdentityOrgLookupFail(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "key1").Return("key1resolved", nil)
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", ctx, "key1resolved").Return(nil, fmt.Errorf("pop"))

	_, err := im.ResolveSigningKeyIdentity(ctx, "key1")
	assert.Regexp(t, "pop", err)
	mbi.AssertExpectations(t)
}

func TestResolveSigningKeyIdentityOrgLookupOkCached(t *testing.T) {

	org := &fftypes.Organization{
		ID:       fftypes.NewUUID(),
		Identity: "key1resolved",
	}

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "key1").Return("key1resolved", nil)
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", ctx, "key1resolved").Return(org, nil).Once()

	author, err := im.ResolveSigningKeyIdentity(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, im.OrgDID(org), author)

	// Cached second time, without any DB call (see "Once()" above)
	author, err = im.ResolveSigningKeyIdentity(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, im.OrgDID(org), author)

	mbi.AssertExpectations(t)
}

func TestResolveSigningKeyIdentityOrgLookupUnresolved(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "key1").Return("key1resolved", nil)
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", ctx, "key1resolved").Return(nil, nil)

	author, err := im.ResolveSigningKeyIdentity(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "", author)

	mbi.AssertExpectations(t)
}

func TestResolveLocalOrgDIDSuccess(t *testing.T) {

	org := &fftypes.Organization{
		ID:       fftypes.NewUUID(),
		Name:     "org1",
		Identity: "0x222222",
	}

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "key1").Return("key1resolved", nil).Once()
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", ctx, "key1resolved").Return(org, nil).Once()

	config.Set(config.OrgIdentityDeprecated, "key1")

	localOrgDID, err := im.ResolveLocalOrgDID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, im.OrgDID(org), localOrgDID)

	// Second one cached
	localOrgDID, err = im.ResolveLocalOrgDID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, im.OrgDID(org), localOrgDID)

	mdi.On("GetOrganizationByID", ctx, org.ID).Return(org, nil)
	localOrg, err := im.GetLocalOrganization(ctx)
	assert.NoError(t, err)
	assert.Equal(t, org, localOrg)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestResolveLocalOrgDIDFail(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "key1").Return("key1resolved", nil).Once()
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", ctx, "key1resolved").Return(nil, fmt.Errorf("pop")).Twice()

	config.Set(config.OrgIdentityDeprecated, "key1")

	_, err := im.ResolveLocalOrgDID(ctx)
	assert.Regexp(t, "FF10280", err)

	_, err = im.GetLocalOrganization(ctx)
	assert.Regexp(t, "FF10290", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestResolveLocalOrgDIDNotFound(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "key1").Return("key1resolved", nil).Once()
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", ctx, "key1resolved").Return(nil, nil).Once()

	config.Set(config.OrgIdentityDeprecated, "key1")

	_, err := im.ResolveLocalOrgDID(ctx)
	assert.Regexp(t, "FF10280", err)

	mbi.AssertExpectations(t)

}
