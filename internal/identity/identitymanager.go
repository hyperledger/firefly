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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/identity"
	"github.com/karlseguin/ccache"
)

type Manager interface {
	ResolveInputSigningIdentity(ctx context.Context, namespace string, msgSignerRef *fftypes.SignerRef) (err error)
	ResolveRootOrgRegistrationSigningKey(ctx context.Context, namespace string, msgSignerRef *fftypes.SignerRef) (err error)
	ResolveNodeOwnerSigningIdentity(ctx context.Context, msgSignerRef *fftypes.SignerRef) (err error)
	ResolveBlockchainKey(ctx context.Context, inputKey string) (verifier *fftypes.VerifierRef, err error)
	FindIdentityForVerifier(ctx context.Context, iTypes []fftypes.IdentityType, namespace string, verifier *fftypes.VerifierRef) (identity *fftypes.Identity, err error)
	CachedIdentityLookupByID(ctx context.Context, id *fftypes.UUID) (identity *fftypes.Identity, err error)
	CachedIdentityLookup(ctx context.Context, did string) (identity *fftypes.Identity, err error)
	CachedVerifierLookup(ctx context.Context, vType fftypes.VerifierType, ns, value string) (verifier *fftypes.Verifier, err error)
	GetNodeOwnerBlockchainKey(ctx context.Context) (*fftypes.VerifierRef, error)
	GetNodeOwnerOrg(ctx context.Context) (*fftypes.Identity, error)
	VerifyIdentityChain(ctx context.Context, identity *fftypes.Identity) (immediateParent *fftypes.Identity, err error)
	IsRootOrgBroadcast(ctx context.Context, msg *fftypes.Message, data ...*fftypes.Data) bool
}

type identityManager struct {
	database   database.Plugin
	plugin     identity.Plugin
	blockchain blockchain.Plugin
	data       data.Manager

	nodeOwnerBlockchainKey *fftypes.VerifierRef
	nodeOwningOrgIdentity  *fftypes.Identity
	identityCacheTTL       time.Duration
	identityCache          *ccache.Cache
	signingKeyCacheTTL     time.Duration
	signingKeyCache        *ccache.Cache
}

