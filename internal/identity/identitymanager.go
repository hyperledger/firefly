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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/namespace"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/identity"
	"github.com/karlseguin/ccache"
)

const (
	KeyNormalizationBlockchainPlugin = iota
	KeyNormalizationNone
)

type Manager interface {
	ResolveInputSigningIdentity(ctx context.Context, namespace string, signerRef *core.SignerRef) (err error)
	ResolveNodeOwnerSigningIdentity(ctx context.Context, namespace string, signerRef *core.SignerRef) (err error)
	NormalizeSigningKey(ctx context.Context, namespace, inputKey string, keyNormalizationMode int) (signingKey string, err error)
	FindIdentityForVerifier(ctx context.Context, iTypes []core.IdentityType, namespace string, verifier *core.VerifierRef) (identity *core.Identity, err error)
	ResolveIdentitySigner(ctx context.Context, identity *core.Identity) (parentSigner *core.SignerRef, err error)
	CachedIdentityLookupByID(ctx context.Context, id *fftypes.UUID) (identity *core.Identity, err error)
	CachedIdentityLookupMustExist(ctx context.Context, namespace, did string) (identity *core.Identity, retryable bool, err error)
	CachedIdentityLookupNilOK(ctx context.Context, namespace, did string) (identity *core.Identity, retryable bool, err error)
	CachedVerifierLookup(ctx context.Context, vType core.VerifierType, namespace, value string) (verifier *core.Verifier, err error)
	GetNodeOwnerBlockchainKey(ctx context.Context, namespace string) (*core.VerifierRef, error)
	GetNodeOwnerOrg(ctx context.Context, namespace string) (*core.Identity, error)
	VerifyIdentityChain(ctx context.Context, identity *core.Identity) (immediateParent *core.Identity, retryable bool, err error)
}

type identityManager struct {
	database   database.Plugin
	plugin     map[string]identity.Plugin
	blockchain blockchain.Plugin
	data       data.Manager
	namespace  namespace.Manager

	nodeOwnerBlockchainKey map[string]*core.VerifierRef
	nodeOwningOrgIdentity  map[string]*core.Identity
	identityCacheTTL       time.Duration
	identityCache          *ccache.Cache
	signingKeyCacheTTL     time.Duration
	signingKeyCache        *ccache.Cache
}

func NewIdentityManager(ctx context.Context, di database.Plugin, ii map[string]identity.Plugin, bi blockchain.Plugin, dm data.Manager, nm namespace.Manager) (Manager, error) {
	if di == nil || ii == nil || bi == nil || nm == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "IdentityManager")
	}
	im := &identityManager{
		database:               di,
		plugin:                 ii,
		blockchain:             bi,
		data:                   dm,
		namespace:              nm,
		nodeOwnerBlockchainKey: make(map[string]*core.VerifierRef),
		nodeOwningOrgIdentity:  make(map[string]*core.Identity),
		identityCacheTTL:       config.GetDuration(coreconfig.IdentityManagerCacheTTL),
		signingKeyCacheTTL:     config.GetDuration(coreconfig.IdentityManagerCacheTTL),
	}
	// For the identity and signingkey caches, we just treat them all equally sized and the max items
	im.identityCache = ccache.New(
		ccache.Configure().MaxSize(config.GetInt64(coreconfig.IdentityManagerCacheLimit)),
	)
	im.signingKeyCache = ccache.New(
		ccache.Configure().MaxSize(config.GetInt64(coreconfig.IdentityManagerCacheLimit)),
	)

	return im, nil
}

func ParseKeyNormalizationConfig(strConfigVal string) int {
	switch strings.ToLower(strConfigVal) {
	case "blockchain_plugin":
		return KeyNormalizationBlockchainPlugin
	default:
		return KeyNormalizationNone
	}
}

