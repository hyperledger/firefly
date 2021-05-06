// Copyright © 2021 Kaleido, Inc.
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

package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildMessageFilter(t *testing.T) {
	fb := MessageFilterBuilder.New(context.Background())
	f, err := fb.And(
		fb.Eq("namespace", "ns1"),
		fb.Or(
			fb.Eq("id", "35c11cba-adff-4a4d-970a-02e3a0858dc8"),
			fb.Eq("id", "caefb9d1-9fc9-4d6a-a155-514d3139adf7"),
		),
		fb.Gt("created", "12345")).
		Skip(50).
		Limit(25).
		Sort("namespace").
		Descending().
		Finalize()

	assert.NoError(t, err)
	assert.Equal(t, "( namespace == 'ns1' ) && ( ( id == '35c11cba-adff-4a4d-970a-02e3a0858dc8' ) || ( id == 'caefb9d1-9fc9-4d6a-a155-514d3139adf7' ) ) && ( created > 12345 ) sort=namespace descending skip=50 limit=25", f.String())
}

func TestBuildMessageFilter2(t *testing.T) {
	fb := MessageFilterBuilder.New(context.Background())
	f, err := fb.Gt("created", "0").
		Sort("created").
		Ascending().
		Finalize()

	assert.NoError(t, err)
	assert.Equal(t, "created > 0 sort=created", f.String())
}

func TestBuildMessageFilter3(t *testing.T) {
	fb := MessageFilterBuilder.New(context.Background())
	f, err := fb.And(
		fb.Lt("created", "0"),
		fb.Lte("created", "0"),
		fb.Gte("created", "0"),
		fb.Neq("created", "0"),
	).Finalize()

	assert.NoError(t, err)
	assert.Equal(t, "( created < 0 ) && ( created <= 0 ) && ( created >= 0 ) && ( created != 0 )", f.String())
}

func TestFilterBuilderBadField(t *testing.T) {
	fb := MessageFilterBuilder.New(context.Background())
	_, err := fb.And(
		fb.Eq("wrong", "ns1"),
	).Finalize()
	assert.Regexp(t, "FF10148.*wrong", err)
}

func TestFilterBuilderBadValue(t *testing.T) {
	fb := MessageFilterBuilder.New(context.Background())
	_, err := fb.And(
		fb.Eq("created", "not an int"),
	).Finalize()
	assert.Regexp(t, "FF10149.*created", err)
}

func TestFilterBuilderBadNestedValue(t *testing.T) {
	fb := MessageFilterBuilder.New(context.Background())
	_, err := fb.And(
		fb.And(
			fb.Eq("created", "not an int"),
		),
	).Finalize()
	assert.Regexp(t, "FF10149.*created", err)
}
