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
	"fmt"

	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
)

// testProvider uses the datadog mocking framework
type mockProvider struct {
	SQLCommon
	callbacks    *databasemocks.Callbacks
	capabilities *database.Capabilities
	prefix       config.Prefix

	mockDB *sql.DB
	mdb    sqlmock.Sqlmock

	fakePSQLInsert          bool
	openError               error
	getMigrationDriverError error
}

func newMockProvider() *mockProvider {
	mp := &mockProvider{
		prefix: config.NewPluginConfig("unittest.mockdb"),
	}
	mp.SQLCommon.InitPrefix(mp, mp.prefix)
	mp.mockDB, mp.mdb, _ = sqlmock.New()
	return mp
}

// init is a convenience to init for tests that aren't testing init itself
func (mp *mockProvider) init() (*mockProvider, sqlmock.Sqlmock) {
	_ = mp.Init(context.Background(), mp, mp.prefix, mp.callbacks, mp.capabilities)
	return mp, mp.mdb
}

func (mp *mockProvider) Name() string {
	return "mockdb"
}

func (mp *mockProvider) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Dollar
}

func (mp *mockProvider) UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (sq.InsertBuilder, bool) {
	if mp.fakePSQLInsert {
		return insert.Suffix(" RETURNING seq"), true
	}
	return insert, false
}

func (mp *mockProvider) SequenceField(tableName string) string {
	if tableName != "" {
		return fmt.Sprintf("%s.seq", tableName)
	}
	return "seq"
}

func (mp *mockProvider) Open(url string) (*sql.DB, error) {
	return mp.mockDB, mp.openError
}

func (mp *mockProvider) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return nil, mp.getMigrationDriverError
}