// NormalizeSigningKey takes in only a "key" (which may be empty to use the default) to be normalized and returned.
// This is for cases where keys are used directly without an "author" field alongside them (custom contracts, tokens),
// or when the author is known by the caller and should not / cannot be confirmed prior to sending (identity claims)
func (im *identityManager) NormalizeSigningKey(ctx context.Context, namespace, inputKey string, keyNormalizationMode int) (signingKey string, err error) {
	if inputKey == "" {
		signerRef := &core.SignerRef{}
		err = im.ResolveNodeOwnerSigningIdentity(ctx, namespace, signerRef)
		if err != nil {
			return "", err
		}
		return signerRef.Key, nil
	}
	// If the caller is not confident that the blockchain plugin/connector should be used to resolve,
	// for example it might be a different blockchain (Eth vs Fabric etc.), or it has its own
	// verification/management of keys, it should set `assets.keyNormalization: "none"` in the config.
	if keyNormalizationMode != KeyNormalizationBlockchainPlugin {
		return inputKey, nil
	}
	signer, err := im.normalizeKeyViaBlockchainPlugin(ctx, inputKey)
	if err != nil {
		return "", err
	}
	return signer.Value, nil
}

// ResolveInputIdentity takes in blockchain signing input information from an API call (which may
// include author or key or both), and updates it with fully resolved and normalized values
func (im *identityManager) ResolveInputSigningIdentity(ctx context.Context, namespace string, signerRef *core.SignerRef) (err error) {
	log.L(ctx).Debugf("Resolving identity input: key='%s' author='%s'", signerRef.Key, signerRef.Author)

	var verifier *core.VerifierRef
	switch {
	case signerRef.Author == "" && signerRef.Key == "":
		// Nothing specified: use the default node identity
		err = im.ResolveNodeOwnerSigningIdentity(ctx, namespace, signerRef)
		if err != nil {
			return err
		}

	case signerRef.Key != "":
		// Key specified: normalize it, then check it against author (if specified)
		if verifier, err = im.normalizeKeyViaBlockchainPlugin(ctx, signerRef.Key); err != nil {
			return err
		}
		signerRef.Key = verifier.Value

		identity, err := im.FindIdentityForVerifier(ctx, []core.IdentityType{
			core.IdentityTypeOrg,
			core.IdentityTypeCustom,
		}, namespace, verifier)
		switch {
		case err != nil:
			return err
		case identity != nil:
			// Key matches a registered verifier: author must be unspecified OR must match verifier identity
			if signerRef.Author == identity.Name || signerRef.Author == "" {
				// Resolve author to DID (if blank or bare name)
				signerRef.Author = identity.DID
			}
			if signerRef.Author != identity.DID {
				return i18n.NewError(ctx, coremsgs.MsgAuthorRegistrationMismatch, verifier.Value, signerRef.Author, identity.DID)
			}
		case signerRef.Author != "":
			// Key is unrecognized, but an author was specified: use the key and resolve author to DID
			identity, _, err := im.CachedIdentityLookupMustExist(ctx, namespace, signerRef.Author)
			if err != nil {
				return err
			}
			signerRef.Author = identity.DID
		default:
			return i18n.NewError(ctx, coremsgs.MsgAuthorMissingForKey, signerRef.Key)
		}

	case signerRef.Author != "":
		// Author specified (without key): use the first blockchain key associated with it
		identity, _, err := im.CachedIdentityLookupMustExist(ctx, namespace, signerRef.Author)
		if err != nil {
			return err
		}
		verifier, _, err = im.firstVerifierForIdentity(ctx, im.blockchain.VerifierType(), identity)
		if err != nil {
			return err
		}
		signerRef.Author = identity.DID
		signerRef.Key = verifier.Value
	}

	log.L(ctx).Debugf("Resolved identity: key='%s' author='%s'", signerRef.Key, signerRef.Author)
	return nil
}

