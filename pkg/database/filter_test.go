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

package database

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestBuildMessageFilter(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	f, err := fb.And().
		Condition(fb.Eq("namespace", "ns1")).
		Condition(fb.Or().
			Condition(fb.Eq("id", "35c11cba-adff-4a4d-970a-02e3a0858dc8")).
			Condition(fb.Eq("id", "caefb9d1-9fc9-4d6a-a155-514d3139adf7")),
		).
		Condition(fb.Gt("sequence", 12345)).
		Condition(fb.Eq("confirmed", nil)).
		Skip(50).
		Limit(25).
		Count(true).
		Sort("namespace").
		Descending().
		Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( namespace == 'ns1' ) && ( ( id == '35c11cba-adff-4a4d-970a-02e3a0858dc8' ) || ( id == 'caefb9d1-9fc9-4d6a-a155-514d3139adf7' ) ) && ( sequence > 12345 ) && ( confirmed == null ) sort=-namespace skip=50 limit=25 count=true", f.String())
}

func TestBuildMessageFilter2(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	f, err := fb.Gt("sequence", "0").
		Sort("sequence").
		Ascending().
		Finalize()

	assert.NoError(t, err)
	assert.Equal(t, "sequence > 0 sort=sequence", f.String())
}

func TestBuildMessageFilter3(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	f, err := fb.And(
		fb.In("created", []driver.Value{1, 2, 3}),
		fb.NotIn("created", []driver.Value{1, 2, 3}),
		fb.Lt("created", "0"),
		fb.Lte("created", "0"),
		fb.Gte("created", "0"),
		fb.Neq("created", "0"),
		fb.Gt("sequence", 12345),
		fb.Contains("topics", "abc"),
		fb.NotContains("topics", "def"),
		fb.IContains("topics", "ghi"),
		fb.NotIContains("topics", "jkl"),
	).
		Sort("-created").
		Sort("topics").
		Sort("-sequence").
		Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( created IN [1000000000,2000000000,3000000000] ) && ( created NI [1000000000,2000000000,3000000000] ) && ( created < 0 ) && ( created <= 0 ) && ( created >= 0 ) && ( created != 0 ) && ( sequence > 12345 ) && ( topics %= 'abc' ) && ( topics %! 'def' ) && ( topics ^= 'ghi' ) && ( topics ^! 'jkl' ) sort=-created,topics,-sequence", f.String())
}

func TestBuildMessageBadInFilterField(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	_, err := fb.And(
		fb.In("!wrong", []driver.Value{"a", "b", "c"}),
	).Finalize()
	assert.Regexp(t, "FF10148", err)
}

func TestBuildMessageBadInFilterValue(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	_, err := fb.And(
		fb.In("sequence", []driver.Value{"!integer"}),
	).Finalize()
	assert.Regexp(t, "FF10149", err)
}

func TestBuildMessageUUIDConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	u := fftypes.MustParseUUID("4066ABDC-8BBD-4472-9D29-1A55B467F9B9")
	b32 := fftypes.UUIDBytes(u)
	var nilB32 *fftypes.Bytes32
	f, err := fb.And(
		fb.Eq("id", u),
		fb.Eq("id", *u),
		fb.In("id", []driver.Value{*u}),
		fb.Eq("id", u.String()),
		fb.Neq("id", nil),
		fb.Eq("id", b32),
		fb.Neq("id", *b32),
		fb.Eq("id", ""),
		fb.Eq("id", nilB32),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( id == '4066abdc-8bbd-4472-9d29-1a55b467f9b9' ) && ( id == '4066abdc-8bbd-4472-9d29-1a55b467f9b9' ) && ( id IN ['4066abdc-8bbd-4472-9d29-1a55b467f9b9'] ) && ( id == '4066abdc-8bbd-4472-9d29-1a55b467f9b9' ) && ( id != null ) && ( id == '4066abdc-8bbd-4472-9d29-1a55b467f9b9' ) && ( id != '4066abdc-8bbd-4472-9d29-1a55b467f9b9' ) && ( id == null ) && ( id == null )", f.String())
}

