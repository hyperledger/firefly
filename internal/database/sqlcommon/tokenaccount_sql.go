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
	tokenAccountColumns = []string{
		"protocol_id",
		"token_index",
		"identity",
		"balance",
	}
	tokenAccountFilterFieldMap = map[string]string{
		"protocolid": "protocol_id",
		"tokenindex": "token_index",
	}
)

func (s *SQLCommon) UpsertTokenAccount(ctx context.Context, account *fftypes.TokenAccount) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("seq").
			From("tokenaccount").
			Where(sq.And{
				sq.Eq{"protocol_id": account.ProtocolID},
				sq.Eq{"token_index": account.TokenIndex},
				sq.Eq{"identity": account.Identity},
			}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if err = s.updateTx(ctx, tx,
			sq.Update("tokenaccount").
				Set("balance", account.Balance).
				Where(sq.And{
					sq.Eq{"protocol_id": account.ProtocolID},
					sq.Eq{"token_index": account.TokenIndex},
					sq.Eq{"identity": account.Identity},
				}),
			nil,
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("tokenaccount").
				Columns(tokenAccountColumns...).
				Values(
					account.ProtocolID,
					account.TokenIndex,
					account.Identity,
					account.Balance,
				),
			nil,
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) tokenAccountResult(ctx context.Context, row *sql.Rows) (*fftypes.TokenAccount, error) {
	account := fftypes.TokenAccount{}
	err := row.Scan(
		&account.ProtocolID,
		&account.TokenIndex,
		&account.Identity,
		&account.Balance,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "tokenaccount")
	}
	return &account, nil
}

func (s *SQLCommon) getTokenAccountPred(ctx context.Context, desc string, pred interface{}) (*fftypes.TokenAccount, error) {
	rows, _, err := s.query(ctx,
		sq.Select(tokenAccountColumns...).
			From("tokenaccount").
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

	account, err := s.tokenAccountResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *SQLCommon) GetTokenAccount(ctx context.Context, protocolID, tokenIndex, identity string) (message *fftypes.TokenAccount, err error) {
	desc := fftypes.TokenAccountIdentifier(protocolID, tokenIndex, identity)
	return s.getTokenAccountPred(ctx, desc, sq.And{
		sq.Eq{"protocol_id": protocolID},
		sq.Eq{"token_index": tokenIndex},
		sq.Eq{"identity": identity},
	})
}

func (s *SQLCommon) GetTokenAccounts(ctx context.Context, filter database.Filter) ([]*fftypes.TokenAccount, *database.FilterResult, error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(tokenAccountColumns...).From("tokenaccount"), filter, tokenAccountFilterFieldMap, []string{"seq"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	accounts := []*fftypes.TokenAccount{}
	for rows.Next() {
		d, err := s.tokenAccountResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		accounts = append(accounts, d)
	}

	return accounts, s.queryRes(ctx, tx, "tokenaccount", fop, fi), err
}