// firstVerifierForIdentity does a lookup of the first verifier of a given type (such as a blockchain signing key) registered to an identity,
// as a convenience to allow you to only specify the org name/DID when sending a message
func (im *identityManager) firstVerifierForIdentity(ctx context.Context, vType core.VerifierType, identity *core.Identity) (verifier *core.VerifierRef, retryable bool, err error) {
	fb := database.VerifierQueryFactory.NewFilterLimit(ctx, 1)
	filter := fb.And(
		fb.Eq("type", vType),
		fb.Eq("identity", identity.ID),
	)
	verifiers, _, err := im.database.GetVerifiers(ctx, filter)
	if err != nil {
		return nil, true /* DB Error */, err
	}
	if len(verifiers) == 0 {
		return nil, false, i18n.NewError(ctx, coremsgs.MsgNoVerifierForIdentity, vType, identity.DID)
	}
	return &verifiers[0].VerifierRef, false, nil
}

// ResolveNodeOwnerSigningIdentity adds the node owner identity into a message
func (im *identityManager) ResolveNodeOwnerSigningIdentity(ctx context.Context, namespace string, signerRef *core.SignerRef) (err error) {
	verifierRef, err := im.GetNodeOwnerBlockchainKey(ctx, namespace)
	if err != nil {
		return err
	}
	identity, err := im.GetNodeOwnerOrg(ctx, namespace)
	if err != nil {
		return err
	}
	signerRef.Author = identity.DID
	signerRef.Key = verifierRef.Value
	return nil
}

// GetNodeOwnerBlockchainKey gets the blockchain key of the node owner, from the configuration
func (im *identityManager) GetNodeOwnerBlockchainKey(ctx context.Context, namespace string) (*core.VerifierRef, error) {
	if key, ok := im.nodeOwnerBlockchainKey[namespace]; ok {
		return key, nil
	}

	orgKey := im.namespace.GetConfigWithFallback(namespace, coreconfig.OrgKey)
	if orgKey == "" {
		orgKey = config.GetString(coreconfig.OrgIdentityDeprecated)
		if orgKey != "" {
			log.L(ctx).Warnf("The %s config key has been deprecated. Please use %s instead", coreconfig.OrgIdentityDeprecated, coreconfig.OrgKey)
		}
	}
	if orgKey == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgNodeMissingBlockchainKey)
	}

	verifier, err := im.normalizeKeyViaBlockchainPlugin(ctx, orgKey)
	if err != nil {
		return nil, err
	}
	im.nodeOwnerBlockchainKey[namespace] = verifier
	return verifier, nil
}

// normalizeKeyViaBlockchainPlugin does a cached lookup of the fully qualified key, associated with a key reference string
func (im *identityManager) normalizeKeyViaBlockchainPlugin(ctx context.Context, inputKey string) (verifier *core.VerifierRef, err error) {
	if inputKey == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgBlockchainKeyNotSet)
	}
	if cached := im.signingKeyCache.Get(inputKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		return cached.Value().(*core.VerifierRef), nil
	}
	keyString, err := im.blockchain.NormalizeSigningKey(ctx, inputKey)
	if err != nil {
		return nil, err
	}
	verifier = &core.VerifierRef{
		Type:  im.blockchain.VerifierType(),
		Value: keyString,
	}
	im.signingKeyCache.Set(inputKey, verifier, im.identityCacheTTL)
	return verifier, nil
}

// FindIdentityForVerifier is a reverse lookup function to look up an identity registered as owner of the specified verifier
func (im *identityManager) FindIdentityForVerifier(ctx context.Context, iTypes []core.IdentityType, namespace string, verifier *core.VerifierRef) (identity *core.Identity, err error) {
	identity, err = im.cachedIdentityLookupByVerifierRef(ctx, namespace, verifier)
	if err != nil || identity != nil {
		return identity, err
	}
	return nil, nil
}

