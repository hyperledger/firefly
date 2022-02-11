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

package sqlcommon

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	migratedb "github.com/golang-migrate/migrate/v4/database"
)

const (
	sequenceColumn = "seq"
)

type SQLFeatures struct {
	UseILIKE              bool
	PlaceholderFormat     sq.PlaceholderFormat
	ExclusiveTableLockSQL func(table string) string
}

func DefaultSQLProviderFeatures() SQLFeatures {
	return SQLFeatures{
		UseILIKE:          false,
		PlaceholderFormat: sq.Dollar,
	}
}

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

	// Features returns database specific configuration switches
	Features() SQLFeatures

	// UpdateInsertForSequenceReturn updates the INSERT query for returning the Sequence, and returns whether it needs to be run as a query to return the Sequence field
	UpdateInsertForSequenceReturn(insert sq.InsertBuilder) (updatedInsert sq.InsertBuilder, runAsQuery bool)
}
