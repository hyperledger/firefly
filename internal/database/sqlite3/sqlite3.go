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

// +build cgo

package sqlite3

import (
	"context"

	"database/sql"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	migratesqlite3 "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/database/sqlcommon"
	"github.com/hyperledger-labs/firefly/pkg/database"

	// Import the derivation of SQLite3 CGO suported by golang-migrate
	_ "github.com/mattn/go-sqlite3"
)

type SQLite3 struct {
	sqlcommon.SQLCommon
}

func (sqlite *SQLite3) Init(ctx context.Context, prefix config.Prefix, callbacks database.Callbacks) error {
	capabilities := &database.Capabilities{}
	return sqlite.SQLCommon.Init(ctx, sqlite, prefix, callbacks, capabilities)
}

func (sqlite *SQLite3) Name() string {
	return "sqlite3"
}

func (sqlite *SQLite3) MigrationsDir() string {
	return "sqlite"
}

func (sqlite *SQLite3) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Dollar
}

func (sqlite *SQLite3) UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (sq.InsertBuilder, bool) {
	return insert, false
}

func (sqlite *SQLite3) Open(url string) (*sql.DB, error) {
	return sql.Open("sqlite3", url)
}

func (sqlite *SQLite3) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return migratesqlite3.WithInstance(db, &migratesqlite3.Config{})
}
