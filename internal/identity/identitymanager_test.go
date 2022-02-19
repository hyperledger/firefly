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
	"github.com/stretchr/testify/mock"
)

func newTestIdentityManager(t *testing.T) (context.Context, *identityManager) {

	mdi := &databasemocks.Plugin{}
	mii := &identitymocks.Plugin{}
	mbi := &blockchainmocks.Plugin{}

	config.Reset()

	mbi.On("VerifierType").Return(fftypes.VerifierTypeEthAddress).Maybe()

	ctx := context.Background()
	im, err := NewIdentityManager(ctx, mdi, mii, mbi)
	assert.NoError(t, err)
	return ctx, im.(*identityManager)
}

func TestNewIdentityManagerMissingDeps(t *testing.T) {
	_, err := NewIdentityManager(context.Background(), nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestResolveInputSigningIdentityNoOrgKey(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	msgIdentity := &fftypes.SignerRef{}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "FF10351", err)

}

func TestResolveInputSigningIdentityOrgFallbackOk(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	config.Set(config.OrgKey, "key123")
	config.Set(config.OrgName, "org1")

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "key123").Return("fullkey123", nil)

	orgID := fftypes.NewUUID()

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").
		Return(&fftypes.Verifier{
			ID:        fftypes.NewUUID(),
			Identity:  orgID,
			Namespace: fftypes.SystemNamespace,
			VerifierRef: fftypes.VerifierRef{
				Type:  fftypes.VerifierTypeEthAddress,
				Value: "fullkey123",
			},
		}, nil)
	mdi.On("GetIdentityByID", ctx, orgID).
		Return(&fftypes.Identity{
			IdentityBase: fftypes.IdentityBase{
				ID:        orgID,
				DID:       "did:firefly:org/org1",
				Namespace: fftypes.SystemNamespace,
				Name:      "org1",
				Type:      fftypes.IdentityTypeOrg,
			},
		}, nil)

	msgIdentity := &fftypes.SignerRef{}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.NoError(t, err)
	assert.Equal(t, "did:firefly:org/org1", msgIdentity.Author)
	assert.Equal(t, "fullkey123", msgIdentity.Key)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityByKeyOk(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("fullkey123", nil)

	idID := fftypes.NewUUID()

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").
		Return(&fftypes.Verifier{
			ID:        fftypes.NewUUID(),
			Identity:  idID,
			Namespace: "ns1",
			VerifierRef: fftypes.VerifierRef{
				Type:  fftypes.VerifierTypeEthAddress,
				Value: "fullkey123",
			},
		}, nil)
	mdi.On("GetIdentityByID", ctx, idID).
		Return(&fftypes.Identity{
			IdentityBase: fftypes.IdentityBase{
				ID:        idID,
				DID:       "did:firefly:ns/ns1/myid",
				Namespace: fftypes.SystemNamespace,
				Name:      "myid",
				Type:      fftypes.IdentityTypeCustom,
			},
		}, nil)

	msgIdentity := &fftypes.SignerRef{
		Key: "mykey123",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.NoError(t, err)
	assert.Equal(t, "did:firefly:ns/ns1/myid", msgIdentity.Author)
	assert.Equal(t, "fullkey123", msgIdentity.Key)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityAnonymousKeyWithAuthorOk(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("fullkey123", nil)

	idID := fftypes.NewUUID()

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "fullkey123").Return(nil, nil)
	mdi.On("GetIdentityByDID", ctx, "did:firefly:ns/ns1/myid").
		Return(&fftypes.Identity{
			IdentityBase: fftypes.IdentityBase{
				ID:        idID,
				DID:       "did:firefly:ns/ns1/myid",
				Namespace: fftypes.SystemNamespace,
				Name:      "myid",
				Type:      fftypes.IdentityTypeCustom,
			},
		}, nil)

	msgIdentity := &fftypes.SignerRef{
		Key:    "mykey123",
		Author: "did:firefly:ns/ns1/myid",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.NoError(t, err)
	assert.Equal(t, "did:firefly:ns/ns1/myid", msgIdentity.Author)
	assert.Equal(t, "fullkey123", msgIdentity.Key)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityKeyWithNoAuthorFail(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("fullkey123", nil)

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "fullkey123").Return(nil, nil)

	msgIdentity := &fftypes.SignerRef{
		Key: "mykey123",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "FF10353", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityByKeyDIDMismatch(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("fullkey123", nil)

	idID := fftypes.NewUUID()

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").
		Return(&fftypes.Verifier{
			ID:        fftypes.NewUUID(),
			Identity:  idID,
			Namespace: "ns1",
			VerifierRef: fftypes.VerifierRef{
				Type:  fftypes.VerifierTypeEthAddress,
				Value: "fullkey123",
			},
		}, nil)
	mdi.On("GetIdentityByID", ctx, idID).
		Return(&fftypes.Identity{
			IdentityBase: fftypes.IdentityBase{
				ID:        idID,
				DID:       "did:firefly:ns/ns1/myid",
				Namespace: "ns1",
				Name:      "myid",
				Type:      fftypes.IdentityTypeCustom,
			},
		}, nil)

	msgIdentity := &fftypes.SignerRef{
		Key:    "mykey123",
		Author: "did:firefly:ns/ns1/notmyid",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "FF10352", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityByKeyNotFound(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("fullkey123", nil)

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").
		Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "fullkey123").
		Return(nil, nil)
	mdi.On("GetIdentityByDID", ctx, "did:firefly:ns/ns1/unknown").
		Return(nil, nil)

	msgIdentity := &fftypes.SignerRef{
		Key:    "mykey123",
		Author: "did:firefly:ns/ns1/unknown",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "FF10277", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityForRootOrgRegOk(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("fullkey123", nil)

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").
		Return(nil, nil)

	msgIdentity := &fftypes.SignerRef{
		Key:    "mykey123",
		Author: "did:firefly:org/neworgname",
	}
	err := im.ResolveRootOrgRegistrationSigningKey(ctx, fftypes.SystemNamespace, msgIdentity)
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityForRootOrgRegBadNS(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("fullkey123", nil)

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").
		Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, "ns1", "fullkey123").
		Return(nil, nil)

	msgIdentity := &fftypes.SignerRef{
		Key:    "mykey123",
		Author: "did:firefly:orgs/neworgname",
	}
	err := im.ResolveRootOrgRegistrationSigningKey(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "FF10354", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityForRootOrgRegBadDID(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("fullkey123", nil)

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").
		Return(nil, nil)

	msgIdentity := &fftypes.SignerRef{
		Key:    "mykey123",
		Author: "did:firefly:nodes/node1",
	}
	err := im.ResolveRootOrgRegistrationSigningKey(ctx, fftypes.SystemNamespace, msgIdentity)
	assert.Regexp(t, "FF10354", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityByKeyFail(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("fullkey123", nil)

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").
		Return(nil, fmt.Errorf("pop"))

	msgIdentity := &fftypes.SignerRef{
		Key: "mykey123",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "pop", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityByKeyResolveFail(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "mykey123").Return("", fmt.Errorf("pop"))

	msgIdentity := &fftypes.SignerRef{
		Key: "mykey123",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "pop", err)

	mbi.AssertExpectations(t)
}

func TestResolveInputSigningIdentityByOrgNameOk(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	idID := fftypes.NewUUID()

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, fftypes.IdentityTypeOrg, fftypes.SystemNamespace, "org1").
		Return(&fftypes.Identity{
			IdentityBase: fftypes.IdentityBase{
				ID:        idID,
				DID:       "did:firefly:org/org1",
				Namespace: fftypes.SystemNamespace,
				Name:      "myid",
				Type:      fftypes.IdentityTypeOrg,
			},
		}, nil)
	mdi.On("GetVerifiers", ctx, mock.Anything).
		Return([]*fftypes.Verifier{
			{
				ID:        fftypes.NewUUID(),
				Identity:  idID,
				Namespace: "ns1",
				VerifierRef: fftypes.VerifierRef{
					Type:  fftypes.VerifierTypeEthAddress,
					Value: "fullkey123",
				},
			},
		}, nil, nil)

	msgIdentity := &fftypes.SignerRef{
		Author: "org1",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.NoError(t, err)
	assert.Equal(t, "did:firefly:org/org1", msgIdentity.Author)
	assert.Equal(t, "fullkey123", msgIdentity.Key)

	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityByOrgLookkupNotFound(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, fftypes.IdentityTypeOrg, fftypes.SystemNamespace, "org1").
		Return(nil, nil)

	msgIdentity := &fftypes.SignerRef{
		Author: "org1",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "FF10278", err)

	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityByOrgLookkupFail(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, fftypes.IdentityTypeOrg, fftypes.SystemNamespace, "org1").
		Return(nil, fmt.Errorf("pop"))

	msgIdentity := &fftypes.SignerRef{
		Author: "org1",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestResolveInputSigningIdentityByOrgVerifierFail(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	idID := fftypes.NewUUID()

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, fftypes.IdentityTypeOrg, fftypes.SystemNamespace, "org1").
		Return(&fftypes.Identity{
			IdentityBase: fftypes.IdentityBase{
				ID:        idID,
				DID:       "did:firefly:org/org1",
				Namespace: fftypes.SystemNamespace,
				Name:      "myid",
				Type:      fftypes.IdentityTypeOrg,
			},
		}, nil)
	mdi.On("GetVerifiers", ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	msgIdentity := &fftypes.SignerRef{
		Author: "org1",
	}
	err := im.ResolveInputSigningIdentity(ctx, "ns1", msgIdentity)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestFirstVerifierForIdentityNotFound(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			DID:       "did:firefly:org/org1",
			Namespace: fftypes.SystemNamespace,
			Name:      "myid",
			Type:      fftypes.IdentityTypeOrg,
		},
	}

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifiers", ctx, mock.Anything).Return([]*fftypes.Verifier{}, nil, nil)

	_, err := im.firstVerifierForIdentity(ctx, fftypes.VerifierTypeEthAddress, id)
	assert.Regexp(t, "FF10350", err)

	mdi.AssertExpectations(t)

}

func TestResolveNodeOwnerSigningIdentityNotFound(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	im.nodeOwnerBlockchainKey = &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeEthAddress,
		Value: "key12345",
	}
	config.Set(config.OrgName, "org1")

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "key12345").Return(nil, nil)

	err := im.ResolveNodeOwnerSigningIdentity(ctx, &fftypes.SignerRef{})
	assert.Regexp(t, "FF10281", err)

	mdi.AssertExpectations(t)

}

func TestGetNodeOwnerBlockchainKeyDeprecatedKeyResolveFailed(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	config.Set(config.OrgIdentityDeprecated, "0x12345")

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "0x12345").Return("", fmt.Errorf("pop"))

	_, err := im.GetNodeOwnerBlockchainKey(ctx)
	assert.Regexp(t, "pop", err)

	mbi.AssertExpectations(t)

}

func TestResolveBlockchainKeyEmptyRequest(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	_, err := im.ResolveBlockchainKey(ctx, "")
	assert.Regexp(t, "FF10349", err)

}

func TestResolveBlockchainKeyCached(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mbi := im.blockchain.(*blockchainmocks.Plugin)
	mbi.On("ResolveSigningKey", ctx, "0x12345").Return("resolved12345", nil).Once()

	v, err := im.ResolveBlockchainKey(ctx, "0x12345")
	assert.NoError(t, err)
	assert.Equal(t, fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeEthAddress,
		Value: "resolved12345",
	}, *v)

	v1, err := im.ResolveBlockchainKey(ctx, "0x12345")
	assert.NoError(t, err)
	assert.Equal(t, v, v1)

}

func TestGetNodeOwnerOrgCached(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	im.nodeOwningOrgIdentity = &fftypes.Identity{}

	id, err := im.GetNodeOwnerOrg(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, id)

}

func TestGetNodeOwnerOrgKeyNotSet(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	_, err := im.GetNodeOwnerOrg(ctx)
	assert.Regexp(t, "FF10351", err)

}

func TestGetNodeOwnerOrgMismatch(t *testing.T) {

	ctx, im := newTestIdentityManager(t)
	im.nodeOwnerBlockchainKey = &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeEthAddress,
		Value: "fullkey123",
	}
	config.Set(config.OrgName, "org1")

	orgID := fftypes.NewUUID()
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, "fullkey123").
		Return(&fftypes.Verifier{
			ID:        fftypes.NewUUID(),
			Identity:  orgID,
			Namespace: fftypes.SystemNamespace,
			VerifierRef: fftypes.VerifierRef{
				Type:  fftypes.VerifierTypeEthAddress,
				Value: "fullkey123",
			},
		}, nil)
	mdi.On("GetIdentityByID", ctx, orgID).
		Return(&fftypes.Identity{
			IdentityBase: fftypes.IdentityBase{
				ID:        orgID,
				DID:       "did:firefly:org/org2",
				Namespace: fftypes.SystemNamespace,
				Name:      "org2",
				Type:      fftypes.IdentityTypeOrg,
			},
		}, nil)

	_, err := im.GetNodeOwnerOrg(ctx)
	assert.Regexp(t, "FF10281", err)

}

func TestCachedIdentityLookupByVerifierRefCaching(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			DID:       "did:firefly:node/peer1",
			Namespace: fftypes.SystemNamespace,
			Name:      "peer1",
			Type:      fftypes.IdentityTypeOrg,
		},
	}
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeFFDXPeerID, fftypes.SystemNamespace, "peer1").
		Return(&fftypes.Verifier{
			ID:        fftypes.NewUUID(),
			Identity:  id.ID,
			Namespace: fftypes.SystemNamespace,
			VerifierRef: fftypes.VerifierRef{
				Type:  fftypes.VerifierTypeFFDXPeerID,
				Value: "peer1",
			},
		}, nil)
	mdi.On("GetIdentityByID", ctx, id.ID).
		Return(id, nil)

	v1, err := im.cachedIdentityLookupByVerifierRef(ctx, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	})
	assert.NoError(t, err)
	assert.Equal(t, id, v1)

	v2, err := im.cachedIdentityLookupByVerifierRef(ctx, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	})
	assert.NoError(t, err)
	assert.Equal(t, id, v2)

}

func TestCachedIdentityLookupByVerifierRefError(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			DID:       "did:firefly:node/peer1",
			Namespace: fftypes.SystemNamespace,
			Name:      "peer1",
			Type:      fftypes.IdentityTypeOrg,
		},
	}
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeFFDXPeerID, fftypes.SystemNamespace, "peer1").
		Return(&fftypes.Verifier{
			ID:        fftypes.NewUUID(),
			Identity:  id.ID,
			Namespace: fftypes.SystemNamespace,
			VerifierRef: fftypes.VerifierRef{
				Type:  fftypes.VerifierTypeFFDXPeerID,
				Value: "peer1",
			},
		}, nil)
	mdi.On("GetIdentityByID", ctx, id.ID).Return(nil, fmt.Errorf("pop"))

	_, err := im.cachedIdentityLookupByVerifierRef(ctx, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	})
	assert.Regexp(t, "pop", err)

}

