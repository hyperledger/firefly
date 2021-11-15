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
	contractParamsColumns = []string{
		"contract_id",
		"namespace",
		"name",
	}
	contractParamsFilterFieldMap = map[string]string{}
)

func (s *SQLCommon) InsertContractParam(ctx context.Context, ns string, contractID *fftypes.UUID, parent_name, role string, param *fftypes.FFIParam) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("contract_methods").
			Where(sq.And{sq.Eq{"contract_id": contractID}, sq.Eq{"namespace": ns}, sq.Eq{"name": param.Name}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if err = s.updateTx(ctx, tx,
			sq.Update("contract_methods").
				Set("contract_id", contractID).
				Set("namespace", ns).
				Set("name", param.Name),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractInterfaces, fftypes.ChangeEventTypeUpdated, ns, contractID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("contract_methods").
				Columns(contractMethodsColumns...).
				Values(
					contractID,
					ns,
					param.Name,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractInterfaces, fftypes.ChangeEventTypeCreated, ns, contractID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractParamResult(ctx context.Context, row *sql.Rows) (*fftypes.FFIParam, error) {
	param := fftypes.FFIParam{}
	err := row.Scan(
		nil,
		nil,
		&param.Name,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contract_params")
	}
	return &param, nil
}

func (s *SQLCommon) getContractParamPred(ctx context.Context, desc string, pred interface{}) (*fftypes.FFIParam, error) {
	rows, _, err := s.query(ctx,
		sq.Select(contractParamsColumns...).
			From("contract_params").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Contract param '%s' not found", desc)
		return nil, nil
	}

	ci, err := s.contractParamResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return ci, nil
}

func (s *SQLCommon) GetContractParams(ctx context.Context, ns string, filter database.Filter) (methods []*fftypes.FFIMethod, res *database.FilterResult, err error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(contractMethodsColumns...).From("contract_methods").Where(sq.Eq{"namespace": ns}), filter, contractMethodsFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ci, err := s.contractMethodResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		methods = append(methods, ci)
	}

	return methods, s.queryRes(ctx, tx, "contract_methods", fop, fi), err

}

func (s *SQLCommon) GetContractParamByName(ctx context.Context, ns, contractID, name string) (*fftypes.FFIParam, error) {
	return s.getContractParamPred(ctx, ns+":"+name, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"contract_id": contractID}, sq.Eq{"name": name}})
}
