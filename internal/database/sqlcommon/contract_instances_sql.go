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
	contractInstancesColumns = []string{
		"id",
		"contract_id",
		"namespace",
		"name",
		"onchain_location",
	}
	contractInstancesFilterFieldMap = map[string]string{}
)

func (s *SQLCommon) InsertContractInstance(ctx context.Context, ci *fftypes.ContractInstance) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("contract_instances").
			Where(sq.And{sq.Eq{"namespace": ci.Namespace}, sq.Eq{"name": ci.Name}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()

	if existing {
		return i18n.NewError(ctx, i18n.MsgContractInstanceExists, ci.Namespace, ci.Name)
	}
	rows.Close()

	if existing {
		if err = s.updateTx(ctx, tx,
			sq.Update("contract_instances").
				Set("namespace", ci.Namespace).
				Set("name", ci.Name).
				Set("contract_id", ci.ContractDefinition.ID).
				Set("onchain_location", ci.OnChainLocation),
			func() {
				s.callbacks.UUIDCollectionNSEvent("contract_instances", fftypes.ChangeEventTypeUpdated, ci.Namespace, ci.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("contract_instances").
				Columns(contractInstancesColumns...).
				Values(
					ci.ID,
					ci.ContractDefinition.ID,
					ci.Namespace,
					ci.Name,
					ci.OnChainLocation,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContracts, fftypes.ChangeEventTypeCreated, ci.Namespace, ci.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractInstanceResult(ctx context.Context, row *sql.Rows) (*fftypes.ContractInstance, error) {
	ci := fftypes.ContractInstance{
		ContractDefinition: &fftypes.ContractDefinition{},
	}
	err := row.Scan(
		&ci.ID,
		&ci.ContractDefinition.ID,
		&ci.Namespace,
		&ci.Name,
		&ci.OnChainLocation,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contract instance")
	}
	return &ci, nil
}

func (s *SQLCommon) getContractInstancePred(ctx context.Context, desc string, pred interface{}) (*fftypes.ContractInstance, error) {
	rows, _, err := s.query(ctx,
		sq.Select(contractInstancesColumns...).
			From("contract_instances").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Contract instance '%s' not found", desc)
		return nil, nil
	}

	ci, err := s.contractInstanceResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return ci, nil
}

func (s *SQLCommon) GetContractInstances(ctx context.Context, ns string, filter database.Filter) (contractInstances []*fftypes.ContractInstance, res *database.FilterResult, err error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(contractInstancesColumns...).From("contract_instances").Where(sq.Eq{"namespace": ns}), filter, contractInstancesFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	ciList := []*fftypes.ContractInstance{}
	for rows.Next() {
		ci, err := s.contractInstanceResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		ciList = append(ciList, ci)
	}

	return ciList, s.queryRes(ctx, tx, "contract_instances", fop, fi), err

}

func (s *SQLCommon) GetContractInstanceByID(ctx context.Context, id string) (*fftypes.ContractInstance, error) {
	return s.getContractInstancePred(ctx, id, sq.Eq{"id": id})
}

func (s *SQLCommon) GetContractInstanceByName(ctx context.Context, ns, name string) (*fftypes.ContractInstance, error) {
	return s.getContractInstancePred(ctx, ns+":"+name, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"name": name}})
}