func TestCachedIdentityLookupByVerifierRefNotFound(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			DID:       "did:firefly:node/peer1",
			Namespace: fftypes.SystemNamespace,
			Name:      "peer1",
			Type:      fftypes.IdentityTypeOrg,
		},
	}
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeFFDXPeerID, fftypes.SystemNamespace, "peer1").
		Return(&fftypes.Verifier{
			ID:        fftypes.NewUUID(),
			Identity:  id.ID,
			Namespace: fftypes.SystemNamespace,
			VerifierRef: fftypes.VerifierRef{
				Type:  fftypes.VerifierTypeFFDXPeerID,
				Value: "peer1",
			},
		}, nil)
	mdi.On("GetIdentityByID", ctx, id.ID).Return(nil, nil)

	_, err := im.cachedIdentityLookupByVerifierRef(ctx, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	})
	assert.Regexp(t, "FF10220", err)

}

func TestCachedIdentityLookupByDIDCaching(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			DID:       "did:firefly:node/peer1",
			Namespace: fftypes.SystemNamespace,
			Name:      "peer1",
			Type:      fftypes.IdentityTypeOrg,
		},
	}
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByDID", ctx, "did:firefly:node/peer1").Return(id, nil).Once()

	v1, err := im.cachedIdentityLookupByDID(ctx, "did:firefly:node/peer1")
	assert.NoError(t, err)
	assert.Equal(t, id, v1)

	v2, err := im.cachedIdentityLookupByDID(ctx, "did:firefly:node/peer1")
	assert.NoError(t, err)
	assert.Equal(t, id, v2)
}

