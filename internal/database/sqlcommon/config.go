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
	"fmt"

	"github.com/hyperledger/firefly/internal/config"
)

const (
	// SQLConfMigrationsAuto enables automatic migrations
	SQLConfMigrationsAuto = "migrations.auto"
	// SQLConfMigrationsDirectory is the directory containing the numerically ordered migration DDL files to apply to the database
	SQLConfMigrationsDirectory = "migrations.directory"
	// SQLConfDatasourceURL is the datasource connection URL string
	SQLConfDatasourceURL = "url"
	// SQLConfMaxConnections maximum connections to the database
	SQLConfMaxConnections = "maxConns"
)

const (
	defaultMigrationsDirectoryTemplate = "./db/migrations/%s"
)

func (s *SQLCommon) InitPrefix(provider Provider, prefix config.Prefix) {
	prefix.AddKnownKey(SQLConfMigrationsAuto, false)
	prefix.AddKnownKey(SQLConfDatasourceURL)
	prefix.AddKnownKey(SQLConfMigrationsDirectory, fmt.Sprintf(defaultMigrationsDirectoryTemplate, provider.MigrationsDir()))
	prefix.AddKnownKey(SQLConfMaxConnections) // some providers may set a default
}
