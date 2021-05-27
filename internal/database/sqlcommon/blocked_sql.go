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
	blockedColumns = []string{
		"id",
		"namespace",
		"context",
		"group_id",
		"message_id",
		"created",
	}
	blockedFilterTypeMap = map[string]string{
		"group":   "group_id",
		"message": "message_id",
	}
)

func (s *SQLCommon) UpsertBlocked(ctx context.Context, blocked *fftypes.Blocked, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		blockedRows, err := s.queryTx(ctx, tx,
			sq.Select("id").
				From("blocked").
				Where(
					sq.Eq{
						"namespace": blocked.Namespace,
						"context":   blocked.Context,
						"group_id":  blocked.Group,
					}),
		)
		if err != nil {
			return err
		}
		existing = blockedRows.Next()
		if existing {
			var id fftypes.UUID
			_ = blockedRows.Scan(&id)
			if blocked.ID != nil {
				if *blocked.ID != id {
					blockedRows.Close()
					return database.IDMismatch
				}
			}
			blocked.ID = &id // Update on returned object
		}
		blockedRows.Close()
	}

	if existing {

		// Update the blocked
		if err = s.updateTx(ctx, tx,
			sq.Update("blocked").
				Set("namespace", blocked.Namespace).
				Set("context", blocked.Context).
				Set("group_id", blocked.Group).
				Set("message_id", blocked.Message).
				Set("created", blocked.Created).
				Where(sq.Eq{"id": blocked.ID}),
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("blocked").
				Columns(blockedColumns...).
				Values(
					blocked.ID,
					blocked.Namespace,
					blocked.Context,
					blocked.Group,
					blocked.Message,
					blocked.Created,
				),
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) blockedResult(ctx context.Context, row *sql.Rows) (*fftypes.Blocked, error) {
	blocked := fftypes.Blocked{}
	err := row.Scan(
		&blocked.ID,
		&blocked.Namespace,
		&blocked.Context,
		&blocked.Group,
		&blocked.Message,
		&blocked.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "blocked")
	}
	return &blocked, nil
}

func (s *SQLCommon) GetBlockedByContext(ctx context.Context, ns, context string, groupID *fftypes.UUID) (message *fftypes.Blocked, err error) {

	rows, err := s.query(ctx,
		sq.Select(blockedColumns...).
			From("blocked").
			Where(sq.Eq{
				"namespace": ns,
				"context":   context,
				"group_id":  groupID,
			}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Blocked '%s:%s:%s' not found", ns, context, groupID)
		return nil, nil
	}

	blocked, err := s.blockedResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return blocked, nil
}

func (s *SQLCommon) GetBlocked(ctx context.Context, filter database.Filter) (message []*fftypes.Blocked, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(blockedColumns...).From("blocked"), filter, blockedFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocked := []*fftypes.Blocked{}
	for rows.Next() {
		d, err := s.blockedResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		blocked = append(blocked, d)
	}

	return blocked, err

}

func (s *SQLCommon) UpdateBlocked(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("blocked"), update, blockedFilterTypeMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) DeleteBlocked(ctx context.Context, ns, context string, groupID *fftypes.UUID) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("blocked").Where(sq.Eq{
		"namespace": ns,
		"context":   context,
		"group_id":  groupID,
	}))
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