func TestCachedIdentityLookupByDIDUnknownResolver(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	_, err := im.cachedIdentityLookupByDID(ctx, "did:random:anything")
	assert.Regexp(t, "FF10346", err)

}

func TestCachedIdentityLookupByDIDGetIDFail(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByDID", ctx, "did:firefly:node/peer1").Return(nil, fmt.Errorf("pop"))

	_, err := im.cachedIdentityLookupByDID(ctx, "did:firefly:node/peer1")
	assert.Regexp(t, "pop", err)
}

func TestCachedIdentityLookupByVerifierByOldDIDFail(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	orgUUID := fftypes.NewUUID()
	did := fftypes.FireFlyOrgDIDPrefix + orgUUID.String()

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByDID", ctx, did).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, mock.MatchedBy(func(uuid *fftypes.UUID) bool {
		return uuid.Equals(orgUUID)
	})).Return(nil, fmt.Errorf("pop"))

	_, err := im.cachedIdentityLookupByDID(ctx, did)
	assert.Regexp(t, "pop", err)

}

func TestCachedIdentityLookupByIDCaching(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			DID:       "did:firefly:node/peer1",
			Namespace: fftypes.SystemNamespace,
			Name:      "peer1",
			Type:      fftypes.IdentityTypeOrg,
		},
	}
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", ctx, id.ID).Return(id, nil).Once()

	v1, err := im.CachedIdentityLookupByID(ctx, id.ID)
	assert.NoError(t, err)
	assert.Equal(t, id, v1)

	v2, err := im.CachedIdentityLookupByID(ctx, id.ID)
	assert.NoError(t, err)
	assert.Equal(t, id, v2)

	mdi.AssertExpectations(t)
}

