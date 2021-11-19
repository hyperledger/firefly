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
		"interface_id",
		"parent_name",
		"namespace",
		"name",
		"type",
		"param_index",
		"role",
	}
	selectContractParamsColumns = []string{
		"name",
		"type",
	}
	contractParamsFilterFieldMap = map[string]string{}
)

func (s *SQLCommon) InsertContractParam(ctx context.Context, ns string, interfaceID *fftypes.UUID, parentName string, role fftypes.FFIParamRole, index int, param *fftypes.FFIParam) error {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("interface_id").
			From("contractparams").
			Where(sq.And{
				sq.Eq{"namespace": ns},
				sq.Eq{"interface_id": interfaceID},
				sq.Eq{"name": param.Name},
				sq.Eq{"role": role},
			}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if err = s.updateTx(ctx, tx,
			sq.Update("contractparams").
				Set("interface_id", interfaceID).
				Set("parent_name", parentName).
				Set("namespace", ns).
				Set("name", param.Name).
				Set("type", param.Type).
				Set("param_index", index).
				Set("role", role),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractParams, fftypes.ChangeEventTypeUpdated, ns, interfaceID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("contractparams").
				Columns(contractParamsColumns...).
				Values(
					interfaceID,
					parentName,
					ns,
					param.Name,
					param.Type,
					index,
					role,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractInterfaces, fftypes.ChangeEventTypeCreated, ns, interfaceID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractParamResults(ctx context.Context, rows *sql.Rows) ([]*fftypes.FFIParam, error) {
	params := []*fftypes.FFIParam{}

	for {
		param := fftypes.FFIParam{}
		err := rows.Scan(
			&param.Name,
			&param.Type,
		)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contractparams")
		}
		params = append(params, &param)
		if !rows.Next() {
			break
		}
	}
	return params, nil
}

func (s *SQLCommon) getContractParamsPred(ctx context.Context, desc string, pred interface{}) ([]*fftypes.FFIParam, error) {
	rows, _, err := s.query(ctx,
		sq.Select(selectContractParamsColumns...).
			From("contractparams").
			Where(pred).
			OrderBy("param_index"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Contract param '%s' not found", desc)
		return nil, nil
	}

	params, err := s.contractParamResults(ctx, rows)
	if err != nil {
		return nil, err
	}

	return params, nil
}

func (s *SQLCommon) GetContractParams(ctx context.Context, ns string, filter database.Filter) (methods []*fftypes.FFIMethod, res *database.FilterResult, err error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(contractParamsColumns...).From("contract_methods").Where(sq.Eq{"namespace": ns}), filter, contractParamsFilterFieldMap, []interface{}{"sequence"})
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

func (s *SQLCommon) GetContractParamsByMethodName(ctx context.Context, ns, contractID, methodName string) (params, returns []*fftypes.FFIParam, err error) {
	params, err = s.getContractParamsPred(ctx, ns+":"+contractID+":"+methodName, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"interface_id": contractID}, sq.Eq{"parent_name": methodName}, sq.Eq{"role": "param"}})
	if err != nil {
		return nil, nil, err
	}
	returns, err = s.getContractParamsPred(ctx, ns+":"+contractID+":"+methodName, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"interface_id": contractID}, sq.Eq{"parent_name": methodName}, sq.Eq{"role": "return"}})
	if err != nil {
		return nil, nil, err
	}
	return
}
