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

package sqlcommon

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"testing"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"

	// Import SQLite driver
	_ "github.com/mattn/go-sqlite3"
)

// sqliteGoTestProvider uses QL in-memory database
type sqliteGoTestProvider struct {
	SQLCommon

	prefix       config.Prefix
	t            *testing.T
	callbacks    *databasemocks.Callbacks
	capabilities *database.Capabilities
}

// newTestProvider creates a real in-memory database provider for e2e testing
func newSQLiteTestProvider(t *testing.T) (*sqliteGoTestProvider, func()) {
	tp := &sqliteGoTestProvider{
		t:            t,
		callbacks:    &databasemocks.Callbacks{},
		capabilities: &database.Capabilities{},
		prefix:       config.NewPluginConfig("unittest.db"),
	}
	tp.SQLCommon.InitPrefix(tp, tp.prefix)
	dir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	tp.prefix.Set(SQLConfDatasourceURL, "file::memory:")
	tp.prefix.Set(SQLConfMigrationsAuto, true)
	tp.prefix.Set(SQLConfMigrationsDirectory, "../../../db/migrations/sqlite")
	tp.prefix.Set(SQLConfMaxConnections, 1)

	err = tp.Init(context.Background(), tp, tp.prefix, tp.callbacks, tp.capabilities)
	assert.NoError(tp.t, err)

	return tp, func() {
		tp.Close()
		_ = os.RemoveAll(dir)
	}
}

func (tp *sqliteGoTestProvider) Name() string {
	return "sqlite3"
}

func (tp *sqliteGoTestProvider) MigrationsDir() string {
	return "sqlite"
}

func (psql *sqliteGoTestProvider) Features() SQLFeatures {
	features := DefaultSQLProviderFeatures()
	features.PlaceholderFormat = sq.Dollar
	features.UseILIKE = false // Not supported
	return features
}

func (tp *sqliteGoTestProvider) UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (sq.InsertBuilder, bool) {
	// Nothing required - QL supports the query for returning the generated ID, and we use that for the sequence
	return insert, false
}

func (tp *sqliteGoTestProvider) Open(url string) (*sql.DB, error) {
	return sql.Open("sqlite3", url)
}

func (tp *sqliteGoTestProvider) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return sqlite3.WithInstance(db, &sqlite3.Config{})
}