func TestVerifyIdentityChainCustomOrgOrgOk(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	idRoot := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			DID:       "did:firefly:org/org1",
			Namespace: fftypes.SystemNamespace,
			Name:      "org1",
			Type:      fftypes.IdentityTypeOrg,
		},
		Messages: fftypes.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	idIntermediateOrg := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Parent:    idRoot.ID,
			DID:       "did:firefly:org/org2",
			Namespace: fftypes.SystemNamespace,
			Name:      "org2",
			Type:      fftypes.IdentityTypeOrg,
		},
		Messages: fftypes.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	idIntermediateCustom := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Parent:    idIntermediateOrg.ID,
			DID:       "did:firefly:ns/ns1/custom1",
			Namespace: "ns1",
			Name:      "custom1",
			Type:      fftypes.IdentityTypeCustom,
		},
		Messages: fftypes.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	idLeaf := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Parent:    idIntermediateCustom.ID,
			DID:       "did:firefly:ns/ns1/custom2",
			Namespace: "ns1",
			Name:      "custom2",
			Type:      fftypes.IdentityTypeCustom,
		},
	}
	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", ctx, idIntermediateOrg.ID).Return(idIntermediateOrg, nil).Once()
	mdi.On("GetIdentityByID", ctx, idIntermediateCustom.ID).Return(idIntermediateCustom, nil).Once()
	mdi.On("GetIdentityByID", ctx, idRoot.ID).Return(idRoot, nil).Once()

	immeidateParent, err := im.VerifyIdentityChain(ctx, idLeaf)
	assert.Equal(t, idIntermediateCustom, immeidateParent)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestVerifyIdentityInvalid(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id1 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{},
	}

	_, err := im.VerifyIdentityChain(ctx, id1)
	assert.Regexp(t, "FF10203", err)

}