func NewIdentityManager(ctx context.Context, di database.Plugin, ii identity.Plugin, bi blockchain.Plugin, dm data.Manager) (Manager, error) {
	if di == nil || ii == nil || bi == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	im := &identityManager{
		database:           di,
		plugin:             ii,
		blockchain:         bi,
		data:               dm,
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

func (im *identityManager) ResolveInputSigningIdentity(ctx context.Context, namespace string, msgSignerRef *fftypes.SignerRef) (err error) {
	return im.resolveInputSigningIdentity(ctx, namespace, msgSignerRef, false)
}

func (im *identityManager) ResolveRootOrgRegistrationSigningKey(ctx context.Context, namespace string, msgSignerRef *fftypes.SignerRef) (err error) {
	return im.resolveInputSigningIdentity(ctx, namespace, msgSignerRef, true)
}

// ResolveInputIdentity takes in blockchain signing input information from an API call,
// and resolves the final information that should be written in the message etc..
func (im *identityManager) resolveInputSigningIdentity(ctx context.Context, namespace string, msgSignerRef *fftypes.SignerRef, rootRegistration bool) (err error) {
	log.L(ctx).Debugf("Resolving identity input: key='%s' author='%s'", msgSignerRef.Key, msgSignerRef.Author)

	var verifier *fftypes.VerifierRef
	switch {
	case msgSignerRef.Author == "" && msgSignerRef.Key == "":
		err = im.ResolveNodeOwnerSigningIdentity(ctx, msgSignerRef)
		if err != nil {
			return err
		}
	case msgSignerRef.Key != "":
		if verifier, err = im.ResolveBlockchainKey(ctx, msgSignerRef.Key); err != nil {
			return err
		}
		msgSignerRef.Key = verifier.Value
		// Fill in or verify the author DID based on the verfier, if it's been registered
		identity, err := im.FindIdentityForVerifier(ctx, []fftypes.IdentityType{
			fftypes.IdentityTypeOrg,
			fftypes.IdentityTypeCustom,
		}, namespace, verifier)
		if err != nil {
			return err
		}
		switch {
		case identity != nil:
			if msgSignerRef.Author == identity.Name || msgSignerRef.Author == "" {
				// Switch to full DID automatically
				msgSignerRef.Author = identity.DID
			}
			if msgSignerRef.Author != identity.DID {
				return i18n.NewError(ctx, i18n.MsgAuthorRegistrationMismatch, verifier.Value, msgSignerRef.Author, identity.DID)
			}
		case msgSignerRef.Author != "":
			if rootRegistration {
				if namespace != fftypes.SystemNamespace || !strings.HasPrefix(msgSignerRef.Author, fftypes.FireFlyOrgDIDPrefix) {
					return i18n.NewError(ctx, i18n.MsgAuthorIncorrectForRootReg, namespace, msgSignerRef.Author)
				}
			} else {
				identity, err := im.CachedIdentityLookup(ctx, msgSignerRef.Author)
				if err != nil {
					return err
				}
				msgSignerRef.Author = identity.DID
			}
		default:
			return i18n.NewError(ctx, i18n.MsgAuthorMissingForKey, msgSignerRef.Key)
		}
	case msgSignerRef.Author != "":
		// Author must be non-empty (see above), so we want to find that identity and then
		// use the first blockchain key that's associated with it.
		identity, err := im.CachedIdentityLookup(ctx, msgSignerRef.Author)
		if err != nil {
			return err
		}
		msgSignerRef.Author = identity.DID
		verifier, err = im.firstVerifierForIdentity(ctx, im.blockchain.VerifierType(), identity)
		if err != nil {
			return err
		}
		msgSignerRef.Key = verifier.Value
	}

	log.L(ctx).Debugf("Resolved identity: key='%s' author='%s'", msgSignerRef.Key, msgSignerRef.Author)
	return nil
}

// firstVerifierForIdentity does a lookup of the first verifier of a given type (such as a blockchain signing key) registered to an identity,
// as a convenience to allow you to only specify the org name/DID when sending a message
func (im *identityManager) firstVerifierForIdentity(ctx context.Context, vType fftypes.VerifierType, identity *fftypes.Identity) (*fftypes.VerifierRef, error) {
	fb := database.VerifierQueryFactory.NewFilterLimit(ctx, 1)
	filter := fb.And(
		fb.Eq("type", vType),
		fb.Eq("identity", identity.ID),
	)
	verifiers, _, err := im.database.GetVerifiers(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(verifiers) == 0 {
		return nil, i18n.NewError(ctx, i18n.MsgNoVerifierForIdentity, vType, identity.DID)
	}
	return &verifiers[0].VerifierRef, nil
}

// ResolveNodeOwnerSigningIdentity add the node owner identity into a message
func (im *identityManager) ResolveNodeOwnerSigningIdentity(ctx context.Context, msgSignerRef *fftypes.SignerRef) (err error) {
	verifierRef, err := im.GetNodeOwnerBlockchainKey(ctx)
	if err != nil {
		return err
	}
	identity, err := im.GetNodeOwnerOrg(ctx)
	if err != nil {
		return err
	}
	msgSignerRef.Author = identity.DID
	msgSignerRef.Key = verifierRef.Value
	return nil
}

// GetNodeOwnerBlockchainKey gets the blockchain key of the node owner, from the configuration
func (im *identityManager) GetNodeOwnerBlockchainKey(ctx context.Context) (*fftypes.VerifierRef, error) {
	if im.nodeOwnerBlockchainKey != nil {
		return im.nodeOwnerBlockchainKey, nil
	}

	orgKey := config.GetString(config.OrgKey)
	if orgKey == "" {
		orgKey = config.GetString(config.OrgIdentityDeprecated)
		if orgKey != "" {
			log.L(ctx).Warnf("The %s config key has been deprecated. Please use %s instead", config.OrgIdentityDeprecated, config.OrgKey)
		}
	}
	if orgKey == "" {
		return nil, i18n.NewError(ctx, i18n.MsgNodeMissingBlockchainKey)
	}

	verifier, err := im.ResolveBlockchainKey(ctx, orgKey)
	if err != nil {
		return nil, err
	}
	im.nodeOwnerBlockchainKey = verifier
	return im.nodeOwnerBlockchainKey, nil
}

// ResolveBlockchainKey does a cached lookup of the fully qualified key, associated with a key reference string
func (im *identityManager) ResolveBlockchainKey(ctx context.Context, inputKey string) (verifier *fftypes.VerifierRef, err error) {
	if inputKey == "" {
		return nil, i18n.NewError(ctx, i18n.MsgBlockchainKeyNotSet)
	}
	if cached := im.signingKeyCache.Get(inputKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		return cached.Value().(*fftypes.VerifierRef), nil
	}
	keyString, err := im.blockchain.ResolveSigningKey(ctx, inputKey)
	if err != nil {
		return nil, err
	}
	verifier = &fftypes.VerifierRef{
		Type:  im.blockchain.VerifierType(),
		Value: keyString,
	}
	im.signingKeyCache.Set(inputKey, verifier, im.identityCacheTTL)
	return verifier, nil
}

// FindIdentityForVerifier is a reverse lookup function to look up an identity registered as owner of the specified verifier.
// Each of the supplied identity types will be checked in order. Returns nil if not found
func (im *identityManager) FindIdentityForVerifier(ctx context.Context, iTypes []fftypes.IdentityType, namespace string, verifier *fftypes.VerifierRef) (identity *fftypes.Identity, err error) {
	for _, iType := range iTypes {
		verifierNS := namespace
		if iType != fftypes.IdentityTypeCustom {
			// Non-custom identity types are always in the system namespace
			verifierNS = fftypes.SystemNamespace
		}
		identity, err = im.cachedIdentityLookupByVerifierRef(ctx, verifierNS, verifier)
		if err != nil || identity != nil {
			return identity, err
		}
	}
	return nil, nil
}

// GetNodeOwnerOrg returns the identity of the organization that owns the node, if fully registered
func (im *identityManager) GetNodeOwnerOrg(ctx context.Context) (*fftypes.Identity, error) {
	if im.nodeOwningOrgIdentity != nil {
		return im.nodeOwningOrgIdentity, nil
	}
	verifierRef, err := im.GetNodeOwnerBlockchainKey(ctx)
	if err != nil {
		return nil, err
	}
	orgName := config.GetString(config.OrgName)
	identity, err := im.cachedIdentityLookupByVerifierRef(ctx, fftypes.SystemNamespace, verifierRef)
	if err != nil || identity == nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgLocalOrgLookupFailed, orgName, verifierRef.Value)
	}
	// Confirm that the specified blockchain key is associated with the correct org
	if identity.Type != fftypes.IdentityTypeOrg || identity.Name != orgName {
		return nil, i18n.NewError(ctx, i18n.MsgLocalOrgLookupFailed, orgName, verifierRef.Value)
	}
	im.nodeOwningOrgIdentity = identity
	return im.nodeOwningOrgIdentity, nil
}

func (im *identityManager) VerifyIdentityChain(ctx context.Context, checkIdentity *fftypes.Identity) (immediateParent *fftypes.Identity, err error) {

	loopDetect := make(map[fftypes.UUID]bool)
	current := checkIdentity
	for {
		err := current.IdentityBase.Validate(ctx)
		if err != nil {
			return nil, err
		}
		loopDetect[*current.ID] = true
		parentID := current.Parent
		if parentID == nil {
			return immediateParent, nil
		}
		if _, ok := loopDetect[*parentID]; ok {
			return nil, i18n.NewError(ctx, i18n.MsgIdentityChainLoop, parentID, current.DID, current.ID)
		}
		parent, err := im.CachedIdentityLookupByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, i18n.NewError(ctx, i18n.MsgParentIdentityNotFound, parentID, current.DID, current.ID)
		}
		if err := im.validateParentType(ctx, current, parent); err != nil {
			return nil, err
		}
		if parent.Messages.Claim == nil {
			return nil, i18n.NewError(ctx, i18n.MsgParentIdentityMissingClaim, parent.DID, parent.ID)
		}
		current = parent
		if immediateParent == nil {
			immediateParent = parent
		}
	}

}

