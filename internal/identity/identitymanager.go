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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/multiparty"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

const (
	KeyNormalizationBlockchainPlugin = iota
	KeyNormalizationNone
)

type Manager interface {
	ResolveInputSigningIdentity(ctx context.Context, signerRef *core.SignerRef) (err error)
	ResolveInputSigningKey(ctx context.Context, inputKey *core.VerifierRef) (*core.VerifierRef, error)
	NormalizeSigningKey(ctx context.Context, inputKey string, keyNormalizationMode int) (signingKey string, err error)
	FindIdentityForVerifier(ctx context.Context, iTypes []core.IdentityType, verifier *core.VerifierRef) (identity *core.Identity, err error)
	ResolveIdentitySigner(ctx context.Context, identity *core.Identity) (parentSigner *core.SignerRef, err error)
	CachedIdentityLookupByID(ctx context.Context, id *fftypes.UUID) (identity *core.Identity, err error)
	CachedIdentityLookupMustExist(ctx context.Context, did string) (identity *core.Identity, retryable bool, err error)
	CachedIdentityLookupNilOK(ctx context.Context, did string) (identity *core.Identity, retryable bool, err error)
	GetMultipartyRootVerifier(ctx context.Context) (*core.VerifierRef, error)
	GetMultipartyRootOrg(ctx context.Context) (*core.Identity, error)
	GetLocalNode(ctx context.Context) (node *core.Identity, err error)
	VerifyIdentityChain(ctx context.Context, identity *core.Identity) (immediateParent *core.Identity, retryable bool, err error)
	ValidateNodeOwner(ctx context.Context, node *core.Identity, identity *core.Identity) (valid bool, err error)
}

type identityManager struct {
	database               database.Plugin
	blockchain             blockchain.Plugin  // optional
	multiparty             multiparty.Manager // optional
	namespace              string
	defaultKey             string
	multipartyRootVerifier *core.VerifierRef
	identityCache          cache.CInterface
	signingKeyCache        cache.CInterface
}

func NewIdentityManager(ctx context.Context, ns, defaultKey string, di database.Plugin, bi blockchain.Plugin, mp multiparty.Manager, cacheManager cache.Manager) (Manager, error) {
	if di == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "IdentityManager")
	}
	im := &identityManager{
		database:   di,
		blockchain: bi,
		namespace:  ns,
		multiparty: mp,
		defaultKey: defaultKey,
	}

	identityCache, err := cacheManager.GetCache(
		cache.NewCacheConfig(
			ctx,
			coreconfig.CacheIdentityLimit,
			coreconfig.CacheIdentityTTL,
			ns,
		),
	)
	if err != nil {
		return nil, err
	}
	im.identityCache = identityCache

	signingKeyCache, err := cacheManager.GetCache(
		cache.NewCacheConfig(
			ctx,
			coreconfig.CacheSigningKeyLimit,
			coreconfig.CacheSigningKeyTTL,
			ns,
		),
	)
	if err != nil {
		return nil, err
	}

	// For the identity and signingkey caches, we just treat them all equally sized and the max items
	im.signingKeyCache = signingKeyCache

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

func (im *identityManager) GetLocalNode(ctx context.Context) (node *core.Identity, err error) {
	nodeName := im.multiparty.LocalNode().Name
	nodeDID := fmt.Sprintf("%s%s", core.FireFlyNodeDIDPrefix, nodeName)
	node, _, err = im.CachedIdentityLookupNilOK(ctx, nodeDID)
	return node, err
}

// NormalizeSigningKey takes in only a "key" (which may be empty to use the default) to be normalized and returned.
// This is for cases where keys are used directly without an "author" field alongside them (custom contracts, tokens),
// or when the author is known by the caller and should not / cannot be confirmed prior to sending (identity claims)
func (im *identityManager) NormalizeSigningKey(ctx context.Context, inputKey string, keyNormalizationMode int) (signingKey string, err error) {
	if inputKey == "" {
		if im.blockchain == nil {
			if im.defaultKey == "" {
				return "", i18n.NewError(ctx, coremsgs.MsgNodeMissingBlockchainKey)
			}

			return im.defaultKey, nil
		}

		verifierRef, err := im.getDefaultVerifier(ctx)
		if err != nil {
			return "", err
		}
		return verifierRef.Value, nil
	}
	// If the caller is not confident that the blockchain plugin/connector should be used to resolve,
	// for example it might be a different blockchain (Eth vs Fabric etc.), or it has its own
	// verification/management of keys, it should set `namespaces.predefined[].asset.manager.keyNormalization: "none"` in the config.
	if keyNormalizationMode != KeyNormalizationBlockchainPlugin {
		return inputKey, nil
	}
	signer, err := im.normalizeKeyViaBlockchainPlugin(ctx, inputKey)
	if err != nil {
		return "", err
	}
	return signer.Value, nil
}

