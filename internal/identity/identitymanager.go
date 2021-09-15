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
	"strings"
	"time"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/identity"
	"github.com/karlseguin/ccache"
)

const (
	fireflyOrgDIDPrefix = "did:firefly:org/"
)

type Manager interface {
	ResolveInputIdentity(ctx context.Context, identity *fftypes.Identity) (err error)
}

type identityManager struct {
	database   database.Plugin
	plugin     identity.Plugin
	blockchain blockchain.Plugin

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

// ResolveInputIdentity takes in identity input information from an API call, or configuration load, and resolves
// the combination
func (im *identityManager) ResolveInputIdentity(ctx context.Context, identity *fftypes.Identity) (err error) {
	log.L(ctx).Debugf("Resolving identity input: key='%s' author='%s'", identity.Key, identity.Author)

	if identity.Key != "" {
		err = im.resolveSigningKey(ctx, identity)
		if err != nil {
			return err
		}
	}

	// Resolve the identity
	if err = im.resolveInputAuthor(ctx, identity); err != nil {
		return err
	}

	log.L(ctx).Debugf("Resolved identity: key='%s' author='%s'", identity.Key, identity.Author)
	return
}

func (im *identityManager) cachedOrgLookupByAuthor(ctx context.Context, author string) (org *fftypes.Organization, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	if cached := im.identityCache.Get(author); cached != nil {
		cached.Extend(im.identityCacheTTL)
		org = cached.Value().(*fftypes.Organization)
	} else {
		// TODO: Per comments in https://github.com/hyperledger-labs/firefly/issues/187 we need to resolve whether "Organization"
		//       is the right thing to resolve here. We might want to fall-back to that in the case of plain string, but likely
		//       we need something more sophisticated here where we have an Identity object in the database.
		if strings.HasPrefix(author, fireflyOrgDIDPrefix) {
			orgUUID, err := fftypes.ParseUUID(ctx, strings.TrimPrefix(author, fireflyOrgDIDPrefix))
			if err != nil {
				return nil, err
			}
			if org, err = im.database.GetOrganizationByID(ctx, orgUUID); err != nil {
				return nil, err
			}
			if org == nil {
				return nil, i18n.NewError(ctx, i18n.MsgAuthorNotFoundByDID, author)
			}
		} else {
			if org, err = im.database.GetOrganizationByName(ctx, author); err != nil {
				return nil, err
			}
			if org == nil {
				return nil, i18n.NewError(ctx, i18n.MsgAuthorOrgNotFoundByName, author)
			}
		}

		// Cache the result
		im.identityCache.Set(author, org, im.identityCacheTTL)
	}
	return org, nil
}

func (im *identityManager) resolveInputAuthor(ctx context.Context, identity *fftypes.Identity) (err error) {

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
	identity.Author = fmt.Sprintf("%s%s", fireflyOrgDIDPrefix, org.ID)
	return nil

}

func (im *identityManager) resolveSigningKey(ctx context.Context, identity *fftypes.Identity) (err error) {
	// Resolve the signing key
	if identity.Key != "" {
		inputKey := identity.Key
		if cached := im.signingKeyCache.Get(inputKey); cached != nil {
			cached.Extend(im.identityCacheTTL)
			identity.Key = cached.Value().(string)
		} else {
			identity.Key, err = im.blockchain.ResolveSigningKey(ctx, inputKey)
			if err != nil {
				return err
			}
			im.signingKeyCache.Set(inputKey, identity.Key, im.identityCacheTTL)
		}
	}

	return
}
