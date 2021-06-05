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
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

var (
	contextColumns = []string{
		"hash",
		"nonce",
		"group_id",
		"topic",
	}
	contextFilterTypeMap = map[string]string{
		"group": "group_id",
	}
)

func (s *SQLCommon) UpsertContextNextNonce(ctx context.Context, context *fftypes.Context) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	// Do a select within the transaction to detemine if the UUID already exists
	contextRows, err := s.queryTx(ctx, tx,
		sq.Select("nonce", s.provider.SequenceField("")).
			From("contexts").
			Where(
				sq.Eq{"hash": context.Hash}),
	)
	if err != nil {
		return err
	}
	existing = contextRows.Next()

	if existing {
		var existingNonce, sequence int64
		err := contextRows.Scan(&existingNonce, &sequence)
		if err != nil {
			contextRows.Close()
			return i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contexts")
		}
		context.Nonce = existingNonce + 1
		contextRows.Close()

		// Update the context
		if err = s.updateTx(ctx, tx,
			sq.Update("contexts").
				Set("nonce", context.Nonce).
				Where(sq.Eq{s.provider.SequenceField(""): sequence}),
		); err != nil {
			return err
		}
	} else {
		contextRows.Close()

		context.Nonce = 0
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("contexts").
				Columns(contextColumns...).
				Values(
					context.Hash,
					context.Nonce,
					context.Group,
					context.Topic,
				),
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contextResult(ctx context.Context, row *sql.Rows) (*fftypes.Context, error) {
	context := fftypes.Context{}
	err := row.Scan(
		&context.Hash,
		&context.Nonce,
		&context.Group,
		&context.Topic,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contexts")
	}
	return &context, nil
}

func (s *SQLCommon) GetContext(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.Context, err error) {

	rows, err := s.query(ctx,
		sq.Select(contextColumns...).
			From("contexts").
			Where(sq.Eq{"hash": hash}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Context '%s' not found", hash)
		return nil, nil
	}

	context, err := s.contextResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return context, nil
}

func (s *SQLCommon) GetContexts(ctx context.Context, filter database.Filter) (message []*fftypes.Context, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(contextColumns...).From("contexts"), filter, contextFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	context := []*fftypes.Context{}
	for rows.Next() {
		d, err := s.contextResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		context = append(context, d)
	}

	return context, err

}

func (s *SQLCommon) DeleteContext(ctx context.Context, hash *fftypes.Bytes32) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("contexts").Where(sq.Eq{
		"hash": hash,
	}))
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