func (im *identityManager) validateParentType(ctx context.Context, child *fftypes.Identity, parent *fftypes.Identity) error {

	switch child.Type {
	case fftypes.IdentityTypeNode, fftypes.IdentityTypeOrg:
		if parent.Type != fftypes.IdentityTypeOrg {
			return i18n.NewError(ctx, i18n.MsgInvalidIdentityParentType, parent.DID, parent.ID, parent.Type, child.DID, child.ID, child.Type)
		}
		return nil
	case fftypes.IdentityTypeCustom:
		if parent.Type != fftypes.IdentityTypeOrg && parent.Type != fftypes.IdentityTypeCustom {
			return i18n.NewError(ctx, i18n.MsgInvalidIdentityParentType, parent.DID, parent.ID, parent.Type, child.DID, child.ID, child.Type)
		}
		return nil
	default:
		return i18n.NewError(ctx, i18n.MsgUnknownIdentityType, child.Type)
	}

}

func (im *identityManager) cachedIdentityLookupByVerifierRef(ctx context.Context, namespace string, verifierRef *fftypes.VerifierRef) (*fftypes.Identity, error) {
	cacheKey := fmt.Sprintf("key=%s|%s|%s", namespace, verifierRef.Type, verifierRef.Value)
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		return cached.Value().(*fftypes.Identity), nil
	}
	verifier, err := im.database.GetVerifierByValue(ctx, verifierRef.Type, namespace, verifierRef.Value)
	if err != nil || verifier == nil {
		return nil, err
	}
	identity, err := im.database.GetIdentityByID(ctx, verifier.Identity)
	if err != nil {
		return nil, err
	}
	if identity == nil {
		return nil, i18n.NewError(ctx, i18n.MsgEmptyMemberIdentity, verifier.Identity)
	}
	// Cache the result
	im.identityCache.Set(cacheKey, identity, im.identityCacheTTL)
	return identity, nil
}

