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

package postgres

import (
	"context"
	"fmt"

	"database/sql"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/database/sqlcommon"
	"github.com/hyperledger-labs/firefly/pkg/database"

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

func (psql *Postgres) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Dollar
}

func (psql *Postgres) UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (sq.InsertBuilder, bool) {
	return insert.Suffix(" RETURNING seq"), true
}

func (psql *Postgres) SequenceField(tableName string) string {
	if tableName != "" {
		return fmt.Sprintf("%s.seq", tableName)
	}
	return "seq"
}

func (psql *Postgres) Open(url string) (*sql.DB, error) {
	return sql.Open(psql.Name(), url)
}

func (psql *Postgres) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return postgres.WithInstance(db, &postgres.Config{})
}

func (psql *Postgres) IndividualSort() bool {
	return true
}
