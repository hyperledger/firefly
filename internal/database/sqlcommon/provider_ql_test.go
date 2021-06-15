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
	"testing"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/ql"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/stretchr/testify/assert"

	// Import the QL driver
	_ "modernc.org/ql/driver"
)

// qlTestProvider uses QL in-memory database
type qlTestProvider struct {
	SQLCommon

	prefix       config.Prefix
	t            *testing.T
	callbacks    *databasemocks.Callbacks
	capabilities *database.Capabilities
}

// newTestProvider creates a real in-memory database provider for e2e testing
func newQLTestProvider(t *testing.T) *qlTestProvider {
	tp := &qlTestProvider{
		t:            t,
		callbacks:    &databasemocks.Callbacks{},
		capabilities: &database.Capabilities{},
		prefix:       config.NewPluginConfig("unittest.db"),
	}
	tp.SQLCommon.InitPrefix(tp, tp.prefix)
	tp.prefix.Set(SQLConfDatasourceURL, "memory://")
	tp.prefix.Set(SQLConfMigrationsAuto, true)
	tp.prefix.Set(SQLConfMigrationsDirectory, "../../../db/migrations/ql")

	err := tp.Init(context.Background(), tp, tp.prefix, tp.callbacks, tp.capabilities)
	assert.NoError(tp.t, err)

	return tp
}

func (tp *qlTestProvider) Name() string {
	return "ql"
}

func (tp *qlTestProvider) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Dollar
}

func (tp *qlTestProvider) UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (sq.InsertBuilder, bool) {
	// Nothing required - QL supports the query for returning the generated ID, and we use that for the sequence
	return insert, false
}

func (tp *qlTestProvider) SequenceField(tableName string) string {
	return fmt.Sprintf("id(%s)", tableName)
}

func (tp *qlTestProvider) Open(url string) (*sql.DB, error) {
	return sql.Open("ql", url)
}

func (tp *qlTestProvider) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return ql.WithInstance(db, &ql.Config{})
}

func (tp *qlTestProvider) IndividualSort() bool {
	return false // QL does not support individual column sorting
}