// GetNodeOwnerOrg returns the identity of the organization that owns the node, if fully registered within the given namespace
func (im *identityManager) GetNodeOwnerOrg(ctx context.Context, namespace string) (*core.Identity, error) {
	if id, ok := im.nodeOwningOrgIdentity[namespace]; ok {
		return id, nil
	}
	verifierRef, err := im.GetNodeOwnerBlockchainKey(ctx, namespace)
	if err != nil {
		return nil, err
	}
	orgName := config.GetString(coreconfig.OrgName)
	identity, err := im.cachedIdentityLookupByVerifierRef(ctx, namespace, verifierRef)
	if err != nil || identity == nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgLocalOrgLookupFailed, orgName, verifierRef.Value)
	}
	// Confirm that the specified blockchain key is associated with the correct org
	if identity.Type != core.IdentityTypeOrg || identity.Name != orgName {
		return nil, i18n.NewError(ctx, coremsgs.MsgLocalOrgLookupFailed, orgName, verifierRef.Value)
	}
	im.nodeOwningOrgIdentity[namespace] = identity
	return identity, nil
}

func (im *identityManager) VerifyIdentityChain(ctx context.Context, checkIdentity *core.Identity) (immediateParent *core.Identity, retryable bool, err error) {

	err = checkIdentity.Validate(ctx)
	if err != nil {
		return nil, false, err
	}

	loopDetect := make(map[fftypes.UUID]bool)
	current := checkIdentity
	for {
		loopDetect[*current.ID] = true
		parentID := current.Parent
		if parentID == nil {
			return immediateParent, false, nil
		}
		if _, ok := loopDetect[*parentID]; ok {
			return nil, false, i18n.NewError(ctx, coremsgs.MsgIdentityChainLoop, parentID, current.DID, current.ID)
		}
		parent, err := im.CachedIdentityLookupByID(ctx, parentID)
		if err != nil {
			return nil, true /* DB Error */, err
		}
		if parent == nil {
			return nil, false, i18n.NewError(ctx, coremsgs.MsgParentIdentityNotFound, parentID, current.DID, current.ID)
		}
		if err := im.validateParentType(ctx, current, parent); err != nil {
			return nil, false, err
		}
		if parent.Messages.Claim == nil {
			return nil, false, i18n.NewError(ctx, coremsgs.MsgParentIdentityMissingClaim, parent.DID, parent.ID)
		}
		current = parent
		if immediateParent == nil {
			immediateParent = parent
		}
	}

}

func (im *identityManager) ResolveIdentitySigner(ctx context.Context, identity *core.Identity) (parentSigner *core.SignerRef, err error) {
	// Find the message that registered the identity
	msg, err := im.database.GetMessageByID(ctx, identity.Messages.Claim)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgParentIdentityMissingClaim, identity.DID, identity.ID)
	}
	// Return the signing identity from that claim
	return &msg.Header.SignerRef, nil
}

func (im *identityManager) validateParentType(ctx context.Context, child *core.Identity, parent *core.Identity) error {

	switch child.Type {
	case core.IdentityTypeNode, core.IdentityTypeOrg:
		if parent.Type != core.IdentityTypeOrg {
			return i18n.NewError(ctx, coremsgs.MsgInvalidIdentityParentType, parent.DID, parent.ID, parent.Type, child.DID, child.ID, child.Type)
		}
		return nil
	case core.IdentityTypeCustom:
		if parent.Type != core.IdentityTypeOrg && parent.Type != core.IdentityTypeCustom {
			return i18n.NewError(ctx, coremsgs.MsgInvalidIdentityParentType, parent.DID, parent.ID, parent.Type, child.DID, child.ID, child.Type)
		}
		return nil
	default:
		return i18n.NewError(ctx, i18n.MsgUnknownIdentityType, child.Type)
	}

}

