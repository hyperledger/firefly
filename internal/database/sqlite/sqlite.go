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
	"fmt"

	"database/sql"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/database/sqlcommon"
	"github.com/hyperledger-labs/firefly/pkg/database"

	// Import the SQLite driver
	_ "modernc.org/sqlite"
)

type SQLite struct {
	sqlcommon.SQLCommon
}

func (sqlite *SQLite) Init(ctx context.Context, prefix config.Prefix, callbacks database.Callbacks) error {
	capabilities := &database.Capabilities{}
	return sqlite.SQLCommon.Init(ctx, sqlite, prefix, callbacks, capabilities)
}

func (sqlite *SQLite) Name() string {
	return "sqlite"
}

func (sqlite *SQLite) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Dollar
}

func (sqlite *SQLite) UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (sq.InsertBuilder, bool) {
	return insert, false
}

func (sqlite *SQLite) SequenceField(tableName string) string {
	if tableName != "" {
		return fmt.Sprintf("%s.seq", tableName)
	}
	return "seq"
}

func (sqlite *SQLite) Open(url string) (*sql.DB, error) {
	return sql.Open(sqlite.Name(), url)
}

func (sqlite *SQLite) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return migratesqlite.WithInstance(db, &migratesqlite.Config{})
}
