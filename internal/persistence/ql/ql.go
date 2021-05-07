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

	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/persistence"
	"github.com/kaleido-io/firefly/internal/persistence/sqlcommon"

	_ "modernc.org/ql/driver"
)

type QL struct {
	sqlcommon.SQLCommon

	conf *Config
}

func (e *QL) Init(ctx context.Context, conf interface{}, events persistence.Events) error {
	e.conf = conf.(*Config)
	capabilities := &persistence.Capabilities{}
	options := &sqlcommon.SQLCommonOptions{
		PlaceholderFormat: squirrel.Dollar,
		SequenceField:     "id()",
	}

	db, err := sql.Open("ql", e.conf.URL)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBInitFailed)
	}

	return sqlcommon.InitSQLCommon(ctx, &e.SQLCommon, db, events, capabilities, options)
}

func (e *QL) ConfigInterface() interface{} { return &Config{} }
