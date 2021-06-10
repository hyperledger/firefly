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

package ql

import (
	"context"
	"fmt"

	"database/sql"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
	migrateql "github.com/golang-migrate/migrate/v4/database/ql"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/database/sqlcommon"
	"github.com/hyperledger-labs/firefly/pkg/database"

	// Import QL driver
	_ "modernc.org/ql/driver"
)

type QL struct {
	sqlcommon.SQLCommon
}

func (ql *QL) Init(ctx context.Context, prefix config.Prefix, callbacks database.Callbacks) error {
	capabilities := &database.Capabilities{}
	return ql.SQLCommon.Init(ctx, ql, prefix, callbacks, capabilities)
}

func (ql *QL) Name() string {
	return "ql"
}

func (ql *QL) PlaceholderFormat() sq.PlaceholderFormat {
	return sq.Dollar
}

func (ql *QL) UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (sq.InsertBuilder, bool) {
	// No tweaking needed, as QL supports the Go semantic for returning the id() from insert
	return insert, false
}

func (ql *QL) SequenceField(tableName string) string {
	return fmt.Sprintf("id(%s)", tableName)
}

func (ql *QL) Open(url string) (*sql.DB, error) {
	return sql.Open(ql.Name(), url)
}

func (ql *QL) GetMigrationDriver(db *sql.DB) (migratedb.Driver, error) {
	return migrateql.WithInstance(db, &migrateql.Config{})
}
