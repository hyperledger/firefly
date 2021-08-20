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
	tokenPoolColumns = []string{
		"id",
		"namespace",
		"name",
		"pool_id",
		"type",
	}
)

func (s *SQLCommon) UpsertTokenPool(ctx context.Context, pool *fftypes.TokenPool, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		rows, _, err := s.queryTx(ctx, tx,
			sq.Select("seq").
				From("tokenpool").
				Where(sq.Eq{"id": pool.ID}),
		)
		if err != nil {
			return err
		}
		existing = rows.Next()
		rows.Close()
	}

	if existing {
		if err = s.updateTx(ctx, tx,
			sq.Update("tokenpool").
				Set("namespace", pool.Namespace).
				Set("name", pool.Name).
				Set("pool_id", pool.PoolID).
				Set("type", pool.Type).
				Where(sq.Eq{"id": pool.ID}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTokenPools, fftypes.ChangeEventTypeUpdated, pool.Namespace, pool.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("tokenpool").
				Columns(tokenPoolColumns...).
				Values(
					pool.ID,
					pool.Namespace,
					pool.Name,
					pool.PoolID,
					pool.Type,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTokenPools, fftypes.ChangeEventTypeCreated, pool.Namespace, pool.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) tokenPoolResult(ctx context.Context, row *sql.Rows) (*fftypes.TokenPool, error) {
	pool := fftypes.TokenPool{
		Type: fftypes.TokenTypeNonFungible,
	}
	err := row.Scan(
		&pool.ID,
		&pool.Namespace,
		&pool.Name,
		&pool.PoolID,
		&pool.Type,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "tokenpool")
	}
	return &pool, nil
}

func (s *SQLCommon) getTokenPoolPred(ctx context.Context, desc string, pred interface{}) (message *fftypes.TokenPool, err error) {
	rows, _, err := s.query(ctx,
		sq.Select(tokenPoolColumns...).
			From("tokenpool").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Token pool '%s' not found", desc)
		return nil, nil
	}

	pool, err := s.tokenPoolResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func (s *SQLCommon) GetTokenPoolByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.TokenPool, err error) {
	return s.getTokenPoolPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetTokenPools(ctx context.Context, filter database.Filter) (message []*fftypes.TokenPool, fr *database.FilterResult, err error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(tokenPoolColumns...).From("tokenpool"), filter, nil, []string{"seq"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	pools := []*fftypes.TokenPool{}
	for rows.Next() {
		d, err := s.tokenPoolResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		pools = append(pools, d)
	}

	return pools, s.queryRes(ctx, tx, "tokenpool", fop, fi), err
}
