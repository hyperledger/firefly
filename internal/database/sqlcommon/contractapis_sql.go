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
	contractAPIsColumns = []string{
		"id",
		"interface_id",
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

const contractapisTable = "contractapis"

func (s *SQLCommon) UpsertContractAPI(ctx context.Context, api *core.ContractAPI) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, contractapisTable, tx,
		sq.Select("id").
			From(contractapisTable).
			Where(sq.And{sq.Eq{"namespace": api.Namespace}, sq.Eq{"name": api.Name}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()

	if existing {
		var id fftypes.UUID
		_ = rows.Scan(&id)
		if api.ID != nil && *api.ID != id {
			rows.Close()
			return database.IDMismatch
		}
		api.ID = &id // Update on returned object
	}
	rows.Close()

	if existing {
		if _, err = s.updateTx(ctx, contractapisTable, tx,
			sq.Update(contractapisTable).
				Set("id", api.ID).
				Set("interface_id", api.Interface.ID).
				Set("location", api.Location).
				Set("name", api.Name).
				Set("namespace", api.Namespace).
				Set("message_id", api.Message),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractAPIs, core.ChangeEventTypeUpdated, api.Namespace, api.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, contractapisTable, tx,
			sq.Insert(contractapisTable).
				Columns(contractAPIsColumns...).
				Values(
					api.ID,
					api.Interface.ID,
					api.Location,
					api.Name,
					api.Namespace,
					api.Message,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractAPIs, core.ChangeEventTypeCreated, api.Namespace, api.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractAPIResult(ctx context.Context, row *sql.Rows) (*core.ContractAPI, error) {
	api := core.ContractAPI{
		Interface: &core.FFIReference{},
	}
	err := row.Scan(
		&api.ID,
		&api.Interface.ID,
		&api.Location,
		&api.Name,
		&api.Namespace,
		&api.Message,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, "contract")
	}
	return &api, nil
}

func (s *SQLCommon) getContractAPIPred(ctx context.Context, desc string, pred interface{}) (*core.ContractAPI, error) {
	rows, _, err := s.query(ctx, contractapisTable,
		sq.Select(contractAPIsColumns...).
			From(contractapisTable).
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

func (s *SQLCommon) GetContractAPIs(ctx context.Context, ns string, filter database.AndFilter) (contractAPIs []*core.ContractAPI, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(contractAPIsColumns...).From(contractapisTable).Where(sq.Eq{"namespace": ns}), filter, contractAPIsFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, contractapisTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	apis := []*core.ContractAPI{}
	for rows.Next() {
		api, err := s.contractAPIResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		apis = append(apis, api)
	}

	return apis, s.queryRes(ctx, contractapisTable, tx, fop, fi), err

}

func (s *SQLCommon) GetContractAPIByID(ctx context.Context, id *fftypes.UUID) (*core.ContractAPI, error) {
	return s.getContractAPIPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetContractAPIByName(ctx context.Context, ns, name string) (*core.ContractAPI, error) {
	return s.getContractAPIPred(ctx, ns+":"+name, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"name": name}})
}