func (im *identityManager) ResolveInputSigningKey(ctx context.Context, inputKey *core.VerifierRef) (*core.VerifierRef, error) {
	log.L(ctx).Debugf("Resolving input signing key: type='%s' value='%s'", inputKey.Type, inputKey.Value)

	if im.blockchain == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgBlockchainNotConfigured)
	}

	verifierType := inputKey.Type

	if verifierType.String() == "" {
		verifierType = im.blockchain.VerifierType()
	}

	if verifierType != im.blockchain.VerifierType() {
		return nil, i18n.NewError(ctx, coremsgs.MsgUnknownVerifierType)
	}

	signingKey, err := im.blockchain.NormalizeSigningKey(ctx, inputKey.Value)
	if err != nil {
		return nil, err
	}

	return &core.VerifierRef{
		Type:  verifierType,
		Value: signingKey,
	}, nil
}

// ResolveInputIdentity takes in blockchain signing input information from an API call (which may
// include author or key or both), and updates it with fully resolved and normalized values
func (im *identityManager) ResolveInputSigningIdentity(ctx context.Context, signerRef *core.SignerRef) (err error) {
	log.L(ctx).Debugf("Resolving identity input: key='%s' author='%s'", signerRef.Key, signerRef.Author)

	if im.blockchain == nil {
		return i18n.NewError(ctx, coremsgs.MsgBlockchainNotConfigured)
	}

	var verifier *core.VerifierRef
	switch {
	case signerRef.Author == "" && signerRef.Key == "":
		// Nothing specified: use the default node identity
		err = im.resolveDefaultSigningIdentity(ctx, signerRef)
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
		}, verifier)
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
			identity, _, err := im.CachedIdentityLookupMustExist(ctx, signerRef.Author)
			if err != nil {
				return err
			}
			signerRef.Author = identity.DID
		default:
			return i18n.NewError(ctx, coremsgs.MsgAuthorMissingForKey, signerRef.Key)
		}

	case signerRef.Author != "":
		// Author specified (without key): use the first blockchain key associated with it
		identity, _, err := im.CachedIdentityLookupMustExist(ctx, signerRef.Author)
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
	verifiers, _, err := im.database.GetVerifiers(ctx, identity.Namespace, filter)
	if err != nil {
		return nil, true /* DB Error */, err
	}
	if len(verifiers) == 0 {
		return nil, false, i18n.NewError(ctx, coremsgs.MsgNoVerifierForIdentity, vType, identity.DID)
	}
	return &verifiers[0].VerifierRef, false, nil
}

// resolveDefaultSigningIdentity adds the default signing identity into a message
func (im *identityManager) resolveDefaultSigningIdentity(ctx context.Context, signerRef *core.SignerRef) (err error) {
	verifierRef, err := im.getDefaultVerifier(ctx)
	if err != nil {
		return err
	}
	identity, err := im.GetMultipartyRootOrg(ctx)
	if err != nil {
		return err
	}
	signerRef.Author = identity.DID
	signerRef.Key = verifierRef.Value
	return nil
}

// getDefaultVerifier gets the default blockchain verifier via the configuration
func (im *identityManager) getDefaultVerifier(ctx context.Context) (verifier *core.VerifierRef, err error) {
	if im.defaultKey != "" {
		return im.normalizeKeyViaBlockchainPlugin(ctx, im.defaultKey)
	}
	if im.multiparty != nil {
		return im.GetMultipartyRootVerifier(ctx)
	}
	return nil, i18n.NewError(ctx, coremsgs.MsgNodeMissingBlockchainKey)
}

// GetMultipartyRootVerifier gets the blockchain verifier of the root org via the configuration
func (im *identityManager) GetMultipartyRootVerifier(ctx context.Context) (*core.VerifierRef, error) {
	if im.multipartyRootVerifier != nil {
		return im.multipartyRootVerifier, nil
	}

	orgKey := im.multiparty.RootOrg().Key
	if orgKey == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgNodeMissingBlockchainKey)
	}

	verifier, err := im.normalizeKeyViaBlockchainPlugin(ctx, orgKey)
	if err != nil {
		return nil, err
	}
	im.multipartyRootVerifier = verifier
	return verifier, nil
}