func TestBuildMessageBytes32Convert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	b32, _ := fftypes.ParseBytes32(context.Background(), "7f4806535f8b3d9bf178af053d2bbdb46047365466ed16bbb0732a71492bdaf0")
	var nilB32 *fftypes.Bytes32
	f, err := fb.And(
		fb.Eq("hash", b32),
		fb.Eq("hash", *b32),
		fb.In("hash", []driver.Value{*b32}),
		fb.Eq("hash", b32.String()),
		fb.Neq("hash", nil),
		fb.Eq("hash", ""),
		fb.Eq("hash", nilB32),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( hash == '7f4806535f8b3d9bf178af053d2bbdb46047365466ed16bbb0732a71492bdaf0' ) && ( hash == '7f4806535f8b3d9bf178af053d2bbdb46047365466ed16bbb0732a71492bdaf0' ) && ( hash IN ['7f4806535f8b3d9bf178af053d2bbdb46047365466ed16bbb0732a71492bdaf0'] ) && ( hash == '7f4806535f8b3d9bf178af053d2bbdb46047365466ed16bbb0732a71492bdaf0' ) && ( hash != null ) && ( hash == null ) && ( hash == null )", f.String())
}
func TestBuildMessageIntConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	f, err := fb.And(
		fb.Lt("sequence", int(111)),
		fb.Lt("sequence", int32(222)),
		fb.Lt("sequence", int64(333)),
		fb.Lt("sequence", uint(444)),
		fb.Lt("sequence", uint32(555)),
		fb.Lt("sequence", uint64(666)),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( sequence < 111 ) && ( sequence < 222 ) && ( sequence < 333 ) && ( sequence < 444 ) && ( sequence < 555 ) && ( sequence < 666 )", f.String())
}

func TestBuildMessageTimeConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	f, err := fb.And(
		fb.Gt("created", int64(1621112824)),
		fb.Gt("created", 0),
		fb.Eq("created", "2021-05-15T21:07:54.123456789Z"),
		fb.Eq("created", nil),
		fb.Lt("created", fftypes.UnixTime(1621112824)),
		fb.Lt("created", *fftypes.UnixTime(1621112824)),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( created > 1621112824000000000 ) && ( created > 0 ) && ( created == 1621112874123456789 ) && ( created == null ) && ( created < 1621112824000000000 ) && ( created < 1621112824000000000 )", f.String())
}

func TestBuildMessageStringConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	u := fftypes.MustParseUUID("3f96e0d5-a10e-47c6-87a0-f2e7604af179")
	b32 := fftypes.UUIDBytes(u)
	f, err := fb.And(
		fb.Lt("namespace", int(111)),
		fb.Lt("namespace", int32(222)),
		fb.Lt("namespace", int64(333)),
		fb.Lt("namespace", uint(444)),
		fb.Lt("namespace", uint32(555)),
		fb.Lt("namespace", uint64(666)),
		fb.Lt("namespace", nil),
		fb.Lt("namespace", *u),
		fb.Lt("namespace", u),
		fb.Lt("namespace", *b32),
		fb.Lt("namespace", b32),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( namespace < '111' ) && ( namespace < '222' ) && ( namespace < '333' ) && ( namespace < '444' ) && ( namespace < '555' ) && ( namespace < '666' ) && ( namespace < '' ) && ( namespace < '3f96e0d5-a10e-47c6-87a0-f2e7604af179' ) && ( namespace < '3f96e0d5-a10e-47c6-87a0-f2e7604af179' ) && ( namespace < '3f96e0d5a10e47c687a0f2e7604af17900000000000000000000000000000000' ) && ( namespace < '3f96e0d5a10e47c687a0f2e7604af17900000000000000000000000000000000' )", f.String())
}

func TestBuildMessageBoolConvert(t *testing.T) {
	fb := PinQueryFactory.NewFilter(context.Background())
	f, err := fb.And(
		fb.Eq("masked", false),
		fb.Eq("masked", true),
		fb.Eq("masked", "false"),
		fb.Eq("masked", "true"),
		fb.Eq("masked", "True"),
		fb.Eq("masked", ""),
		fb.Eq("masked", int(111)),
		fb.Eq("masked", int32(222)),
		fb.Eq("masked", int64(333)),
		fb.Eq("masked", uint(444)),
		fb.Eq("masked", uint32(555)),
		fb.Eq("masked", uint64(666)),
		fb.Eq("masked", nil),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "( masked == false ) && ( masked == true ) && ( masked == false ) && ( masked == true ) && ( masked == true ) && ( masked == false ) && ( masked == true ) && ( masked == true ) && ( masked == true ) && ( masked == true ) && ( masked == true ) && ( masked == true ) && ( masked == false )", f.String())
}

