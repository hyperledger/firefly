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

package sqlcommon

import (
	"context"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/stretchr/testify/assert"
)

func TestSQLFilterBuilder(t *testing.T) {

	fb := persistence.MessageFilterBuilder.New(context.Background())
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

	sel := squirrel.Select("*").From("mytable")
	sel, err = filterSelect(context.Background(), sel, f)
	assert.NoError(t, err)

	sqlFilter, args, err := sel.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "SELECT * FROM mytable WHERE (namespace = ? AND (id = ? OR id = ?) AND created > ?)", sqlFilter)
	assert.Equal(t, "ns1", args[0])
	assert.Equal(t, "35c11cba-adff-4a4d-970a-02e3a0858dc8", args[1])
	assert.Equal(t, "caefb9d1-9fc9-4d6a-a155-514d3139adf7", args[2])
	assert.Equal(t, int64(12345), args[3])
}

func TestSQLFilterBuilderExtraOps(t *testing.T) {

	fb := persistence.MessageFilterBuilder.New(context.Background())
	f, err := fb.And(
		fb.Lt("created", "0"),
		fb.Lte("created", "0"),
		fb.Gte("created", "0"),
		fb.Neq("created", "0"),
		fb.Contains("id", "abc"),
		fb.NotContains("id", "def"),
		fb.IContains("id", "ghi"),
		fb.INotContains("id", "jkl"),
	).Finalize()
	assert.NoError(t, err)

	sel := squirrel.Select("*").From("mytable")
	sel, err = filterSelect(context.Background(), sel, f)
	assert.NoError(t, err)

	sqlFilter, _, err := sel.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "SELECT * FROM mytable WHERE (created < ? AND created <= ? AND created >= ? AND created <> ? AND id LIKE ? AND id NOT LIKE ? AND id ILIKE ? AND id NOT ILIKE ?)", sqlFilter)
}

func TestSQLFilterBuilderBadOp(t *testing.T) {

	sel := squirrel.Select("*").From("mytable")
	_, err := filterSelect(context.Background(), sel, &persistence.FilterInfo{
		Op: persistence.FilterOp("wrong"),
	})
	assert.Regexp(t, "FF10150.*wrong", err.Error())
}

func TestSQLFilterBuilderBadOpInOr(t *testing.T) {

	sel := squirrel.Select("*").From("mytable")
	_, err := filterSelect(context.Background(), sel, &persistence.FilterInfo{
		Op: persistence.FilterOpOr,
		Children: []*persistence.FilterInfo{
			{Op: persistence.FilterOp("wrong")},
		},
	})
	assert.Regexp(t, "FF10150.*wrong", err.Error())
}

func TestSQLFilterBuilderBadOpInAnd(t *testing.T) {

	sel := squirrel.Select("*").From("mytable")
	_, err := filterSelect(context.Background(), sel, &persistence.FilterInfo{
		Op: persistence.FilterOpAnd,
		Children: []*persistence.FilterInfo{
			{Op: persistence.FilterOp("wrong")},
		},
	})
	assert.Regexp(t, "FF10150.*wrong", err.Error())
}
