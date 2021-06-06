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
	groupcontextColumns = []string{
		"hash",
		"nonce",
		"group_id",
		"topic",
	}
	groupcontextFilterTypeMap = map[string]string{
		"group": "group_id",
	}
)

func (s *SQLCommon) UpsertGroupContextNextNonce(ctx context.Context, groupcontext *fftypes.GroupContext) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	// Do a select within the transaction to detemine if the UUID already exists
	groupcontextRows, err := s.queryTx(ctx, tx,
		sq.Select("nonce", s.provider.SequenceField("")).
			From("groupcontexts").
			Where(
				sq.Eq{"hash": groupcontext.Hash}),
	)
	if err != nil {
		return err
	}
	existing = groupcontextRows.Next()

	if existing {
		var existingNonce, sequence int64
		err := groupcontextRows.Scan(&existingNonce, &sequence)
		if err != nil {
			groupcontextRows.Close()
			return i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "groupcontexts")
		}
		groupcontext.Nonce = existingNonce + 1
		groupcontextRows.Close()

		// Update the groupcontext
		if err = s.updateTx(ctx, tx,
			sq.Update("groupcontexts").
				Set("nonce", groupcontext.Nonce).
				Where(sq.Eq{s.provider.SequenceField(""): sequence}),
		); err != nil {
			return err
		}
	} else {
		groupcontextRows.Close()

		groupcontext.Nonce = 0
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("groupcontexts").
				Columns(groupcontextColumns...).
				Values(
					groupcontext.Hash,
					groupcontext.Nonce,
					groupcontext.Group,
					groupcontext.Topic,
				),
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) groupcontextResult(ctx context.Context, row *sql.Rows) (*fftypes.GroupContext, error) {
	groupcontext := fftypes.GroupContext{}
	err := row.Scan(
		&groupcontext.Hash,
		&groupcontext.Nonce,
		&groupcontext.Group,
		&groupcontext.Topic,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "groupcontexts")
	}
	return &groupcontext, nil
}

func (s *SQLCommon) GetGroupContext(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.GroupContext, err error) {

	rows, err := s.query(ctx,
		sq.Select(groupcontextColumns...).
			From("groupcontexts").
			Where(sq.Eq{"hash": hash}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("GroupContext '%s' not found", hash)
		return nil, nil
	}

	groupcontext, err := s.groupcontextResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return groupcontext, nil
}

func (s *SQLCommon) GetGroupContexts(ctx context.Context, filter database.Filter) (message []*fftypes.GroupContext, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(groupcontextColumns...).From("groupcontexts"), filter, groupcontextFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groupcontext := []*fftypes.GroupContext{}
	for rows.Next() {
		d, err := s.groupcontextResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		groupcontext = append(groupcontext, d)
	}

	return groupcontext, err

}

func (s *SQLCommon) DeleteGroupContext(ctx context.Context, hash *fftypes.Bytes32) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("groupcontexts").Where(sq.Eq{
		"hash": hash,
	}))
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
