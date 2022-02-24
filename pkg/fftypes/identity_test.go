// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.identity/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fftypes

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testOrg() *Identity {
	return &Identity{
		IdentityBase: IdentityBase{
			ID:        NewUUID(),
			DID:       "did:firefly:org/org1",
			Type:      IdentityTypeOrg,
			Namespace: SystemNamespace,
			Name:      "org1",
		},
		IdentityProfile: IdentityProfile{
			Description: "desc",
			Profile: JSONObject{
				"some": "profiledata",
			},
		},
	}
}

func testNode() *Identity {
	return &Identity{
		IdentityBase: IdentityBase{
			ID:        NewUUID(),
			DID:       "did:firefly:node/node1",
			Parent:    NewUUID(),
			Type:      IdentityTypeNode,
			Namespace: SystemNamespace,
			Name:      "node1",
		},
		IdentityProfile: IdentityProfile{
			Description: "desc",
			Profile: JSONObject{
				"some": "profiledata",
			},
		},
	}
}

func testCustom(ns, name string) *Identity {
	return &Identity{
		IdentityBase: IdentityBase{
			ID:        NewUUID(),
			DID:       fmt.Sprintf("did:firefly:ns/%s/%s", ns, name),
			Parent:    NewUUID(),
			Type:      IdentityTypeCustom,
			Namespace: ns,
			Name:      name,
		},
		IdentityProfile: IdentityProfile{
			Description: "desc",
			Profile: JSONObject{
				"some": "profiledata",
			},
		},
	}
}

func TestIdentityValidationOrgs(t *testing.T) {

	ctx := context.Background()
	assert.Regexp(t, "FF10368", (*Identity)(nil).Validate(ctx))

	o := testOrg()
	assert.NoError(t, o.Validate(ctx))

	o = testOrg()
	o.ID = nil
	assert.Regexp(t, "FF10203", o.Validate(ctx))

	o = testOrg()
	o.Namespace = "!namespace"
	assert.Regexp(t, "FF10131", o.Validate(ctx))

	o = testOrg()
	o.Name = "!name"
	assert.Regexp(t, "FF10131", o.Validate(ctx))

	o = testOrg()
	o.Type = IdentityType("wrong")
	assert.Regexp(t, "FF10362", o.Validate(ctx))

	o = testOrg()
	o.Description = string(make([]byte, 4097))
	assert.Regexp(t, "FF10188", o.Validate(ctx))

	o = testOrg()
	o.DID = "did:firefly:node/node1"
	assert.Regexp(t, "FF10363", o.Validate(ctx))

	o = testOrg()
	o.Namespace = "nonsystem"
	assert.Regexp(t, "FF10361", o.Validate(ctx))

}

func TestIdentityValidationNodes(t *testing.T) {

	ctx := context.Background()
	n := testNode()
	assert.NoError(t, n.Validate(ctx))

	n = testNode()
	n.Parent = nil
	assert.Regexp(t, "FF10360", n.Validate(ctx))

	n = testNode()
	n.DID = "did:firefly:org/org1"
	assert.Regexp(t, "FF10363", n.Validate(ctx))

	n = testNode()
	n.Namespace = "nonsystem"
	assert.Regexp(t, "FF10361", n.Validate(ctx))

}

func TestIdentityValidationCustom(t *testing.T) {

	ctx := context.Background()
	c := testCustom("ns1", "custom1")
	assert.NoError(t, c.Validate(ctx))

	c = testCustom("ns1", "custom1")
	c.Parent = nil
	assert.Regexp(t, "FF10360", c.Validate(ctx))

	c = testCustom("ns1", "custom1")
	c.DID = "did:firefly:ns/ns2/custom1"
	assert.Regexp(t, "FF10363", c.Validate(ctx))

	c = testCustom("ns1", "custom1")
	c.Namespace = SystemNamespace
	assert.Regexp(t, "FF10359", c.Validate(ctx))

}

func TestIdentityCompare(t *testing.T) {

	ctx := context.Background()

	getMatching := func() (*IdentityBase, *IdentityBase) {
		i1 := testCustom("ns1", "custom1")
		i2 := testCustom("ns1", "custom1")
		*i1.ID = *i2.ID
		*i1.Parent = *i2.Parent
		return &i1.IdentityBase, &i2.IdentityBase
	}

	i1, i2 := getMatching()
	assert.True(t, i1.Equals(ctx, i2))

	i1, i2 = getMatching()
	i1.ID = NewUUID()
	assert.False(t, i1.Equals(ctx, i2))

	i1, i2 = getMatching()
	i1.Parent = NewUUID()
	assert.False(t, i1.Equals(ctx, i2))
	i1.Parent = nil
	assert.False(t, i1.Equals(ctx, i2))

	i1, i2 = getMatching()
	i1.DID = "bad"
	assert.False(t, i1.Equals(ctx, i2))

	i1, i2 = getMatching()
	i2.DID = "bad"
	assert.False(t, i1.Equals(ctx, i2))

	i1, i2 = getMatching()
	i2 = &testCustom("ns1", "custom2").IdentityBase
	*i2.ID = *i1.ID
	i1.Parent = nil
	i2.Parent = nil
	assert.False(t, i1.Equals(ctx, i2))

	i1, i2 = getMatching()
	i2 = &testCustom("ns2", "custom1").IdentityBase
	*i2.ID = *i1.ID
	i1.Parent = nil
	i2.Parent = nil
	assert.False(t, i1.Equals(ctx, i2))
}

func TestDefinitionObjects(t *testing.T) {

	o := testOrg()
	assert.Equal(t, "7ea456fa05fc63778e7c4cb22d0498d73f184b2778c11fd2ba31b5980f8490b9", o.IdentityBase.Topic())
	assert.Equal(t, o.Topic(), o.IdentityBase.Topic())

	ic := IdentityClaim{
		Identity: o,
	}
	assert.Equal(t, o.Topic(), ic.Topic())
	claimMsg := NewUUID()
	ic.SetBroadcastMessage(claimMsg)
	assert.Equal(t, *claimMsg, *o.Messages.Claim)

	iv := IdentityVerification{
		Identity:  o.IdentityBase,
		SignerRef: o,
	}
	assert.Equal(t, o.Topic(), iv.Topic())
	verificationMsg := NewUUID()
	iv.SetBroadcastMessage(verificationMsg)
	assert.Equal(t, *verificationMsg, *o.Messages.Verification)

	var iu Definition = &IdentityUpdate{
		Identity:    o.IdentityBase,
		IdentityRef: o,
	}
	assert.Equal(t, o.Topic(), iu.Topic())
	updateMsg := NewUUID()
	iu.SetBroadcastMessage(updateMsg)
	assert.Equal(t, *updateMsg, *o.Messages.Update)

}
