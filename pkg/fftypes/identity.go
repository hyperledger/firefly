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

package fftypes

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
)

// IdentityType is the type of an identity
type IdentityType = FFEnum

var (
	// IdentityTypeOrg is an organization
	IdentityTypeOrg IdentityType = ffEnum("identitytype", "org")
	// IdentityTypeNode is a node
	IdentityTypeNode IdentityType = ffEnum("identitytype", "node")
	// IdentityTypeCustom is a user defined identity within a namespace
	IdentityTypeCustom IdentityType = ffEnum("identitytype", "custom")
)

const (
	DIDPrefix              = "did:"
	FireFlyDIDPrefix       = "did:firefly:"
	FireFlyOrgDIDPrefix    = "did:firefly:org/"
	FireFlyNodeDIDPrefix   = "did:firefly:node/"
	FireFlyCustomDIDPrefix = "did:firefly:ns/"
)

type IdentityMessages struct {
	Claim        *UUID `json:"claim"`
	Verification *UUID `json:"verification"`
	Update       *UUID `json:"update"`
}

// IdentityBase are the immutable fields of an identity that determine what the identity itself is
type IdentityBase struct {
	ID        *UUID        `json:"id"`
	DID       string       `json:"did"`
	Type      IdentityType `json:"type" ffenum:"identitytype"`
	Parent    *UUID        `json:"parent,omitempty"`
	Namespace string       `json:"namespace"`
	Name      string       `json:"name,omitempty"`
}

// IdentityProfile are the field of a profile that can be updated over time
type IdentityProfile struct {
	Description string     `json:"description,omitempty"`
	Profile     JSONObject `json:"profile,omitempty"`
}

// Identity is the persisted structure backing all identities, including orgs, nodes and custom identities
type Identity struct {
	IdentityBase
	IdentityProfile
	Messages IdentityMessages `json:"messages,omitempty"`
	Created  *FFTime          `json:"created,omitempty"`
}

// IdentityRef is the nested structure representing an identity, that might comprise a resolvable
// by FireFly identity DID, a blockchain signing key, or both.
type IdentityRef struct {
	Author string `json:"author,omitempty"`
	Key    string `json:"key,omitempty"`
}

// IdentityClaim is the data payload used in a message to broadcast an intent to publish a new identity.
// Most claims (except root orgs, where different requirements apply) require a separate IdentityValidation
// from the parent identity to be published (on the same topic) before the identity is considered valid
// and is stored as a confirmed identity.
type IdentityClaim struct {
	Identity *Identity `json:"identity"`
}

// IdentityVerification is the data payload used in message to broadcast a verification of a child identity.
// Must refer to the UUID and Hash of the IdentityClaim message, and must contain the same base identity data.
type IdentityVerification struct {
	Claim    MessageRef   `json:"claim"`
	Identity IdentityBase `json:"identity"`

	// IdentityRef lets us store back the message when broadcasting, but isn't part of the payload
	IdentityRef *Identity `json:"-"`
}

// IdentityProfileUpdate is the data payload used in message to broadcast an update to an identity profile.
// The broadcast must be on the same identity as the currently established identity claim message for the identity,
// and it must contain the same identity data.
// The profile is replaced in its entirety.
type IdentityProfileUpdate struct {
	Identity IdentityBase    `json:"identity"`
	Profile  IdentityProfile `json:"profile,omitempty"`

	// IdentityRef lets us store back the message when broadcasting, but isn't part of the payload
	IdentityRef *Identity `json:"-"`
}

func (ic *IdentityClaim) Topic() string {
	return ic.Identity.Topic()
}

func (ic *IdentityClaim) SetBroadcastMessage(msgID *UUID) {
	ic.Identity.Messages.Claim = msgID
}

func (iv *IdentityVerification) Topic() string {
	return iv.Identity.Topic()
}

func (iv *IdentityVerification) SetBroadcastMessage(msgID *UUID) {
	if iv.IdentityRef != nil {
		iv.IdentityRef.Messages.Verification = msgID
	}
}

func (iu *IdentityProfileUpdate) Topic() string {
	return iu.Identity.Topic()
}

func (iu *IdentityProfileUpdate) SetBroadcastMessage(msgID *UUID) {
	if iu.IdentityRef != nil {
		iu.IdentityRef.Messages.Update = msgID
	}
}

func (i *IdentityBase) Topic() string {
	h := sha256.New()
	h.Write([]byte(i.DID))
	return HashResult(h).String()
}

func (i *IdentityBase) Validate(ctx context.Context) (err error) {
	if i.ID == nil {
		return i18n.NewError(ctx, i18n.MsgNilID)
	}
	if err = ValidateFFNameFieldNoUUID(ctx, i.Namespace, "namespace"); err != nil {
		return err
	}
	if err = ValidateFFNameFieldNoUUID(ctx, i.Name, "name"); err != nil {
		return err
	}
	if requiredDID, err := i.GenerateDID(ctx); err != nil {
		return err
	} else if i.DID != requiredDID {
		return i18n.NewError(ctx, i18n.MsgInvalidDIDForType, i.DID, i.Type, i.Namespace, i.Name)
	}
	return nil
}

func (i *IdentityBase) GenerateDID(ctx context.Context) (string, error) {
	switch i.Type {
	case IdentityTypeCustom:
		if i.Namespace == SystemNamespace {
			return "", i18n.NewError(ctx, i18n.MsgCustomIdentitySystemNS, SystemNamespace)
		}
		if i.Parent == nil {
			return "", i18n.NewError(ctx, i18n.MsgNilParentIdentity, i.Type)
		}
		return fmt.Sprintf("%s%s/%s", FireFlyCustomDIDPrefix, i.Namespace, i.Name), nil
	case IdentityTypeNode:
		if i.Namespace != SystemNamespace {
			return "", i18n.NewError(ctx, i18n.MsgSystemIdentityCustomNS, SystemNamespace)
		}
		if i.Parent == nil {
			return "", i18n.NewError(ctx, i18n.MsgNilParentIdentity, i.Type)
		}
		return fmt.Sprintf("%s%s", FireFlyNodeDIDPrefix, i.Name), nil
	case IdentityTypeOrg:
		if i.Namespace != SystemNamespace {
			return "", i18n.NewError(ctx, i18n.MsgSystemIdentityCustomNS, SystemNamespace)
		}
		return fmt.Sprintf("%s%s", FireFlyOrgDIDPrefix, i.Name), nil
	default:
		return "", i18n.NewError(ctx, i18n.MsgUnknownIdentityType, i.Type)
	}
}

func (i *IdentityBase) Equals(ctx context.Context, i2 *IdentityBase) bool {
	if err := i.Validate(ctx); err != nil {
		log.L(ctx).Warnf("Comparing invalid identity (source) %s (%v): %s", i.DID, i.ID, err)
		return false
	}
	if err := i2.Validate(ctx); err != nil {
		log.L(ctx).Warnf("Comparing invalid identity (target) %s (%v): %s", i.DID, i2.ID, err)
		return false
	}
	return i.ID.Equals(i2.ID) &&
		i.DID == i2.DID &&
		i.Type == i2.Type &&
		i.Parent.Equals(i2.Parent) &&
		i.Namespace == i2.Namespace &&
		i.Name == i2.Name
}

func (identity *Identity) Validate(ctx context.Context) (err error) {
	if err = identity.IdentityBase.Validate(ctx); err != nil {
		return err
	}
	if err = ValidateLength(ctx, identity.Description, "description", 4096); err != nil {
		return err
	}
	return nil
}