func TestVerifyIdentityChainLoop(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	idID1 := fftypes.NewUUID()
	idID2 := fftypes.NewUUID()
	id1 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        idID1,
			Parent:    idID2,
			DID:       "did:firefly:org/org1",
			Namespace: fftypes.SystemNamespace,
			Name:      "org1",
			Type:      fftypes.IdentityTypeOrg,
		},
		Messages: fftypes.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}
	id2 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        idID2,
			Parent:    idID1,
			DID:       "did:firefly:org/org2",
			Namespace: fftypes.SystemNamespace,
			Name:      "org2",
			Type:      fftypes.IdentityTypeOrg,
		},
		Messages: fftypes.IdentityMessages{
			Claim: fftypes.NewUUID(),
		},
	}

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", ctx, idID2).Return(id2, nil).Once()

	_, err := im.VerifyIdentityChain(ctx, id1)
	assert.Regexp(t, "FF10361", err)

	mdi.AssertExpectations(t)
}

func TestVerifyIdentityChainBadParent(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	idID1 := fftypes.NewUUID()
	idID2 := fftypes.NewUUID()
	id1 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        idID1,
			Parent:    idID2,
			DID:       "did:firefly:org/org1",
			Namespace: fftypes.SystemNamespace,
			Name:      "org1",
			Type:      fftypes.IdentityTypeOrg,
		},
	}
	id2 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        idID2,
			DID:       "did:firefly:org/org2",
			Namespace: fftypes.SystemNamespace,
			Name:      "org2",
			Type:      fftypes.IdentityTypeOrg,
		},
	}

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", ctx, idID2).Return(id2, nil).Once()

	_, err := im.VerifyIdentityChain(ctx, id1)
	assert.Regexp(t, "FF10363", err)

	mdi.AssertExpectations(t)
}

