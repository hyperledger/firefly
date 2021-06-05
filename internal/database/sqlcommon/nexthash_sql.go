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
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

var (
	nexthashColumns = []string{
		"context",
		"identity",
		"hash",
		"nonce",
	}
	nexthashFilterTypeMap = map[string]string{}
)

func (s *SQLCommon) InsertNextHash(ctx context.Context, nexthash *fftypes.NextHash) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	sequence, err := s.insertTx(ctx, tx,
		sq.Insert("nexthashes").
			Columns(nexthashColumns...).
			Values(
				nexthash.Context,
				nexthash.Identity,
				nexthash.Hash,
				nexthash.Nonce,
			),
	)
	if err != nil {
		return err
	}
	nexthash.Sequence = sequence

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) nexthashResult(ctx context.Context, row *sql.Rows) (*fftypes.NextHash, error) {
	nexthash := fftypes.NextHash{}
	err := row.Scan(
		&nexthash.Context,
		&nexthash.Identity,
		&nexthash.Hash,
		&nexthash.Nonce,
		&nexthash.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "nexthashes")
	}
	return &nexthash, nil
}

func (s *SQLCommon) getNextHashPred(ctx context.Context, desc string, pred interface{}) (message *fftypes.NextHash, err error) {
	cols := append([]string{}, nexthashColumns...)
	cols = append(cols, s.provider.SequenceField(""))
	rows, err := s.query(ctx,
		sq.Select(cols...).
			From("nexthashes").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("NextHash '%s' not found", desc)
		return nil, nil
	}

	nexthash, err := s.nexthashResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return nexthash, nil
}

func (s *SQLCommon) GetNextHashByContextAndIdentity(ctx context.Context, context *fftypes.Bytes32, identity string) (message *fftypes.NextHash, err error) {
	return s.getNextHashPred(ctx, fmt.Sprintf("%s:%s", context, identity), sq.Eq{
		"context":  context,
		"identity": identity,
	})
}

func (s *SQLCommon) GetNextHashByHash(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.NextHash, err error) {
	return s.getNextHashPred(ctx, hash.String(), sq.Eq{
		"hash": hash,
	})
}

func (s *SQLCommon) GetNextHashes(ctx context.Context, filter database.Filter) (message []*fftypes.NextHash, err error) {

	cols := append([]string{}, nexthashColumns...)
	cols = append(cols, s.provider.SequenceField(""))
	query, err := s.filterSelect(ctx, "", sq.Select(cols...).From("nexthashes"), filter, nexthashFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nexthash := []*fftypes.NextHash{}
	for rows.Next() {
		d, err := s.nexthashResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		nexthash = append(nexthash, d)
	}

	return nexthash, err

}

func (s *SQLCommon) UpdateNextHash(ctx context.Context, sequence int64, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("nexthashes"), update, nodeFilterTypeMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{s.provider.SequenceField(""): sequence})

	err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) DeleteNextHash(ctx context.Context, sequence int64) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("nexthashes").Where(sq.Eq{
		s.provider.SequenceField(""): sequence,
	}))
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
