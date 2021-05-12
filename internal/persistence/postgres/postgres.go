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

package postgres

import (
	"context"

	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/kaleido-io/firefly/internal/persistence/sqlcommon"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	_ "github.com/lib/pq"
)

type Postgres struct {
	sqlcommon.SQLCommon
}

func (e *Postgres) Init(ctx context.Context, prefix config.ConfigPrefix, events persistence.Events) error {
	InitConfigPrefix(prefix)

	capabilities := &persistence.Capabilities{}
	options := &sqlcommon.SQLCommonOptions{
		PlaceholderFormat: squirrel.Dollar,
		SequenceField:     "seq",
	}

	db, err := sql.Open("postgres", prefix.GetString(PSQLConfURL))
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBInitFailed)
	}

	if prefix.GetBool(PSQLConfMigrationsAuto) {
		if err := e.applyDbMigrations(ctx, prefix, db); err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgDBMigrationFailed)
		}
	}

	return sqlcommon.InitSQLCommon(ctx, &e.SQLCommon, db, events, capabilities, options)
}

func (e *Postgres) applyDbMigrations(ctx context.Context, prefix config.ConfigPrefix, db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+prefix.GetString(PSQLConfMigrationsDirectory),
		"postgres", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			return err
		}
	}
	return nil
}
