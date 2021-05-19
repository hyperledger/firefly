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

package ql

import (
	"context"
	"fmt"

	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/database/sqlcommon"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/pkg/database"

	_ "modernc.org/ql/driver"
)

type QL struct {
	sqlcommon.SQLCommon
}

func (e *QL) Name() string {
	return "ql"
}

func (e *QL) Init(ctx context.Context, prefix config.ConfigPrefix, callbacks database.Callbacks) error {

	capabilities := &database.Capabilities{}
	options := &sqlcommon.SQLCommonOptions{
		PlaceholderFormat: squirrel.Dollar,
		SequenceField: func(tableName string) string {
			return fmt.Sprintf("id(%s)", tableName)
		},
	}

	db, err := sql.Open("ql", prefix.GetString(QLConfURL))
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBInitFailed)
	}

	return sqlcommon.InitSQLCommon(ctx, &e.SQLCommon, db, callbacks, capabilities, options)
}
