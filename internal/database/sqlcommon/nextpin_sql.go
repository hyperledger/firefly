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
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var (
	nextpinColumns = []string{
		"context",
		"identity",
		"hash",
		"nonce",
	}
	nextpinFilterFieldMap = map[string]string{}
)

func (s *SQLCommon) InsertNextPin(ctx context.Context, nextpin *fftypes.NextPin) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	sequence, err := s.insertTx(ctx, tx,
		sq.Insert("nextpins").
			Columns(nextpinColumns...).
			Values(
				nextpin.Context,
				nextpin.Identity,
				nextpin.Hash,
				nextpin.Nonce,
			),
	)
	if err != nil {
		return err
	}
	nextpin.Sequence = sequence

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) nextpinResult(ctx context.Context, row *sql.Rows) (*fftypes.NextPin, error) {
	nextpin := fftypes.NextPin{}
	err := row.Scan(
		&nextpin.Context,
		&nextpin.Identity,
		&nextpin.Hash,
		&nextpin.Nonce,
		&nextpin.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "nextpins")
	}
	return &nextpin, nil
}

func (s *SQLCommon) getNextPinPred(ctx context.Context, desc string, pred interface{}) (message *fftypes.NextPin, err error) {
	cols := append([]string{}, nextpinColumns...)
	cols = append(cols, "seq")
	rows, err := s.query(ctx,
		sq.Select(cols...).
			From("nextpins").
			Where(pred).
			Limit(1),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("NextPin '%s' not found", desc)
		return nil, nil
	}

	nextpin, err := s.nextpinResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return nextpin, nil
}

func (s *SQLCommon) GetNextPinByContextAndIdentity(ctx context.Context, context *fftypes.Bytes32, identity string) (message *fftypes.NextPin, err error) {
	return s.getNextPinPred(ctx, fmt.Sprintf("%s:%s", context, identity), sq.Eq{
		"context":  context,
		"identity": identity,
	})
}

func (s *SQLCommon) GetNextPinByHash(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.NextPin, err error) {
	return s.getNextPinPred(ctx, hash.String(), sq.Eq{
		"hash": hash,
	})
}

func (s *SQLCommon) GetNextPins(ctx context.Context, filter database.Filter) (message []*fftypes.NextPin, err error) {

	cols := append([]string{}, nextpinColumns...)
	cols = append(cols, "seq")
	query, err := s.filterSelect(ctx, "", sq.Select(cols...).From("nextpins"), filter, nextpinFilterFieldMap, []string{"sequence"})
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nextpin := []*fftypes.NextPin{}
	for rows.Next() {
		d, err := s.nextpinResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		nextpin = append(nextpin, d)
	}

	return nextpin, err

}

func (s *SQLCommon) UpdateNextPin(ctx context.Context, sequence int64, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("nextpins"), update, nodeFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"seq": sequence})

	err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) DeleteNextPin(ctx context.Context, sequence int64) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("nextpins").Where(sq.Eq{
		"seq": sequence,
	}))
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
