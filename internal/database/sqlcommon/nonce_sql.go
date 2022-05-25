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
	nonceColumns = []string{
		"hash",
		"nonce",
	}
	nonceFilterFieldMap = map[string]string{}
)

const noncesTable = "nonces"

func (s *SQLCommon) UpdateNonce(ctx context.Context, nonce *core.Nonce) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	// Update the nonce
	if _, err = s.updateTx(ctx, noncesTable, tx,
		sq.Update(noncesTable).
			Set("nonce", nonce.Nonce).
			Where(sq.Eq{"hash": nonce.Hash}),
		nil, // no change events for nonces
	); err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) InsertNonce(ctx context.Context, nonce *core.Nonce) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	// Insert the nonce
	if _, err = s.insertTx(ctx, noncesTable, tx,
		sq.Insert(noncesTable).
			Columns(nonceColumns...).
			Values(
				nonce.Hash,
				nonce.Nonce,
			),
		nil, // no change events for nonces
	); err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) nonceResult(ctx context.Context, row *sql.Rows) (*core.Nonce, error) {
	nonce := core.Nonce{}
	err := row.Scan(
		&nonce.Hash,
		&nonce.Nonce,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, noncesTable)
	}
	return &nonce, nil
}

func (s *SQLCommon) GetNonce(ctx context.Context, hash *fftypes.Bytes32) (message *core.Nonce, err error) {

	rows, _, err := s.query(ctx, noncesTable,
		sq.Select(nonceColumns...).
			From(noncesTable).
			Where(sq.Eq{"hash": hash}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Nonce '%s' not found", hash)
		return nil, nil
	}

	nonce, err := s.nonceResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return nonce, nil
}

func (s *SQLCommon) GetNonces(ctx context.Context, filter database.Filter) (message []*core.Nonce, fr *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(nonceColumns...).From(noncesTable), filter, nonceFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, noncesTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	nonce := []*core.Nonce{}
	for rows.Next() {
		d, err := s.nonceResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		nonce = append(nonce, d)
	}

	return nonce, s.queryRes(ctx, noncesTable, tx, fop, fi), err

}

func (s *SQLCommon) DeleteNonce(ctx context.Context, hash *fftypes.Bytes32) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, noncesTable, tx, sq.Delete(noncesTable).Where(sq.Eq{
		"hash": hash,
	}), nil /* no change events for nonces */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
