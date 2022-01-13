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
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	ffiColumns = []string{
		"id",
		"namespace",
		"name",
		"version",
		"description",
		"message_id",
	}
	ffiFilterFieldMap = map[string]string{
		"message": "message_id",
	}
)

func (s *SQLCommon) UpsertFFI(ctx context.Context, ffi *fftypes.FFI) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("ffi").
			Where(sq.And{sq.Eq{"id": ffi.ID}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if _, err = s.updateTx(ctx, tx,
			sq.Update("ffi").
				Set("namespace", ffi.Namespace).
				Set("name", ffi.Name).
				Set("version", ffi.Version).
				Set("description", ffi.Description).
				Set("message_id", ffi.Message),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionFFIs, fftypes.ChangeEventTypeUpdated, ffi.Namespace, ffi.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("ffi").
				Columns(ffiColumns...).
				Values(
					ffi.ID,
					ffi.Namespace,
					ffi.Name,
					ffi.Version,
					ffi.Description,
					ffi.Message,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionFFIs, fftypes.ChangeEventTypeCreated, ffi.Namespace, ffi.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) ffiResult(ctx context.Context, row *sql.Rows) (*fftypes.FFI, error) {
	ffi := fftypes.FFI{}
	err := row.Scan(
		&ffi.ID,
		&ffi.Namespace,
		&ffi.Name,
		&ffi.Version,
		&ffi.Description,
		&ffi.Message,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "ffi")
	}
	return &ffi, nil
}

func (s *SQLCommon) getFFIPred(ctx context.Context, desc string, pred interface{}) (*fftypes.FFI, error) {
	rows, _, err := s.query(ctx,
		sq.Select(ffiColumns...).
			From("ffi").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("FFI '%s' not found", desc)
		return nil, nil
	}

	ffi, err := s.ffiResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return ffi, nil
}

func (s *SQLCommon) GetFFIs(ctx context.Context, ns string, filter database.Filter) (ffis []*fftypes.FFI, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(ffiColumns...).From("ffi").Where(sq.Eq{"namespace": ns}), filter, ffiFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	ffis = []*fftypes.FFI{}
	for rows.Next() {
		cd, err := s.ffiResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		ffis = append(ffis, cd)
	}

	return ffis, s.queryRes(ctx, tx, "ffi", fop, fi), err

}

func (s *SQLCommon) GetFFIByID(ctx context.Context, id *fftypes.UUID) (*fftypes.FFI, error) {
	return s.getFFIPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetFFI(ctx context.Context, ns, name, version string) (*fftypes.FFI, error) {
	return s.getFFIPred(ctx, ns+":"+name+":"+version, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"name": name}, sq.Eq{"version": version}})
}
