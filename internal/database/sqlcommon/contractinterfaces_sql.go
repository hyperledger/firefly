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
	contractInterfacesColumns = []string{
		"id",
		"namespace",
		"name",
		"version",
	}
	contractInterfacesFilterFieldMap = map[string]string{}
)

func (s *SQLCommon) InsertContractInterface(ctx context.Context, cd *fftypes.FFI) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("contractinterfaces").
			Where(sq.And{sq.Eq{"namespace": cd.Namespace}, sq.Eq{"name": cd.Name}, sq.Eq{"version": cd.Version}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if _, err = s.updateTx(ctx, tx,
			sq.Update("contractinterfaces").
				Set("namespace", cd.Namespace).
				Set("name", cd.Name).
				Set("version", cd.Version),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractInterfaces, fftypes.ChangeEventTypeUpdated, cd.Namespace, cd.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("contractinterfaces").
				Columns(contractInterfacesColumns...).
				Values(
					cd.ID,
					cd.Namespace,
					cd.Name,
					cd.Version,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractInterfaces, fftypes.ChangeEventTypeCreated, cd.Namespace, cd.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractInterfaceResult(ctx context.Context, row *sql.Rows) (*fftypes.FFI, error) {
	ffi := fftypes.FFI{}
	err := row.Scan(
		&ffi.ID,
		&ffi.Namespace,
		&ffi.Name,
		&ffi.Version,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contract")
	}
	return &ffi, nil
}

func (s *SQLCommon) getContractPred(ctx context.Context, desc string, pred interface{}) (*fftypes.FFI, error) {
	rows, _, err := s.query(ctx,
		sq.Select(contractInterfacesColumns...).
			From("contractinterfaces").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Contract definition '%s' not found", desc)
		return nil, nil
	}

	ffi, err := s.contractInterfaceResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return ffi, nil
}

func (s *SQLCommon) GetContractInterfaces(ctx context.Context, ns string, filter database.Filter) (contractInterfaces []*fftypes.FFI, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(contractInterfacesColumns...).From("contractinterfaces").Where(sq.Eq{"namespace": ns}), filter, contractInterfacesFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	ffis := []*fftypes.FFI{}
	for rows.Next() {
		cd, err := s.contractInterfaceResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		ffis = append(ffis, cd)
	}

	return ffis, s.queryRes(ctx, tx, "contract_interfaces", fop, fi), err

}

func (s *SQLCommon) GetContractInterfaceByID(ctx context.Context, id string) (*fftypes.FFI, error) {
	return s.getContractPred(ctx, id, sq.Eq{"id": id})
}

func (s *SQLCommon) GetContractInterfaceByNameAndVersion(ctx context.Context, ns, name, version string) (*fftypes.FFI, error) {
	return s.getContractPred(ctx, ns+":"+name+":"+version, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"name": name}, sq.Eq{"version": version}})
}