func (im *identityManager) CachedIdentityLookup(ctx context.Context, didLookupStr string) (identity *fftypes.Identity, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	cacheKey := fmt.Sprintf("did=%s", didLookupStr)
	defer func() {
		log.L(ctx).Debugf("Resolved DID '%s' to identity: %v (err=%v)", didLookupStr, identity, err)
	}()
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		identity = cached.Value().(*fftypes.Identity)
	} else {
		if strings.HasPrefix(didLookupStr, fftypes.DIDPrefix) {
			if !strings.HasPrefix(didLookupStr, fftypes.FireFlyDIDPrefix) {
				return nil, i18n.NewError(ctx, i18n.MsgDIDResovlerUnknown, didLookupStr)
			}
			// Look up by the full DID
			if identity, err = im.database.GetIdentityByDID(ctx, didLookupStr); err != nil {
				return nil, err
			}
			if identity == nil && strings.HasPrefix(didLookupStr, fftypes.FireFlyOrgDIDPrefix) {
				// We allow the UUID to be used to resolve DIDs as an alias to the name
				uuid, err := fftypes.ParseUUID(ctx, strings.TrimPrefix(didLookupStr, fftypes.FireFlyOrgDIDPrefix))
				if err == nil {
					if identity, err = im.database.GetIdentityByID(ctx, uuid); err != nil {
						return nil, err
					}
				}
			}
			if identity == nil {
				return nil, i18n.NewError(ctx, i18n.MsgIdentityNotFoundByString, didLookupStr)
			}
		} else {
			// If there is just a name in there, then it could be an Org type identity (from the very original usage of the field)
			if identity, err = im.database.GetIdentityByName(ctx, fftypes.IdentityTypeOrg, fftypes.SystemNamespace, didLookupStr); err != nil {
				return nil, err
			}
			if identity == nil {
				return nil, i18n.NewError(ctx, i18n.MsgAuthorOrgNotFoundByName, didLookupStr)
			}
		}

		// Cache the result
		im.identityCache.Set(cacheKey, identity, im.identityCacheTTL)
	}
	return identity, nil
}

