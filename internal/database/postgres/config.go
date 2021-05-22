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
	"github.com/kaleido-io/firefly/internal/config"
)

const (
	defaultMigrationsDirectory = "./db/migrations/postgres"
)

const (
	// PSQLConfDatabase is the database name to use (can be blank if specified in the connection URL)
	PSQLConfDatabase = "database"
	// PSQLConfURL is the PostgreSQL connection URL string
	PSQLConfURL = "url"
	// PSQLConfMigrationsAuto enables automatic migrations
	PSQLConfMigrationsAuto = "migrations.auto"
	// PSQLConfMigrationsDirectory is the directory containing the numerically ordered migration DDL files to apply to the database
	PSQLConfMigrationsDirectory = "migrations.directory"
)

func (e *Postgres) InitPrefix(prefix config.Prefix) {
	prefix.AddKnownKey(PSQLConfDatabase)
	prefix.AddKnownKey(PSQLConfURL)
	prefix.AddKnownKey(PSQLConfMigrationsAuto)
	prefix.AddKnownKey(PSQLConfMigrationsDirectory, defaultMigrationsDirectory)
}
