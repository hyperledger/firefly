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
	tokenBalanceColumns = []string{
		"pool_protocol_id",
		"token_index",
		"connector",
		"namespace",
		"key",
		"balance",
		"updated",
	}
	tokenBalanceFilterFieldMap = map[string]string{
		"poolprotocolid": "pool_protocol_id",
		"tokenindex":     "token_index",
	}
)

func (s *SQLCommon) addTokenBalance(ctx context.Context, tx *txWrapper, transfer *fftypes.TokenTransfer, key string, negate bool) error {
	account, err := s.GetTokenBalance(ctx, transfer.PoolProtocolID, transfer.TokenIndex, key)
	if err != nil {
		return err
	}

	var balance *fftypes.BigInt
	if account != nil {
		balance = &account.Balance
	} else {
		balance = &fftypes.BigInt{}
	}
	if negate {
		balance.Int().Sub(balance.Int(), transfer.Amount.Int())
	} else {
		balance.Int().Add(balance.Int(), transfer.Amount.Int())
	}

	if account != nil {
		if err = s.updateTx(ctx, tx,
			sq.Update("tokenbalance").
				Set("balance", balance).
				Set("updated", fftypes.Now()).
				Where(sq.And{
					sq.Eq{"pool_protocol_id": account.PoolProtocolID},
					sq.Eq{"token_index": account.TokenIndex},
					sq.Eq{"key": account.Key},
				}),
			nil,
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("tokenbalance").
				Columns(tokenBalanceColumns...).
				Values(
					transfer.PoolProtocolID,
					transfer.TokenIndex,
					transfer.Connector,
					transfer.Namespace,
					key,
					balance,
					fftypes.Now(),
				),
			nil,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLCommon) UpdateTokenBalances(ctx context.Context, transfer *fftypes.TokenTransfer) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	if transfer.From != "" {
		if err := s.addTokenBalance(ctx, tx, transfer, transfer.From, true); err != nil {
			return err
		}
	}
	if transfer.To != "" {
		if err := s.addTokenBalance(ctx, tx, transfer, transfer.To, false); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) tokenBalanceResult(ctx context.Context, row *sql.Rows) (*fftypes.TokenBalance, error) {
	account := fftypes.TokenBalance{}
	err := row.Scan(
		&account.PoolProtocolID,
		&account.TokenIndex,
		&account.Connector,
		&account.Namespace,
		&account.Key,
		&account.Balance,
		&account.Updated,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "tokenbalance")
	}
	return &account, nil
}

func (s *SQLCommon) getTokenBalancePred(ctx context.Context, desc string, pred interface{}) (*fftypes.TokenBalance, error) {
	rows, _, err := s.query(ctx,
		sq.Select(tokenBalanceColumns...).
			From("tokenbalance").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Token account '%s' not found", desc)
		return nil, nil
	}

	account, err := s.tokenBalanceResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *SQLCommon) GetTokenBalance(ctx context.Context, protocolID, tokenIndex, key string) (message *fftypes.TokenBalance, err error) {
	desc := fftypes.TokenBalanceIdentifier(protocolID, tokenIndex, key)
	return s.getTokenBalancePred(ctx, desc, sq.And{
		sq.Eq{"pool_protocol_id": protocolID},
		sq.Eq{"token_index": tokenIndex},
		sq.Eq{"key": key},
	})
}

func (s *SQLCommon) GetTokenBalances(ctx context.Context, filter database.Filter) ([]*fftypes.TokenBalance, *database.FilterResult, error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(tokenBalanceColumns...).From("tokenbalance"), filter, tokenBalanceFilterFieldMap, []interface{}{"seq"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	accounts := []*fftypes.TokenBalance{}
	for rows.Next() {
		d, err := s.tokenBalanceResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		accounts = append(accounts, d)
	}

	return accounts, s.queryRes(ctx, tx, "tokenbalance", fop, fi), err
}

func (s *SQLCommon) GetTokenAccounts(ctx context.Context, filter database.Filter) ([]*fftypes.TokenAccount, *database.FilterResult, error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select("key").Distinct().From("tokenbalance"), filter, tokenBalanceFilterFieldMap, []interface{}{"seq"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var accounts []*fftypes.TokenAccount
	for rows.Next() {
		var account fftypes.TokenAccount
		err := rows.Scan(&account.Key)
		if err != nil {
			return nil, nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "tokenbalance")
		}
		accounts = append(accounts, &account)
	}

	return accounts, s.queryRes(ctx, tx, "tokenbalance", fop, fi), err
}
