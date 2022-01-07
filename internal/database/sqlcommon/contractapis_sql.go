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
	contractAPIsColumns = []string{
		"id",
		"interface_id",
		"ledger",
		"location",
		"name",
		"namespace",
		"message_id",
	}
	contractAPIsFilterFieldMap = map[string]string{
		"interface": "interface_id",
		"message":   "message_id",
	}
)

func (s *SQLCommon) UpsertContractAPI(ctx context.Context, api *fftypes.ContractAPI) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("contractapis").
			Where(sq.And{sq.Eq{"id": api.ID}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if _, err = s.updateTx(ctx, tx,
			sq.Update("contractapis").
				Set("id", api.ID).
				Set("interface_id", api.Interface.ID).
				Set("ledger", api.Ledger).
				Set("location", api.Location).
				Set("name", api.Name).
				Set("namespace", api.Namespace).
				Set("message_id", api.Message),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractAPIs, fftypes.ChangeEventTypeUpdated, api.Namespace, api.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("contractapis").
				Columns(contractAPIsColumns...).
				Values(
					api.ID,
					api.Interface.ID,
					api.Ledger,
					api.Location,
					api.Name,
					api.Namespace,
					api.Message,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractAPIs, fftypes.ChangeEventTypeCreated, api.Namespace, api.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractAPIResult(ctx context.Context, row *sql.Rows) (*fftypes.ContractAPI, error) {
	api := fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{},
	}
	err := row.Scan(
		&api.ID,
		&api.Interface.ID,
		&api.Ledger,
		&api.Location,
		&api.Name,
		&api.Namespace,
		&api.Message,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contract")
	}
	return &api, nil
}

func (s *SQLCommon) getContractAPIPred(ctx context.Context, desc string, pred interface{}) (*fftypes.ContractAPI, error) {
	rows, _, err := s.query(ctx,
		sq.Select(contractAPIsColumns...).
			From("contractapis").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Contract API '%s' not found", desc)
		return nil, nil
	}

	api, err := s.contractAPIResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return api, nil
}

func (s *SQLCommon) GetContractAPIs(ctx context.Context, ns string, filter database.AndFilter) (contractAPIs []*fftypes.ContractAPI, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(contractAPIsColumns...).From("contractapis").Where(sq.Eq{"namespace": ns}), filter, contractAPIsFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	apis := []*fftypes.ContractAPI{}
	for rows.Next() {
		api, err := s.contractAPIResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		apis = append(apis, api)
	}

	return apis, s.queryRes(ctx, tx, "contract_interfaces", fop, fi), err

}

func (s *SQLCommon) GetContractAPIByID(ctx context.Context, id *fftypes.UUID) (*fftypes.ContractAPI, error) {
	return s.getContractAPIPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetContractAPIByName(ctx context.Context, ns, name string) (*fftypes.ContractAPI, error) {
	return s.getContractAPIPred(ctx, ns+":"+name, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"name": name}})
}
