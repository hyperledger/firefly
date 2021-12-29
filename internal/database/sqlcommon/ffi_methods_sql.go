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
	ffiMethodsColumns = []string{
		"id",
		"interface_id",
		"namespace",
		"name",
		"pathname",
		"description",
		"params",
		"returns",
	}
	ffiMethodFilterFieldMap = map[string]string{
		"interface": "interface_id",
	}
)

func (s *SQLCommon) UpsertFFIMethod(ctx context.Context, method *fftypes.FFIMethod) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("ffimethods").
			Where(sq.And{sq.Eq{"interface_id": method.Contract}, sq.Eq{"namespace": method.Namespace}, sq.Eq{"pathname": method.Pathname}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if _, err = s.updateTx(ctx, tx,
			sq.Update("ffimethods").
				Set("params", method.Params).
				Set("returns", method.Returns).
				Where(sq.And{sq.Eq{"interface_id": method.Contract}, sq.Eq{"namespace": method.Namespace}, sq.Eq{"pathname": method.Pathname}}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionFFIMethods, fftypes.ChangeEventTypeUpdated, method.Namespace, method.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("ffimethods").
				Columns(ffiMethodsColumns...).
				Values(
					method.ID,
					method.Contract,
					method.Namespace,
					method.Name,
					method.Pathname,
					method.Description,
					method.Params,
					method.Returns,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionFFIMethods, fftypes.ChangeEventTypeCreated, method.Namespace, method.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) ffiMethodResult(ctx context.Context, row *sql.Rows) (*fftypes.FFIMethod, error) {
	method := fftypes.FFIMethod{}
	err := row.Scan(
		&method.ID,
		&method.Contract,
		&method.Namespace,
		&method.Name,
		&method.Pathname,
		&method.Description,
		&method.Params,
		&method.Returns,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "ffimethods")
	}
	return &method, nil
}

func (s *SQLCommon) getFFIMethodPred(ctx context.Context, desc string, pred interface{}) (*fftypes.FFIMethod, error) {
	rows, _, err := s.query(ctx,
		sq.Select(ffiMethodsColumns...).
			From("ffimethods").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("FFI method '%s' not found", desc)
		return nil, nil
	}

	ci, err := s.ffiMethodResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return ci, nil
}

func (s *SQLCommon) GetFFIMethods(ctx context.Context, filter database.Filter) (methods []*fftypes.FFIMethod, res *database.FilterResult, err error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(ffiMethodsColumns...).From("ffimethods"), filter, ffiMethodFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ci, err := s.ffiMethodResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		methods = append(methods, ci)
	}

	return methods, s.queryRes(ctx, tx, "ffimethods", fop, fi), err

}

func (s *SQLCommon) GetFFIMethod(ctx context.Context, ns string, contractID *fftypes.UUID, pathName string) (*fftypes.FFIMethod, error) {
	return s.getFFIMethodPred(ctx, ns+":"+pathName, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"interface_id": contractID}, sq.Eq{"pathname": pathName}})
}