// normalizeKeyViaBlockchainPlugin does a cached lookup of the fully qualified key, associated with a key reference string
func (im *identityManager) normalizeKeyViaBlockchainPlugin(ctx context.Context, inputKey string) (verifier *core.VerifierRef, err error) {
	if inputKey == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgBlockchainKeyNotSet)
	}

	if im.blockchain == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgBlockchainNotConfigured)
	}

	if cachedValue := im.signingKeyCache.Get(inputKey); cachedValue != nil {
		return cachedValue.(*core.VerifierRef), nil
	}
	keyString, err := im.blockchain.NormalizeSigningKey(ctx, inputKey)
	if err != nil {
		return nil, err
	}
	verifier = &core.VerifierRef{
		Type:  im.blockchain.VerifierType(),
		Value: keyString,
	}
	im.signingKeyCache.Set(inputKey, verifier)
	return verifier, nil
}

// FindIdentityForVerifier is a reverse lookup function to look up an identity registered as owner of the specified verifier
func (im *identityManager) FindIdentityForVerifier(ctx context.Context, iTypes []core.IdentityType, verifier *core.VerifierRef) (identity *core.Identity, err error) {
	identity, err = im.cachedIdentityLookupByVerifierRef(ctx, im.namespace, verifier)
	if err != nil || identity != nil {
		return identity, err
	}
	return nil, nil
}

// GetMultipartyRootOrg returns the identity of the organization that owns the node, if fully registered within the given namespace
func (im *identityManager) GetMultipartyRootOrg(ctx context.Context) (*core.Identity, error) {
	verifierRef, err := im.GetMultipartyRootVerifier(ctx)
	if err != nil {
		return nil, err
	}

	orgName := im.multiparty.RootOrg().Name
	identity, err := im.cachedIdentityLookupByVerifierRef(ctx, im.namespace, verifierRef)
	if err != nil || identity == nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgLocalOrgLookupFailed, orgName, verifierRef.Value)
	}
	// Confirm that the specified blockchain key is associated with the correct org
	if identity.Type != core.IdentityTypeOrg || identity.Name != orgName {
		return nil, i18n.NewError(ctx, coremsgs.MsgLocalOrgLookupFailed, orgName, verifierRef.Value)
	}
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
		if im.multiparty != nil && parent.Messages.Claim == nil {
			return nil, false, i18n.NewError(ctx, coremsgs.MsgParentIdentityMissingClaim, parent.DID, parent.ID)
		}
		current = parent
		if immediateParent == nil {
			immediateParent = parent
		}
	}

}

func (im *identityManager) ResolveIdentitySigner(ctx context.Context, identity *core.Identity) (signer *core.SignerRef, err error) {

	// Find the message that registered the identity
	msg, err := im.database.GetMessageByID(ctx, im.namespace, identity.Messages.Claim)
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
	cacheKey := fmt.Sprintf("ns=%s,type=%s,verifier=%s", namespace, verifierRef.Type, verifierRef.Value)
	if cachedValue := im.identityCache.Get(cacheKey); cachedValue != nil {
		return cachedValue.(*core.Identity), nil
	}
	verifier, err := im.database.GetVerifierByValue(ctx, verifierRef.Type, namespace, verifierRef.Value)
	if err != nil {
		return nil, err
	} else if verifier == nil {
		if namespace != core.LegacySystemNamespace && im.multiparty != nil && im.multiparty.GetNetworkVersion() == 1 {
			// For V1 networks, fall back to LegacySystemNamespace for looking up identities
			// This assumes that the system namespace shares a database with this manager's namespace!
			return im.cachedIdentityLookupByVerifierRef(ctx, core.LegacySystemNamespace, verifierRef)
		}
		return nil, err
	}
	identity, err := im.database.GetIdentityByID(ctx, namespace, verifier.Identity)
	if err != nil {
		return nil, err
	}
	if identity == nil {
		return nil, i18n.NewError(ctx, i18n.MsgEmptyMemberIdentity, verifier.Identity)
	}
	// Cache the result
	im.identityCache.Set(cacheKey, identity)
	return identity, nil
}

