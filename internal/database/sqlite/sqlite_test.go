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

package sqlite

import (
	"context"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/database/sqlcommon"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
)

func TestSQLiteProvider(t *testing.T) {
	sqlite := &SQLite{}
	dcb := &databasemocks.Callbacks{}
	prefix := config.NewPluginConfig("unittest")
	sqlite.InitPrefix(prefix)
	prefix.Set(sqlcommon.SQLConfDatasourceURL, "!wrong://")
	err := sqlite.Init(context.Background(), prefix, dcb)
	assert.NoError(t, err)
	_, err = sqlite.GetMigrationDriver(sqlite.DB())
	assert.Error(t, err)

	assert.Equal(t, "sqlite", sqlite.Name())
	assert.Equal(t, sq.Dollar, sqlite.PlaceholderFormat())

	assert.Equal(t, "seq", sqlite.SequenceField(""))
	assert.Equal(t, "m.seq", sqlite.SequenceField("m"))

	insert := sq.Insert("test").Columns("col1").Values("val1")
	insert, query := sqlite.UpdateInsertForSequenceReturn(insert)
	sql, _, err := insert.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "INSERT INTO test (col1) VALUES (?)", sql)
	assert.False(t, query)
}
