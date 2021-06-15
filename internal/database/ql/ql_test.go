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

package ql

import (
	"context"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/database/sqlcommon"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
)

func TestQLProvider(t *testing.T) {
	ql := &QL{}
	dcb := &databasemocks.Callbacks{}
	prefix := config.NewPluginConfig("unittest")
	ql.InitPrefix(prefix)
	prefix.Set(sqlcommon.SQLConfDatasourceURL, "!bad connection")
	err := ql.Init(context.Background(), prefix, dcb)
	assert.NoError(t, err)
	_, err = ql.GetMigrationDriver(ql.DB())
	assert.Error(t, err)

	assert.Equal(t, "ql", ql.Name())
	assert.Equal(t, sq.Dollar, ql.PlaceholderFormat())

	assert.Equal(t, "id()", ql.SequenceField(""))
	assert.Equal(t, "id(m)", ql.SequenceField("m"))

	insert := sq.Insert("test").Columns("col1").Values("val1")
	insert, query := ql.UpdateInsertForSequenceReturn(insert)
	sql, _, err := insert.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "INSERT INTO test (col1) VALUES (?)", sql)
	assert.False(t, query)

	assert.False(t, ql.IndividualSort())

}