func (im *identityManager) cachedIdentityLookupByVerifierRef(ctx context.Context, namespace string, verifierRef *core.VerifierRef) (*core.Identity, error) {
	cacheKey := fmt.Sprintf("key=%s|%s|%s", namespace, verifierRef.Type, verifierRef.Value)
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		return cached.Value().(*core.Identity), nil
	}
	verifier, err := im.database.GetVerifierByValue(ctx, verifierRef.Type, namespace, verifierRef.Value)
	if err != nil {
		return nil, err
	} else if verifier == nil {
		if namespace != core.LegacySystemNamespace && config.GetBool(coreconfig.IdentityManagerLegacySystemIdentities) {
			// Fall back to SystemNamespace
			return im.cachedIdentityLookupByVerifierRef(ctx, core.LegacySystemNamespace, verifierRef)
		}
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

func (im *identityManager) CachedIdentityLookupNilOK(ctx context.Context, namespace, didLookupStr string) (identity *core.Identity, retryable bool, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	cacheKey := fmt.Sprintf("did=%s", didLookupStr)
	defer func() {
		didResolved := ""
		var uuidResolved *fftypes.UUID
		if identity != nil {
			didResolved = identity.DID
			uuidResolved = identity.ID
		}
		log.L(ctx).Debugf("Resolved DID '%s' to identity: %s / %s (err=%v)", didLookupStr, uuidResolved, didResolved, err)
	}()
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		identity = cached.Value().(*core.Identity)
	} else {
		if strings.HasPrefix(didLookupStr, core.DIDPrefix) {
			if !strings.HasPrefix(didLookupStr, core.FireFlyDIDPrefix) {
				return nil, false, i18n.NewError(ctx, coremsgs.MsgDIDResolverUnknown, didLookupStr)
			}
			// Look up by the full DID
			if identity, err = im.database.GetIdentityByDID(ctx, didLookupStr); err != nil {
				return nil, true /* DB Error */, err
			}
			if identity == nil && strings.HasPrefix(didLookupStr, core.FireFlyOrgDIDPrefix) {
				// We allow the UUID to be used to resolve DIDs as an alias to the name
				uuid, err := fftypes.ParseUUID(ctx, strings.TrimPrefix(didLookupStr, core.FireFlyOrgDIDPrefix))
				if err == nil {
					if identity, err = im.database.GetIdentityByID(ctx, uuid); err != nil {
						return nil, true /* DB Error */, err
					}
				}
			}
		} else {
			// If there is just a name in there, then it could be an Org type identity (from the very original usage of the field)
			if identity, err = im.database.GetIdentityByName(ctx, core.IdentityTypeOrg, namespace, didLookupStr); err != nil {
				return nil, true /* DB Error */, err
			}
		}

		if identity != nil {
			// Cache the result
			im.identityCache.Set(cacheKey, identity, im.identityCacheTTL)
		}
	}
	return identity, false, nil
}

func (im *identityManager) CachedIdentityLookupMustExist(ctx context.Context, namespace, didLookupStr string) (identity *core.Identity, retryable bool, err error) {
	identity, retryable, err = im.CachedIdentityLookupNilOK(ctx, namespace, didLookupStr)
	if err != nil {
		return nil, retryable, err
	}
	if identity == nil {
		return nil, false, i18n.NewError(ctx, coremsgs.MsgIdentityNotFoundByString, didLookupStr)
	}
	return identity, false, nil
}

func (im *identityManager) CachedIdentityLookupByID(ctx context.Context, id *fftypes.UUID) (identity *core.Identity, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	cacheKey := fmt.Sprintf("id=%s", id)
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		identity = cached.Value().(*core.Identity)
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

func (im *identityManager) CachedVerifierLookup(ctx context.Context, vType core.VerifierType, namespace, value string) (verifier *core.Verifier, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	cacheKey := fmt.Sprintf("v=%s|%s|%s", vType, namespace, value)
	if cached := im.identityCache.Get(cacheKey); cached != nil {
		cached.Extend(im.identityCacheTTL)
		verifier = cached.Value().(*core.Verifier)
	} else {
		verifier, err = im.database.GetVerifierByValue(ctx, vType, namespace, value)
		if err != nil || verifier == nil {
			return verifier, err
		}
		// Cache the result
		im.identityCache.Set(cacheKey, verifier, im.identityCacheTTL)
	}
	return verifier, nil
}
