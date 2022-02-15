// Copyright © 2022 Kaleido, Inc.
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
	"strings"
	"time"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/identity"
	"github.com/karlseguin/ccache"
)

type Manager interface {
	ResolveInputIdentity(ctx context.Context, identity *fftypes.IdentityRef) (err error)
	ResolveSigningKey(ctx context.Context, inputKey string) (outputKey string, err error)
	ResolveSigningKeyIdentity(ctx context.Context, signingKey string) (author string, err error)
	ResolveLocalOrgDID(ctx context.Context) (localOrgDID string, err error)
	GetLocalOrgKey(ctx context.Context) (string, error)
	OrgDID(org *fftypes.Organization) string
	GetLocalOrganization(ctx context.Context) (*fftypes.Organization, error)
}

type identityManager struct {
	database   database.Plugin
	plugin     identity.Plugin
	blockchain blockchain.Plugin

	localOrgSigningKey string
	localOrgDID        string
	identityCacheTTL   time.Duration
	identityCache      *ccache.Cache
	signingKeyCacheTTL time.Duration
	signingKeyCache    *ccache.Cache
}

func NewIdentityManager(ctx context.Context, di database.Plugin, ii identity.Plugin, bi blockchain.Plugin) (Manager, error) {
	if di == nil || ii == nil || bi == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	im := &identityManager{
		database:           di,
		plugin:             ii,
		blockchain:         bi,
		identityCacheTTL:   config.GetDuration(config.IdentityManagerCacheTTL),
		signingKeyCacheTTL: config.GetDuration(config.IdentityManagerCacheTTL),
	}
	// For the identity and signingkey caches, we just treat them all equally sized and the max items
	im.identityCache = ccache.New(
		ccache.Configure().MaxSize(config.GetInt64(config.IdentityManagerCacheLimit)),
	)
	im.signingKeyCache = ccache.New(
		ccache.Configure().MaxSize(config.GetInt64(config.IdentityManagerCacheLimit)),
	)

	return im, nil
}

func (im *identityManager) GetLocalOrganization(ctx context.Context) (*fftypes.Organization, error) {
	orgDID, err := im.ResolveLocalOrgDID(ctx)
	if err != nil {
		return nil, err
	}
	return im.cachedOrgLookupByAuthor(ctx, orgDID)
}

func (im *identityManager) OrgDID(org *fftypes.Organization) string {
	return org.GetDID()
}

// ResolveInputIdentity takes in identity input information from an API call, or configuration load, and resolves
// the combination
func (im *identityManager) ResolveInputIdentity(ctx context.Context, identity *fftypes.IdentityRef) (err error) {
	log.L(ctx).Debugf("Resolving identity input: key='%s' author='%s'", identity.Key, identity.Author)

	identity.Key, err = im.ResolveSigningKey(ctx, identity.Key)
	if err != nil {
		return err
	}

	// Resolve the identity
	if err = im.resolveInputAuthor(ctx, identity); err != nil {
		return err
	}

	log.L(ctx).Debugf("Resolved identity: key='%s' author='%s'", identity.Key, identity.Author)
	return
}

func (im *identityManager) ResolveSigningKeyIdentity(ctx context.Context, signingKey string) (author string, err error) {

	signingKey, err = im.ResolveSigningKey(ctx, signingKey)
	if err != nil {
		return "", err
	}

	// TODO: Consider other ways identity could be resolved
	org, err := im.cachedOrgLookupBySigningKey(ctx, signingKey)
	if err != nil {
		return "", err
	}

	return im.OrgDID(org), nil

}

func (im *identityManager) getConfigOrgKey() string {
	orgKey := config.GetString(config.OrgKey)
	if orgKey == "" {
		orgKey = config.GetString(config.OrgIdentityDeprecated)
	}
	return orgKey
}

func (im *identityManager) GetLocalOrgKey(ctx context.Context) (string, error) {
	if im.localOrgSigningKey != "" {
		return im.localOrgSigningKey, nil
	}
	resolvedSigningKey, err := im.blockchain.ResolveSigningKey(ctx, im.getConfigOrgKey())
	if err != nil {
		return "", err
	}
	im.localOrgSigningKey = resolvedSigningKey
	return im.localOrgSigningKey, nil
}

func (im *identityManager) ResolveLocalOrgDID(ctx context.Context) (localOrgDID string, err error) {
	if im.localOrgDID != "" {
		return im.localOrgDID, nil
	}
	orgKey := im.getConfigOrgKey()

	im.localOrgDID, err = im.ResolveSigningKeyIdentity(ctx, orgKey)
	if err != nil {
		return "", i18n.WrapError(ctx, err, i18n.MsgLocalOrgLookupFailed, orgKey)
	}
	if im.localOrgDID == "" {
		return "", i18n.NewError(ctx, i18n.MsgLocalOrgLookupFailed, orgKey)
	}
	return im.localOrgDID, err
}

