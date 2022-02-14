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

//go:build cgo
// +build cgo

package sqlite3

import (
	"context"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/database/sqlcommon"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
)

func TestSQLite3GoProvider(t *testing.T) {
	sqlite := &SQLite3{}
	dcb := &databasemocks.Callbacks{}
	prefix := config.NewPluginConfig("unittest")
	sqlite.InitPrefix(prefix)
	prefix.Set(sqlcommon.SQLConfDatasourceURL, "!wrong://")
	err := sqlite.Init(context.Background(), prefix, dcb)
	assert.NoError(t, err)
	_, err = sqlite.GetMigrationDriver(sqlite.DB())
	assert.Error(t, err)

	db, err := sqlite.Open("file::memory:")
	assert.NoError(t, err)
	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	conn.Close()

	assert.Equal(t, "sqlite3", sqlite.Name())
	assert.Equal(t, sq.Dollar, sqlite.Features().PlaceholderFormat)

	insert := sq.Insert("test").Columns("col1").Values("val1")
	insert, query := sqlite.ApplyInsertQueryCustomizations(insert, false)
	sql, _, err := insert.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "INSERT INTO test (col1) VALUES (?)", sql)
	assert.False(t, query)
}
