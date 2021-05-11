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

// +build cgo

package sqlite

import (
	"context"

	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/kaleido-io/firefly/internal/persistence/sqlcommon"
	"github.com/spf13/viper"

	_ "github.com/mattn/go-sqlite3"
)

type SQLite struct {
	sqlcommon.SQLCommon
}

func (e *SQLite) Init(ctx context.Context, config config.PluginConfig, events persistence.Events) error {
	capabilities := &persistence.Capabilities{}
	options := &sqlcommon.SQLCommonOptions{
		PlaceholderFormat: squirrel.Dollar,
		SequenceField:     "seq",
	}

	db, err := sql.Open("sqlite", viper.GetString("URL"))
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBInitFailed)
	}

	return sqlcommon.InitSQLCommon(ctx, &e.SQLCommon, db, events, capabilities, options)
}

func (e *SQLite) ConfigInterface() interface{} { return &Config{} }
