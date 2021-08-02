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
		"pool_id",
		"base_uri",
		"is_fungible",
	}
	tokenAccountColumns = []string{
		"pool_id",
		"member",
		"balance",
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
		transactionRows, err := s.queryTx(ctx, tx,
			sq.Select("seq").
				From("tokenpool").
				Where(sq.Eq{"pool_id": pool.PoolID}),
		)
		if err != nil {
			return err
		}
		existing = transactionRows.Next()
		transactionRows.Close()
	}

	isFungible := 0
	if pool.Type == fftypes.TokenTypeFungible {
		isFungible = 1
	}

	if existing {
		if err = s.updateTx(ctx, tx,
			sq.Update("tokenpool").
				Set("pool_id", pool.PoolID).
				Set("base_uri", pool.BaseURI).
				Set("is_fungible", isFungible),
			func() {
				// s.callbacks.UUIDCollectionEvent(database.CollectionTokenPools, fftypes.ChangeEventTypeUpdated, pool.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("tokenpool").
				Columns(tokenPoolColumns...).
				Values(
					pool.PoolID,
					pool.BaseURI,
					isFungible,
				),
			func() {
				// s.callbacks.UUIDCollectionEvent(database.CollectionTokenPools, fftypes.ChangeEventTypeCreated, pool.ID)
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
	var isFungible int
	err := row.Scan(
		&pool.PoolID,
		&pool.BaseURI,
		&isFungible,
	)
	if isFungible == 1 {
		pool.Type = fftypes.TokenTypeFungible
	}
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "tokenpool")
	}
	return &pool, nil
}

func (s *SQLCommon) getTokenPoolPred(ctx context.Context, desc string, pred interface{}) (message *fftypes.TokenPool, err error) {
	rows, err := s.query(ctx,
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

func (s *SQLCommon) GetTokenPoolByID(ctx context.Context, poolID string) (message *fftypes.TokenPool, err error) {
	return s.getTokenPoolPred(ctx, poolID, sq.Eq{"pool_id": poolID})
}

func (s *SQLCommon) GetTokenPools(ctx context.Context, filter database.Filter) (message []*fftypes.TokenPool, err error) {
	query, err := s.filterSelect(ctx, "", sq.Select(tokenPoolColumns...).From("tokenpool"), filter, nil, []string{"seq"})
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pools := []*fftypes.TokenPool{}
	for rows.Next() {
		d, err := s.tokenPoolResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		pools = append(pools, d)
	}

	return pools, err
}

func (s *SQLCommon) UpdateTokenBalance(ctx context.Context, poolID string, identity string, amount int) error {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	transactionRows, err := s.queryTx(ctx, tx,
		sq.Select("pool_id").
			From("tokenaccount").
			Where(sq.Eq{"pool_id": poolID, "member": identity}),
	)
	if err != nil {
		return err
	}
	existing := transactionRows.Next()
	transactionRows.Close()

	if existing {
		if err = s.updateTx(ctx, tx,
			sq.Update("tokenaccount").
				Set("pool_id", poolID).
				Set("member", identity).
				Set("balance", sq.Expr("balance + ?", amount)),
			func() {
				// s.callbacks.UUIDCollectionEvent(database.CollectionTokenPools, fftypes.ChangeEventTypeUpdated, pool.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("tokenaccount").
				Columns(tokenAccountColumns...).
				Values(
					poolID,
					identity,
					amount,
				),
			func() {
				// s.callbacks.UUIDCollectionEvent(database.CollectionTokenPools, fftypes.ChangeEventTypeCreated, pool.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}