func (im *identityManager) CachedIdentityLookupByID(ctx context.Context, id *fftypes.UUID) (identity *fftypes.Identity, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	cacheKey := fmt.Sprintf("id=%s", id)
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		identity = cached.Value().(*fftypes.Identity)
	} else {
		identity, err = im.database.GetIdentityByID(ctx, id)
		if err != nil || identity == nil {
			return identity, err
		}
		// Cache the result
		im.identityCache.Set(cacheKey, identity, im.identityCacheTTL)
	}
	return identity, nil
}

func (im *identityManager) CachedVerifierLookup(ctx context.Context, vType fftypes.VerifierType, ns, value string) (verifier *fftypes.Verifier, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	cacheKey := fmt.Sprintf("v=%s|%s|%s", vType, ns, value)
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		verifier = cached.Value().(*fftypes.Verifier)
	} else {
		verifier, err = im.database.GetVerifierByValue(ctx, vType, ns, value)
		if err != nil || verifier == nil {
			return verifier, err
		}
		// Cache the result
		im.identityCache.Set(cacheKey, verifier, im.identityCacheTTL)
	}
	return verifier, nil
}

func (im *identityManager) isValidRootOrgIdentityClaim(ctx context.Context, msg *fftypes.Message, data *fftypes.Data) bool {
	var claim *fftypes.IdentityClaim
	if data.Value != nil {
		err := json.Unmarshal([]byte(*data.Value), &claim)
		if err != nil {
			return false
		}
	}
	return im.isValidRootOrgCommon(ctx, msg, claim.Identity)
}

func (im *identityManager) isValidRootOrgDeprecated(ctx context.Context, msg *fftypes.Message, data *fftypes.Data) bool {
	var org *fftypes.Identity
	if data.Value != nil {
		err := json.Unmarshal([]byte(*data.Value), &org)
		if err != nil {
			return false
		}
	}
	org.DID, _ = org.GenerateDID(ctx)
	return im.isValidRootOrgCommon(ctx, msg, org)
}

func (im *identityManager) isValidRootOrgCommon(ctx context.Context, msg *fftypes.Message, org *fftypes.Identity) bool {
	if org == nil || org.Parent != nil {
		return false
	}
	err := org.Validate(ctx)
	if err != nil {
		log.L(ctx).Warnf("Invalid org broadcast '%s': %s", msg.Header.ID, err)
		return false
	}
	return true
}

func (im *identityManager) IsRootOrgBroadcast(ctx context.Context, msg *fftypes.Message, data ...*fftypes.Data) bool {
	// Look into message to see if it contains a data item that is a root organization definition
	if msg.Header.Type == fftypes.MessageTypeDefinition &&
		(msg.Header.Tag == fftypes.DeprecatedSystemTagDefineOrganization || msg.Header.Tag == fftypes.SystemTagIdentityClaim) {
		messageData := data
		if len(messageData) == 0 {
			data, ok, err := im.data.GetMessageData(ctx, msg, true)
			if !ok || err != nil {
				return false
			}
			messageData = data
		}
		if len(messageData) > 0 {
			dataItem := messageData[0]
			if dataItem.Validator == fftypes.MessageTypeDefinition {
				if msg.Header.Tag == fftypes.DeprecatedSystemTagDefineOrganization {
					return im.isValidRootOrgDeprecated(ctx, msg, dataItem)
				}
				return im.isValidRootOrgIdentityClaim(ctx, msg, dataItem)
			}
		}
	}
	return false
}