func TestBuildMessageJSONConvert(t *testing.T) {
	fb := TransactionQueryFactory.NewFilter(context.Background())
	f, err := fb.And(
		fb.Eq("info", nil),
		fb.Eq("info", `{}`),
		fb.Eq("info", []byte(`{}`)),
		fb.Eq("info", fftypes.JSONObject{"some": "value"}),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, `( info == null ) && ( info == '{}' ) && ( info == '{}' ) && ( info == '{"some":"value"}' )`, f.String())
}

func TestBuildFFNameArrayConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	f, err := fb.And(
		fb.Eq("topics", nil),
		fb.Eq("topics", `test1`),
		fb.Eq("topics", []byte(`test2`)),
	).Finalize()
	assert.NoError(t, err)
	assert.Equal(t, `( topics == '' ) && ( topics == 'test1' ) && ( topics == 'test2' )`, f.String())
}

func TestBuildMessageFailStringConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	_, err := fb.Lt("namespace", map[bool]bool{true: false}).Finalize()
	assert.Regexp(t, "FF10149.*namespace", err)
}

func TestBuildMessageFailBoolConvert(t *testing.T) {
	fb := PinQueryFactory.NewFilter(context.Background())
	_, err := fb.Lt("masked", map[bool]bool{true: false}).Finalize()
	assert.Regexp(t, "FF10149.*masked", err)
}

func TestBuildMessageFailBypes32Convert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	_, err := fb.Lt("group", map[bool]bool{true: false}).Finalize()
	assert.Regexp(t, "FF10149.*group", err)
}

func TestBuildMessageFailInt64Convert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	_, err := fb.Lt("sequence", map[bool]bool{true: false}).Finalize()
	assert.Regexp(t, "FF10149.*sequence", err)
}

func TestBuildMessageFailTimeConvert(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	_, err := fb.Lt("created", map[bool]bool{true: false}).Finalize()
	assert.Regexp(t, "FF10149.*created", err)
}

func TestQueryFactoryBadField(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	_, err := fb.And(
		fb.Eq("wrong", "ns1"),
	).Finalize()
	assert.Regexp(t, "FF10148.*wrong", err)
}

func TestQueryFactoryBadValue(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	_, err := fb.And(
		fb.Eq("sequence", "not an int"),
	).Finalize()
	assert.Regexp(t, "FF10149.*sequence", err)
}

func TestQueryFactoryBadNestedValue(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	_, err := fb.And(
		fb.And(
			fb.Eq("sequence", "not an int"),
		),
	).Finalize()
	assert.Regexp(t, "FF10149.*sequence", err)
}

func TestQueryFactoryGetFields(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background())
	assert.NotNil(t, fb.Fields())
}

func TestQueryFactoryGetBuilder(t *testing.T) {
	fb := MessageQueryFactory.NewFilter(context.Background()).Gt("sequence", 0)
	assert.NotNil(t, fb.Builder())
}

func TestBuildMessageFailJSONConvert(t *testing.T) {
	fb := TransactionQueryFactory.NewFilter(context.Background())
	_, err := fb.Lt("info", map[bool]bool{true: false}).Finalize()
	assert.Regexp(t, "FF10149.*info", err)
}

func TestStringsForTypes(t *testing.T) {

	assert.Equal(t, "test", (&stringField{s: "test"}).String())
	assert.Equal(t, "037a025d-681d-4150-a413-05f368729c66", (&uuidField{fftypes.MustParseUUID("037a025d-681d-4150-a413-05f368729c66")}).String())
	b32 := fftypes.NewRandB32()
	assert.Equal(t, b32.String(), (&bytes32Field{b32: b32}).String())
	assert.Equal(t, "12345", (&int64Field{i: 12345}).String())
	now := fftypes.Now()
	assert.Equal(t, now.String(), (&timeField{t: now}).String())
	assert.Equal(t, `{"some":"value"}`, (&jsonField{b: []byte(`{"some":"value"}`)}).String())
	assert.Equal(t, "t1,t2", (&ffNameArrayField{na: fftypes.FFNameArray{"t1", "t2"}}).String())
	assert.Equal(t, "true", (&boolField{b: true}).String())
}
