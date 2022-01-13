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

package postgres

import (
	"context"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/database/sqlcommon"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
)

func TestPostgresProvider(t *testing.T) {
	psql := &Postgres{}
	dcb := &databasemocks.Callbacks{}
	prefix := config.NewPluginConfig("unittest")
	psql.InitPrefix(prefix)
	prefix.Set(sqlcommon.SQLConfDatasourceURL, "!bad connection")
	err := psql.Init(context.Background(), prefix, dcb)
	assert.NoError(t, err)
	_, err = psql.GetMigrationDriver(psql.DB())
	assert.Error(t, err)

	assert.Equal(t, "postgres", psql.Name())
	assert.Equal(t, sq.Dollar, psql.Features().PlaceholderFormat)

	insert := sq.Insert("test").Columns("col1").Values("val1")
	insert, query := psql.UpdateInsertForSequenceReturn(insert)
	sql, _, err := insert.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, "INSERT INTO test (col1) VALUES (?)  RETURNING seq", sql)
	assert.True(t, query)
}
