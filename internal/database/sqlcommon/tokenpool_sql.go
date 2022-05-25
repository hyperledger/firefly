// Copyright © 2022 Kaleido, Inc.
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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

var (
	tokenPoolColumns = []string{
		"id",
		"namespace",
		"name",
		"standard",
		"locator",
		"type",
		"connector",
		"symbol",
		"decimals",
		"message_id",
		"state",
		"created",
		"tx_type",
		"tx_id",
		"info",
	}
	tokenPoolFilterFieldMap = map[string]string{
		"message": "message_id",
		"tx.type": "tx_type",
		"tx.id":   "tx_id",
	}
)

const tokenpoolTable = "tokenpool"

func (s *SQLCommon) UpsertTokenPool(ctx context.Context, pool *core.TokenPool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tokenpoolTable, tx,
		sq.Select("id").
			From(tokenpoolTable).
			Where(sq.And{sq.Eq{"namespace": pool.Namespace}, sq.Eq{"name": pool.Name}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()

	if existing {
		var id fftypes.UUID
		_ = rows.Scan(&id)
		if pool.ID != nil && *pool.ID != id {
			rows.Close()
			return database.IDMismatch
		}
		pool.ID = &id // Update on returned object
	}
	rows.Close()

	if existing {
		if _, err = s.updateTx(ctx, tokenpoolTable, tx,
			sq.Update(tokenpoolTable).
				Set("namespace", pool.Namespace).
				Set("name", pool.Name).
				Set("standard", pool.Standard).
				Set("locator", pool.Locator).
				Set("type", pool.Type).
				Set("connector", pool.Connector).
				Set("symbol", pool.Symbol).
				Set("decimals", pool.Decimals).
				Set("message_id", pool.Message).
				Set("state", pool.State).
				Set("tx_type", pool.TX.Type).
				Set("tx_id", pool.TX.ID).
				Set("info", pool.Info).
				Where(sq.Eq{"id": pool.ID}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTokenPools, core.ChangeEventTypeUpdated, pool.Namespace, pool.ID)
			},
		); err != nil {
			return err
		}
	} else {
		pool.Created = fftypes.Now()
		if _, err = s.insertTx(ctx, tokenpoolTable, tx,
			sq.Insert(tokenpoolTable).
				Columns(tokenPoolColumns...).
				Values(
					pool.ID,
					pool.Namespace,
					pool.Name,
					pool.Standard,
					pool.Locator,
					pool.Type,
					pool.Connector,
					pool.Symbol,
					pool.Decimals,
					pool.Message,
					pool.State,
					pool.Created,
					pool.TX.Type,
					pool.TX.ID,
					pool.Info,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTokenPools, core.ChangeEventTypeCreated, pool.Namespace, pool.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) tokenPoolResult(ctx context.Context, row *sql.Rows) (*core.TokenPool, error) {
	pool := core.TokenPool{}
	err := row.Scan(
		&pool.ID,
		&pool.Namespace,
		&pool.Name,
		&pool.Standard,
		&pool.Locator,
		&pool.Type,
		&pool.Connector,
		&pool.Symbol,
		&pool.Decimals,
		&pool.Message,
		&pool.State,
		&pool.Created,
		&pool.TX.Type,
		&pool.TX.ID,
		&pool.Info,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, tokenpoolTable)
	}
	return &pool, nil
}

func (s *SQLCommon) getTokenPoolPred(ctx context.Context, desc string, pred interface{}) (*core.TokenPool, error) {
	rows, _, err := s.query(ctx, tokenpoolTable,
		sq.Select(tokenPoolColumns...).
			From(tokenpoolTable).
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

func (s *SQLCommon) GetTokenPool(ctx context.Context, ns string, name string) (message *core.TokenPool, err error) {
	return s.getTokenPoolPred(ctx, ns+":"+name, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"name": name}})
}

func (s *SQLCommon) GetTokenPoolByID(ctx context.Context, id *fftypes.UUID) (message *core.TokenPool, err error) {
	return s.getTokenPoolPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetTokenPoolByLocator(ctx context.Context, connector, locator string) (*core.TokenPool, error) {
	return s.getTokenPoolPred(ctx, locator, sq.And{
		sq.Eq{"connector": connector},
		sq.Eq{"locator": locator},
	})
}

func (s *SQLCommon) GetTokenPools(ctx context.Context, filter database.Filter) (message []*core.TokenPool, fr *database.FilterResult, err error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(tokenPoolColumns...).From("tokenpool"), filter, tokenPoolFilterFieldMap, []interface{}{"seq"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, tokenpoolTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	pools := []*core.TokenPool{}
	for rows.Next() {
		d, err := s.tokenPoolResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		pools = append(pools, d)
	}

	return pools, s.queryRes(ctx, tokenpoolTable, tx, fop, fi), err
}
