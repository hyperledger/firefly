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
	nonceColumns = []string{
		"context",
		"nonce",
		"group_hash",
		"topic",
	}
	nonceFilterFieldMap = map[string]string{
		"group": "group_hash",
	}
)

func (s *SQLCommon) UpsertNonceNext(ctx context.Context, nonce *fftypes.Nonce) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	// Do a select within the transaction to detemine if the UUID already exists
	nonceRows, err := s.queryTx(ctx, tx,
		sq.Select("nonce", sequenceColumn).
			From("nonces").
			Where(
				sq.Eq{"context": nonce.Context}),
	)
	if err != nil {
		return err
	}
	existing = nonceRows.Next()

	if existing {
		var existingNonce, sequence int64
		err := nonceRows.Scan(&existingNonce, &sequence)
		if err != nil {
			nonceRows.Close()
			return i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "nonces")
		}
		nonce.Nonce = existingNonce + 1
		nonceRows.Close()

		// Update the nonce
		if err = s.updateTx(ctx, tx,
			sq.Update("nonces").
				Set("nonce", nonce.Nonce).
				Where(sq.Eq{sequenceColumn: sequence}),
			nil, // no change events for nonces
		); err != nil {
			return err
		}
	} else {
		nonceRows.Close()

		nonce.Nonce = 0
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("nonces").
				Columns(nonceColumns...).
				Values(
					nonce.Context,
					nonce.Nonce,
					nonce.Group,
					nonce.Topic,
				),
			nil, // no change events for nonces
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) nonceResult(ctx context.Context, row *sql.Rows) (*fftypes.Nonce, error) {
	nonce := fftypes.Nonce{}
	err := row.Scan(
		&nonce.Context,
		&nonce.Nonce,
		&nonce.Group,
		&nonce.Topic,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "nonces")
	}
	return &nonce, nil
}

func (s *SQLCommon) GetNonce(ctx context.Context, context *fftypes.Bytes32) (message *fftypes.Nonce, err error) {

	rows, err := s.query(ctx,
		sq.Select(nonceColumns...).
			From("nonces").
			Where(sq.Eq{"context": context}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Nonce '%s' not found", context)
		return nil, nil
	}

	nonce, err := s.nonceResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return nonce, nil
}

func (s *SQLCommon) GetNonces(ctx context.Context, filter database.Filter) (message []*fftypes.Nonce, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(nonceColumns...).From("nonces"), filter, nonceFilterFieldMap, []string{"sequence"})
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nonce := []*fftypes.Nonce{}
	for rows.Next() {
		d, err := s.nonceResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		nonce = append(nonce, d)
	}

	return nonce, err

}

func (s *SQLCommon) DeleteNonce(ctx context.Context, context *fftypes.Bytes32) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("nonces").Where(sq.Eq{
		"context": context,
	}), nil /* no change events for nonces */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
