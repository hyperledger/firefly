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

package sqlcommon

import (
	"context"
	"database/sql"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/ql"
	"github.com/stretchr/testify/assert"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "modernc.org/ql/driver"
)

var db *sql.DB
var m *migrate.Migrate

func ensureTestDB(t *testing.T) *sql.DB {
	// We use a simple pure go DB (QL) as the reference test for the SQLCommon implementation in these unit tests.
	if db != nil {
		return db
	}

	var err error
	db, err = sql.Open("ql", "memory://")
	assert.NoError(t, err)

	driver, err := ql.WithInstance(db, &ql.Config{})
	assert.NoError(t, err)

	m, err = migrate.NewWithDatabaseInstance("file://../../../db/migrations/ql", "ql", driver)
	assert.NoError(t, err)
	err = m.Up()
	assert.NoError(t, err)

	return db
}

func TestInitSQLCommon(t *testing.T) {

	s := &SQLCommon{}
	c, err := InitSQLCommon(context.Background(), s, ensureTestDB(t), nil)
	assert.NoError(t, err)
	assert.NotNil(t, c)

}

func TestTeardown(t *testing.T) {
	ensureTestDB(t)
	err := m.Down()
	assert.NoError(t, err)
	db.Close()
	db = nil
}
