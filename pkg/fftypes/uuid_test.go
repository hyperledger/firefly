// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, souware
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fftypes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUUID(t *testing.T) {
	assert.NotNil(t, NewUUID())
}

func TestDatabaseSerialization(t *testing.T) {

	var u *UUID
	v, err := u.Value() // nill
	assert.NoError(t, err)
	assert.Nil(t, v)
	assert.Equal(t, "", u.String())

	u, err = ParseUUID(context.Background(), "!not an id")
	assert.Regexp(t, "FF10142", err)
	u, err = ParseUUID(context.Background(), "03D31DFB-9DBB-43F2-9E0B-84DD3D293499")
	assert.NoError(t, err)
	v, err = u.Value()
	assert.NoError(t, err)
	assert.Equal(t, "03d31dfb-9dbb-43f2-9e0b-84dd3d293499", v)

	err = u.Scan("8A57D469-D123-4CD1-81B2-6371AFB87C21")
	assert.NoError(t, err)
	assert.Equal(t, "8a57d469-d123-4cd1-81b2-6371afb87c21", u.String())

}

func TestBinaryMarshaling(t *testing.T) {

	u := MustParseUUID("03D31DFB-9DBB-43F2-9E0B-84DD3D293499")
	b, err := u.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, u[:], b)

	var u2 UUID
	err = u2.UnmarshalBinary(u[:])
	assert.NoError(t, err)
	assert.Equal(t, "03d31dfb-9dbb-43f2-9e0b-84dd3d293499", u2.String())

}

func TestSafeEquals(t *testing.T) {

	var u1, u2 *UUID
	assert.True(t, u1.Equals(u2))
	u1 = NewUUID()
	assert.False(t, u1.Equals(u2))
	u2 = MustParseUUID(u1.String())
	assert.True(t, u1.Equals(u2))
	u2 = NewUUID()
	assert.False(t, u1.Equals(u2))

}

func TestHashBucket(t *testing.T) {

	u1 := MustParseUUID("03D31DFB-9DBB-43F2-9E0B-84DD3D293499")
	assert.Equal(t, 64, u1.HashBucket(255))
	assert.Equal(t, 9, u1.HashBucket(16))

	u2 := MustParseUUID("8a57d469-d123-4cd1-81b2-6371afb87c21")
	assert.Equal(t, 15, u2.HashBucket(255))
	assert.Equal(t, 1, u2.HashBucket(4))

	u3 := MustParseUUID("8a57d469-d123-4cd1-0000-000000000000")
	assert.Equal(t, 0, u3.HashBucket(2))
	assert.Equal(t, 0, u3.HashBucket(2))

	assert.Equal(t, 0, ((*UUID)(nil)).HashBucket(12345))

}
