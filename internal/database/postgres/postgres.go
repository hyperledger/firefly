// Copyright © 2022 Kaleido, Inc.
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

package postgres

import (
	"context"
	"fmt"

	"database/sql"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/database/sqlcommon"
	"github.com/hyperledger/firefly/pkg/database"

	// Import pq driver
	_ "github.com/lib/pq"
)

type Postgres struct {
	sqlcommon.SQLCommon
}

func (psql *Postgres) Init(ctx context.Context, prefix config.Prefix, callbacks database.Callbacks) error {
	capabilities := &database.Capabilities{}
	return psql.SQLCommon.Init(ctx, psql, prefix, callbacks, capabilities)
}

func (psql *Postgres) Name() string {
	return "postgres"
}

func (psql *Postgres) MigrationsDir() string {
	return psql.Name()
}

func (psql *Postgres) Features() sqlcommon.SQLFeatures {
	features := sqlcommon.DefaultSQLProviderFeatures()
	features.PlaceholderFormat = sq.Dollar
	features.UseILIKE = false // slower than lower()
	features.ExclusiveTableLockSQL = func(table string) string {
		return fmt.Sprintf(`LOCK TABLE "%s" IN EXCLUSIVE MODE;`, table)
	}
	features.MultiRowInsert = true
	return features
}

func (psql *Postgres) ApplyInsertQueryCustomizations(insert sq.InsertBuilder, requestConflictEmptyResult bool) (sq.InsertBuilder, bool) {
	suffix := " RETURNING seq"
	if requestConflictEmptyResult {
		// Caller wants us to return an empty result set on insert conflict, rather than an error
		suffix = fmt.Sprintf(" ON CONFLICT DO NOTHING%s", suffix)
	}
	return insert.Suffix(suffix), true
}

func (psql *Postgres) Open(url string) (*sql.DB, error) {
	return sql.Open(psql.Name(), url)
}

func (psql *Postgres) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return postgres.WithInstance(db, &postgres.Config{})
}