func (im *identityManager) ResolveSigningKey(ctx context.Context, inputKey string) (outputKey string, err error) {
	// Resolve the signing key
	if inputKey != "" {
		if cached := im.signingKeyCache.Get(inputKey); cached != nil {
			cached.Extend(im.identityCacheTTL)
			outputKey = cached.Value().(string)
		} else {
			outputKey, err = im.blockchain.ResolveSigningKey(ctx, inputKey)
			if err != nil {
				return "", err
			}
			im.signingKeyCache.Set(inputKey, outputKey, im.identityCacheTTL)
		}
	} else {
		return im.localOrgSigningKey, nil
	}
	return
}

func (im *identityManager) cachedIdentityLookupBySigningKey(ctx context.Context, iType fftypes.VerifierType, namespace, signingKey string) (identity *fftypes.Identity, err error) {
	cacheKey := fmt.Sprintf("key=%s|%s|%s", iType, namespace, signingKey)
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		identity = cached.Value().(*fftypes.Identity)
	} else {
		verifier, err := im.database.GetVerifierByValue(ctx, iType, namespace, signingKey)
		if err != nil || verifier == nil {
			return nil, err
		}
		identity, err = im.database.GetIdentityByID(ctx, verifier.Identity)
		if err != nil || identity == nil {
			return nil, err
		}
		// Cache the result
		im.identityCache.Set(cacheKey, identity, im.identityCacheTTL)
	}
	return identity, nil
}

func (im *identityManager) cachedIdentityLookupByDID(ctx context.Context, namespace, did string) (identity *fftypes.Identity, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	cacheKey := fmt.Sprintf("did=%s|%s|%s", iType, namespace, did)
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		identity = cached.Value().(*fftypes.Identity)
	} else {
		if !strings.HasPrefix(did, fftypes.DIDPrefix) {
			if !strings.HasPrefix(did, fftypes.FireFlyDIDPrefix) {
				return nil, i18n.NewError(ctx, i18n.MsgDIDResovlerUnknown)
			}
			// Look up by the full DID
			if identity, err = im.database.GetIdentityByDID(ctx, namespace, did); err != nil {
				return nil, err
			}
			if identity == nil && strings.HasPrefix(did, fftypes.FireFlyOrgDIDPrefix) {
				// We allow the org UUID to be used to resolve organization DIDs as an alias to the name
				// This is historical, for when we first introduced Organization DIDs
				orgUUID, err := fftypes.ParseUUID(ctx, strings.TrimPrefix(namespace, fftypes.FireFlyOrgDIDPrefix))
				if err == nil {
					if identity, err = im.database.GetIdentityByID(ctx, orgUUID); err != nil {
						return nil, err
					}
				}
			}
			if identity == nil {
				return nil, i18n.NewError(ctx, i18n.MsgAuthorNotFoundByDID, did)
			}
		} else {
			// If there is just a name in there, then it could be an Org type identity (from the very original usage of the field)
			if identity, err = im.database.GetIdentityByName(ctx, fftypes.IdentityTypeOrg, fftypes.SystemNamespace, did); err != nil {
				return nil, err
			}
			if identity == nil {
				return nil, i18n.NewError(ctx, i18n.MsgAuthorOrgNotFoundByName, did)
			}
		}

		// Cache the result
		im.identityCache.Set(cacheKey, org, im.identityCacheTTL)
	}
	return org, nil
}

func (im *identityManager) resolveInputAuthor(ctx context.Context, identity *fftypes.IdentityRef) (err error) {

	var org *fftypes.Organization
	if identity.Author == "" {
		// We allow lookup of an org by signing key (this convenience mechanism is currently not cached)
		if identity.Key != "" {
			if org, err = im.database.GetOrganizationByIdentity(ctx, identity.Key); err != nil {
				return err
			}
		}
		if org == nil {
			// Otherwise default to the org identity that owns this node, if no input author specified
			identity.Author = config.GetString(config.OrgName)
		}
	}

	if org == nil {
		if org, err = im.cachedOrgLookupByAuthor(ctx, identity.Author); err != nil {
			return err
		}
	}

	// TODO: Organizations should be able to have multiple signing keys. See notes below about whether a level of
	//       indirection is needed in front of orgs (likely it is).
	if identity.Key == "" {
		identity.Key = org.Identity
	} else if org.Identity != identity.Key {
		return i18n.NewError(ctx, i18n.MsgAuthorOrgSigningKeyMismatch, org.ID, identity.Key)
	}

	// We normalize the author to the DID
	identity.Author = im.OrgDID(org)
	return nil

}