func TestVerifyIdentityChainErr(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	idID2 := fftypes.NewUUID()
	id1 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Parent:    idID2,
			DID:       "did:firefly:org/org1",
			Namespace: fftypes.SystemNamespace,
			Name:      "org1",
			Type:      fftypes.IdentityTypeOrg,
		},
	}

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", ctx, idID2).Return(nil, fmt.Errorf("pop"))

	_, err := im.VerifyIdentityChain(ctx, id1)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestVerifyIdentityChainNotFound(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	idID2 := fftypes.NewUUID()
	id1 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Parent:    idID2,
			DID:       "did:firefly:org/org1",
			Namespace: fftypes.SystemNamespace,
			Name:      "org1",
			Type:      fftypes.IdentityTypeOrg,
		},
	}

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", ctx, idID2).Return(nil, nil)

	_, err := im.VerifyIdentityChain(ctx, id1)
	assert.Regexp(t, "FF10214", err)

	mdi.AssertExpectations(t)
}

func TestVerifyIdentityChainInvalidParent(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id1 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Parent:    nil,
			DID:       "did:firefly:ns/ns1/custom1",
			Namespace: "ns1",
			Name:      "custom1",
			Type:      fftypes.IdentityTypeCustom,
		},
	}
	id2 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Parent:    id1.ID,
			DID:       "did:firefly:org/org2",
			Namespace: fftypes.SystemNamespace,
			Name:      "org2",
			Type:      fftypes.IdentityTypeOrg,
		},
	}

	mdi := im.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", ctx, id1.ID).Return(id1, nil).Once()

	_, err := im.VerifyIdentityChain(ctx, id2)
	assert.Regexp(t, "FF10362", err)

	mdi.AssertExpectations(t)
}

func TestValidateParentTypeCustomToNode(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id1 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			Type: fftypes.IdentityTypeNode,
		},
	}
	id2 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			Type: fftypes.IdentityTypeCustom,
		},
	}

	err := im.validateParentType(ctx, id2, id1)
	assert.Regexp(t, "FF10362", err)

}

func TestValidateParentTypeInvalidType(t *testing.T) {

	ctx, im := newTestIdentityManager(t)

	id1 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			Type: fftypes.IdentityTypeCustom,
		},
	}
	id2 := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			Type: fftypes.IdentityType("unknown"),
		},
	}

	err := im.validateParentType(ctx, id2, id1)
	assert.Regexp(t, "FF10359", err)

}
