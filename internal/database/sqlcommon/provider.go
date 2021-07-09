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
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
)

// Provider defines the interface an individual provider muse implement to customize the SQLCommon implementation
type Provider interface {

	// Name is the name of the database driver
	Name() string

	// MigrationDir is the subdirectory for migrations
	MigrationsDir() string

	// Open creates the DB instances
	Open(url string) (*sql.DB, error)

	// GetDriver returns the driver implementation
	GetMigrationDriver(*sql.DB) (migratedb.Driver, error)

	// PlaceholderFormat gets the Squirrel placeholder format
	PlaceholderFormat() sq.PlaceholderFormat

	// UpdateInsertForReturn updates the insert query for returning the Sequenc, and returns whether it needs to be run as a query to return the Sequence field
	UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (updatedInsert sq.InsertBuilder, runAsQuery bool)

	// SequenceField must be auto added by the database to each table, via appropriate DDL in the migrations
	// Different formats exist for putting a table prefix. QL is "id(prefix)" rather than "prefix.seq"
	SequenceField(tableName string) string

	// IndividualSort returns true if individual column sorting is supported in "ORDER BY" clauses
	IndividualSort() bool
}
