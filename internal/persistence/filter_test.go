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

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestBuildMessageFilter(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	f, err := fb.And().
		Condition(fb.Eq("namespace", "ns1")).
		Condition(fb.Or().
			Condition(fb.Eq("id", "35c11cba-adff-4a4d-970a-02e3a0858dc8")).
			Condition(fb.Eq("id", "caefb9d1-9fc9-4d6a-a155-514d3139adf7")),
		).
		Condition(fb.Gt("created", 12345)).
		Skip(50).
		Limit(25).
		Sort("namespace").
		Descending().
		Finalize()

	assert.NoError(t, err)
	assert.Equal(t, "( namespace == 'ns1' ) && ( ( id == '35c11cba-adff-4a4d-970a-02e3a0858dc8' ) || ( id == 'caefb9d1-9fc9-4d6a-a155-514d3139adf7' ) ) && ( created > 12345 ) sort=namespace descending skip=50 limit=25", f.String())
}

func TestBuildMessageFilter2(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	f, err := fb.Gt("created", "0").
		Sort("created").
		Ascending().
		Finalize()

	assert.NoError(t, err)
	assert.Equal(t, "created > 0 sort=created", f.String())
}

func TestBuildMessageFilter3(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	f, err := fb.And(
		fb.Lt("created", "0"),
		fb.Lte("created", "0"),
		fb.Gte("created", "0"),
		fb.Neq("created", "0"),
		fb.Contains("id", "abc"),
		fb.NotContains("id", "def"),
		fb.IContains("id", "ghi"),
		fb.NotIContains("id", "jkl"),
	).Finalize()

	assert.NoError(t, err)
	assert.Equal(t, "( created < 0 ) && ( created <= 0 ) && ( created >= 0 ) && ( created != 0 ) && ( id %= 'abc' ) && ( id %! 'def' ) && ( id ^= 'ghi' ) && ( id ^! 'jkl' )", f.String())
}

func TestBuildMessageIntConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	f, err := fb.And(
		fb.Lt("created", int(111)),
		fb.Lt("created", int32(222)),
		fb.Lt("created", int64(333)),
		fb.Lt("created", uint(444)),
		fb.Lt("created", uint32(555)),
		fb.Lt("created", uint64(666)),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( created < 111 ) && ( created < 222 ) && ( created < 333 ) && ( created < 444 ) && ( created < 555 ) && ( created < 666 )", f.String())
}

func TestBuildMessageStringConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	u := uuid.MustParse("3f96e0d5-a10e-47c6-87a0-f2e7604af179")
	b32 := fftypes.UUIDBytes(u)
	f, err := fb.And(
		fb.Lt("namespace", int(111)),
		fb.Lt("namespace", int32(222)),
		fb.Lt("namespace", int64(333)),
		fb.Lt("namespace", uint(444)),
		fb.Lt("namespace", uint32(555)),
		fb.Lt("namespace", uint64(666)),
		fb.Lt("namespace", nil),
		fb.Lt("namespace", u),
		fb.Lt("namespace", &u),
		fb.Lt("namespace", b32),
		fb.Lt("namespace", &b32),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( namespace < '111' ) && ( namespace < '222' ) && ( namespace < '333' ) && ( namespace < '444' ) && ( namespace < '555' ) && ( namespace < '666' ) && ( namespace < '' ) && ( namespace < '3f96e0d5-a10e-47c6-87a0-f2e7604af179' ) && ( namespace < '3f96e0d5-a10e-47c6-87a0-f2e7604af179' ) && ( namespace < '3f96e0d5a10e47c687a0f2e7604af17900000000000000000000000000000000' ) && ( namespace < '3f96e0d5a10e47c687a0f2e7604af17900000000000000000000000000000000' )", f.String())
}

func TestBuildMessageFailStringConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	_, err := fb.Lt("namespace", map[bool]bool{true: false}).Finalize()
	assert.Regexp(t, "FF10149.*namespace", err.Error())
}

func TestBuildMessageFailInt64Convert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	_, err := fb.Lt("created", map[bool]bool{true: false}).Finalize()
	assert.Regexp(t, "FF10149.*created", err.Error())
}

func TestQueryFactoryBadField(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	_, err := fb.And(
		fb.Eq("wrong", "ns1"),
	).Finalize()
	assert.Regexp(t, "FF10148.*wrong", err)
}

func TestQueryFactoryBadValue(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	_, err := fb.And(
		fb.Eq("created", "not an int"),
	).Finalize()
	assert.Regexp(t, "FF10149.*created", err)
}

func TestQueryFactoryBadNestedValue(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	_, err := fb.And(
		fb.And(
			fb.Eq("created", "not an int"),
		),
	).Finalize()
	assert.Regexp(t, "FF10149.*created", err)
}

func TestQueryFactoryGetFields(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0)
	assert.NotNil(t, fb.Fields())
}

func TestQueryFactoryGetBuilder(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background(), 0).Gt("created", 0)
	assert.NotNil(t, fb.Builder())
}
