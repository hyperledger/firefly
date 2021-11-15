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
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	tokenTransferColumns = []string{
		"type",
		"local_id",
		"pool_id",
		"token_index",
		"connector",
		"namespace",
		"key",
		"from_key",
		"to_key",
		"amount",
		"protocol_id",
		"message_hash",
		"tx_type",
		"tx_id",
		"created",
	}
	tokenTransferFilterFieldMap = map[string]string{
		"localid":          "local_id",
		"pool":             "pool_id",
		"tokenindex":       "token_index",
		"from":             "from_key",
		"to":               "to_key",
		"protocolid":       "protocol_id",
		"messagehash":      "message_hash",
		"transaction.type": "tx_type",
		"transaction.id":   "tx_id",
	}
)

func (s *SQLCommon) UpsertTokenTransfer(ctx context.Context, transfer *fftypes.TokenTransfer) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("seq").
			From("tokentransfer").
			Where(sq.Eq{"protocol_id": transfer.ProtocolID}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if _, err = s.updateTx(ctx, tx,
			sq.Update("tokentransfer").
				Set("type", transfer.Type).
				Set("local_id", transfer.LocalID).
				Set("pool_id", transfer.Pool).
				Set("token_index", transfer.TokenIndex).
				Set("connector", transfer.Connector).
				Set("namespace", transfer.Namespace).
				Set("key", transfer.Key).
				Set("from_key", transfer.From).
				Set("to_key", transfer.To).
				Set("amount", transfer.Amount).
				Set("message_hash", transfer.MessageHash).
				Set("tx_type", transfer.TX.Type).
				Set("tx_id", transfer.TX.ID).
				Where(sq.Eq{"protocol_id": transfer.ProtocolID}),
			func() {
				s.callbacks.UUIDCollectionEvent(database.CollectionTokenTransfers, fftypes.ChangeEventTypeUpdated, transfer.LocalID)
			},
		); err != nil {
			return err
		}
	} else {
		transfer.Created = fftypes.Now()
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("tokentransfer").
				Columns(tokenTransferColumns...).
				Values(
					transfer.Type,
					transfer.LocalID,
					transfer.Pool,
					transfer.TokenIndex,
					transfer.Connector,
					transfer.Namespace,
					transfer.Key,
					transfer.From,
					transfer.To,
					transfer.Amount,
					transfer.ProtocolID,
					transfer.MessageHash,
					transfer.TX.Type,
					transfer.TX.ID,
					transfer.Created,
				),
			func() {
				s.callbacks.UUIDCollectionEvent(database.CollectionTokenTransfers, fftypes.ChangeEventTypeCreated, transfer.LocalID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) tokenTransferResult(ctx context.Context, row *sql.Rows) (*fftypes.TokenTransfer, error) {
	transfer := fftypes.TokenTransfer{}
	err := row.Scan(
		&transfer.Type,
		&transfer.LocalID,
		&transfer.Pool,
		&transfer.TokenIndex,
		&transfer.Connector,
		&transfer.Namespace,
		&transfer.Key,
		&transfer.From,
		&transfer.To,
		&transfer.Amount,
		&transfer.ProtocolID,
		&transfer.MessageHash,
		&transfer.TX.Type,
		&transfer.TX.ID,
		&transfer.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "tokentransfer")
	}
	return &transfer, nil
}

func (s *SQLCommon) getTokenTransferPred(ctx context.Context, desc string, pred interface{}) (*fftypes.TokenTransfer, error) {
	rows, _, err := s.query(ctx,
		sq.Select(tokenTransferColumns...).
			From("tokentransfer").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Token transfer '%s' not found", desc)
		return nil, nil
	}

	transfer, err := s.tokenTransferResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return transfer, nil
}

func (s *SQLCommon) GetTokenTransfer(ctx context.Context, localID *fftypes.UUID) (*fftypes.TokenTransfer, error) {
	return s.getTokenTransferPred(ctx, localID.String(), sq.Eq{"local_id": localID})
}

func (s *SQLCommon) GetTokenTransfers(ctx context.Context, filter database.Filter) (message []*fftypes.TokenTransfer, fr *database.FilterResult, err error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(tokenTransferColumns...).From("tokentransfer"), filter, tokenTransferFilterFieldMap, []interface{}{"seq"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	transfers := []*fftypes.TokenTransfer{}
	for rows.Next() {
		d, err := s.tokenTransferResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		transfers = append(transfers, d)
	}

	return transfers, s.queryRes(ctx, tx, "tokentransfer", fop, fi), err
}
