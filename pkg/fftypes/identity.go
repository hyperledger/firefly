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

	"github.com/hyperledger/firefly/internal/i18n"
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

// Identity is the persisted structure backing all identities, including orgs, nodes and custom identities
type Identity struct {
	ID          *UUID        `json:"id"`
	DID         string       `json:"did"`
	Type        IdentityType `json:"type" ffenum:"identitytype"`
	Parent      *UUID        `json:"parent,omitempty"`
	Namespace   string       `json:"namespace"`
	Name        string       `json:"name,omitempty"`
	Description string       `json:"description,omitempty"`
	Profile     JSONObject   `json:"profile,omitempty"`
	Message     *UUID        `json:"message,omitempty"`
	Created     *FFTime      `json:"created,omitempty"`
}

// IdentityRef is the nested structure representing an identity, that might comprise a resolvable
// by FireFly identity DID, a blockchain signing key, or both.
type IdentityRef struct {
	Author string `json:"author,omitempty"`
	Key    string `json:"key,omitempty"`
}

const (
	DIDPrefix            = "did:"
	FireFlyDIDPrefix     = "did:firefly:"
	FireFlyOrgDIDPrefix  = "did:firefly:org/"
	FireFlyNodeDIDPrefix = "did:firefly:node/"
	OrgTopic             = "ff_organizations"
)

func (identity *Identity) Validate(ctx context.Context, existing bool) (err error) {
	if err = ValidateFFNameFieldNoUUID(ctx, identity.Name, "name"); err != nil {
		return err
	}
	if err = ValidateLength(ctx, identity.Description, "description", 4096); err != nil {
		return err
	}
	if existing {
		if identity.ID == nil {
			return i18n.NewError(ctx, i18n.MsgNilID)
		}
	}
	return nil
}

func (identity *Identity) Topic() string {
	// Topic is the hash of the DID to assure all nodes process first-come-first-serve consistently
	var b Bytes32 = sha256.Sum256([]byte(identity.DID))
	return b.String()
}

func (identity *Identity) SetBroadcastMessage(msgID *UUID) {
	identity.Message = msgID
}