func (im *identityManager) cachedIdentityLookup(ctx context.Context, namespace, didLookupStr string) (identity *core.Identity, retryable bool, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	cacheKey := fmt.Sprintf("ns=%s,did=%s", namespace, didLookupStr)
	defer func() {
		didResolved := ""
		var uuidResolved *fftypes.UUID
		if identity != nil {
			didResolved = identity.DID
			uuidResolved = identity.ID
		}
		log.L(ctx).Debugf("Resolved DID '%s' to identity: %s / %s (err=%v)", didLookupStr, uuidResolved, didResolved, err)
	}()
	if cachedValue := im.identityCache.Get(cacheKey); cachedValue != nil {
		identity = cachedValue.(*core.Identity)
	} else {
		if strings.HasPrefix(didLookupStr, core.DIDPrefix) {
			if !strings.HasPrefix(didLookupStr, core.FireFlyDIDPrefix) {
				return nil, false, i18n.NewError(ctx, coremsgs.MsgDIDResolverUnknown, didLookupStr)
			}
			// Look up by the full DID
			if identity, err = im.database.GetIdentityByDID(ctx, namespace, didLookupStr); err != nil {
				return nil, true /* DB Error */, err
			}
			if identity == nil && strings.HasPrefix(didLookupStr, core.FireFlyOrgDIDPrefix) {
				// We allow the UUID to be used to resolve DIDs as an alias to the name
				uuid, err := fftypes.ParseUUID(ctx, strings.TrimPrefix(didLookupStr, core.FireFlyOrgDIDPrefix))
				if err == nil {
					if identity, err = im.database.GetIdentityByID(ctx, namespace, uuid); err != nil {
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
			im.identityCache.Set(cacheKey, identity)
		} else if namespace != core.LegacySystemNamespace && im.multiparty != nil && im.multiparty.GetNetworkVersion() == 1 {
			// For V1 networks, fall back to LegacySystemNamespace for looking up identities
			// This assumes that the system namespace shares a database with this manager's namespace!
			return im.cachedIdentityLookup(ctx, core.LegacySystemNamespace, didLookupStr)
		}
	}
	return identity, false, nil
}

func (im *identityManager) CachedIdentityLookupNilOK(ctx context.Context, didLookupStr string) (identity *core.Identity, retryable bool, err error) {
	return im.cachedIdentityLookup(ctx, im.namespace, didLookupStr)
}

func (im *identityManager) CachedIdentityLookupMustExist(ctx context.Context, didLookupStr string) (identity *core.Identity, retryable bool, err error) {
	identity, retryable, err = im.CachedIdentityLookupNilOK(ctx, didLookupStr)
	if err != nil {
		return nil, retryable, err
	}
	if identity == nil {
		return nil, false, i18n.NewError(ctx, coremsgs.MsgIdentityNotFoundByString, didLookupStr)
	}
	return identity, false, nil
}

func (im *identityManager) cachedIdentityLookupByID(ctx context.Context, namespace string, id *fftypes.UUID) (identity *core.Identity, err error) {
	// Use an LRU cache for the author identity, as it's likely for the same identity to be re-used over and over
	cacheKey := fmt.Sprintf("ns=%s,id=%s", namespace, id)
	if cachedValue := im.identityCache.Get(cacheKey); cachedValue != nil {
		identity = cachedValue.(*core.Identity)
	} else {
		identity, err = im.database.GetIdentityByID(ctx, namespace, id)
		if err != nil {
			return nil, err
		}
		if identity == nil {
			if namespace != core.LegacySystemNamespace && im.multiparty != nil && im.multiparty.GetNetworkVersion() == 1 {
				// For V1 networks, fall back to LegacySystemNamespace for looking up identities
				// This assumes that the system namespace shares a database with this manager's namespace!
				return im.cachedIdentityLookupByID(ctx, core.LegacySystemNamespace, id)
			}
			return nil, nil
		}
		// Cache the result
		im.identityCache.Set(cacheKey, identity)
	}
	return identity, nil
}

func (im *identityManager) CachedIdentityLookupByID(ctx context.Context, id *fftypes.UUID) (identity *core.Identity, err error) {
	return im.cachedIdentityLookupByID(ctx, im.namespace, id)
}

// Validate that the given identity or one of its ancestors owns the given node.
func (im *identityManager) ValidateNodeOwner(ctx context.Context, node *core.Identity, identity *core.Identity) (valid bool, err error) {
	l := log.L(ctx)
	candidate := identity
	foundOwner := candidate.ID.Equals(node.Parent)
	for !foundOwner && candidate.Parent != nil {
		parent := candidate.Parent
		candidate, err = im.CachedIdentityLookupByID(ctx, parent)
		if err != nil {
			l.Errorf("Failed to retrieve node org '%s': %v", parent, err)
			return false, err // retry for persistence error
		}
		if candidate == nil {
			l.Errorf("Did not find '%s' in chain for identity '%s' (%s)", parent, identity.DID, identity.ID)
			return false, nil
		}
		foundOwner = candidate.ID.Equals(node.Parent)
	}
	if !foundOwner {
		l.Errorf("No identity in the chain matches owner '%s' of node '%s' ('%s')", node.Parent, node.ID, node.Name)
		return false, nil
	}
	return true, nil
}
