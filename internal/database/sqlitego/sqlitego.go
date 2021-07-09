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

package sqlitego

import (
	"context"
	"fmt"

	"database/sql"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/database/sqlcommon"
	"github.com/hyperledger-labs/firefly/pkg/database"

	// Import the pure Go SQLite driver
	_ "modernc.org/sqlite"
)

type SQLiteGo struct {
	sqlcommon.SQLCommon
}

func (sqlite *SQLiteGo) Init(ctx context.Context, prefix config.Prefix, callbacks database.Callbacks) error {
	capabilities := &database.Capabilities{}
	return sqlite.SQLCommon.Init(ctx, sqlite, prefix, callbacks, capabilities)
}

func (sqlite *SQLiteGo) Name() string {
	return "sqlitego"
}

func (sqlite *SQLiteGo) MigrationsDir() string {
	return "sqlite"
}

func (sqlite *SQLiteGo) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Dollar
}

func (sqlite *SQLiteGo) UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (sq.InsertBuilder, bool) {
	return insert, false
}

func (sqlite *SQLiteGo) SequenceField(tableName string) string {
	if tableName != "" {
		return fmt.Sprintf("%s.seq", tableName)
	}
	return "seq"
}

func (sqlite *SQLiteGo) Open(url string) (*sql.DB, error) {
	return sql.Open("sqlite", url)
}

func (sqlite *SQLiteGo) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return migratesqlite.WithInstance(db, &migratesqlite.Config{})
}

func (sqlite *SQLiteGo) IndividualSort() bool {
	return true
}
