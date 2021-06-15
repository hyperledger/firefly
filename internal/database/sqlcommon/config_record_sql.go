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
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var (
	configRecordColumns = []string{
		"config_key",
		"config_value",
	}
	configRecordFilterFieldMap = map[string]string{
		"key":   "config_key",
		"value": "config_value",
	}
)

func (s *SQLCommon) UpsertConfigRecord(ctx context.Context, configRecord *fftypes.ConfigRecord, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to determine if the key already exists
		configRows, err := s.queryTx(ctx, tx,
			sq.Select("config_key").
				From("config").
				Where(sq.Eq{"config_key": configRecord.Key}),
		)
		if err != nil {
			return err
		}
		existing = configRows.Next()
		configRows.Close()
	}

	if existing {
		// Update the config record
		if err = s.updateTx(ctx, tx,
			sq.Update("config").
				Set("config_value", configRecord.Value).
				Where(sq.Eq{"config_key": configRecord.Key}),
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("config").
				Columns(configRecordColumns...).
				Values(
					configRecord.Key,
					configRecord.Value,
				),
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) configRecordResult(ctx context.Context, row *sql.Rows) (*fftypes.ConfigRecord, error) {
	configRecord := fftypes.ConfigRecord{}
	err := row.Scan(
		&configRecord.Key,
		&configRecord.Value,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "config")
	}
	return &configRecord, nil
}

func (s *SQLCommon) GetConfigRecord(ctx context.Context, key string) (result *fftypes.ConfigRecord, err error) {
	rows, err := s.query(ctx,
		sq.Select(configRecordColumns...).
			From("config").
			Where(sq.Eq{"config_key": key}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Config record '%s' not found", key)
		return nil, nil
	}

	configRecord, err := s.configRecordResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return configRecord, nil
}

func (s *SQLCommon) GetConfigRecords(ctx context.Context, filter database.Filter) (result []*fftypes.ConfigRecord, err error) {
	query, err := s.filterSelect(ctx, "", sq.Select(configRecordColumns...).From("config"), filter, configRecordFilterFieldMap, []string{"sequence"})
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configRecord := []*fftypes.ConfigRecord{}
	for rows.Next() {
		d, err := s.configRecordResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		configRecord = append(configRecord, d)
	}

	return configRecord, err

}

func (s *SQLCommon) DeleteConfigRecord(ctx context.Context, key string) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("config").Where(sq.Eq{
		"config_key": key,
	}))
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
