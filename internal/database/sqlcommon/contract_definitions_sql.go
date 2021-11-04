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
	"bytes"
	"context"
	"database/sql"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	contractDefinitionsColumns = []string{
		"id",
		"namespace",
		"name",
		"version",
		"ffabi",
		"onchain_location",
	}
	contractDefinitionsFilterFieldMap = map[string]string{}
)

func (s *SQLCommon) InsertContractDefinition(ctx context.Context, cd *fftypes.ContractDefinition) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("contract_definitions").
			Where(sq.And{sq.Eq{"namespace": cd.Namespace}, sq.Eq{"name": cd.Name}, sq.Eq{"version": cd.Version}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()

	if existing {
		return i18n.NewError(ctx, i18n.MsgContractDefinitionExists, cd.Namespace, cd.Name, cd.Version)
	}
	rows.Close()

	ffabiBytes := new(bytes.Buffer)
	err = json.NewEncoder(ffabiBytes).Encode(cd.FFABI)
	if err != nil {
		return err
	}

	if existing {
		if err = s.updateTx(ctx, tx,
			sq.Update("contract_definitions").
				Set("namespace", cd.Namespace).
				Set("name", cd.Name).
				Set("version", cd.Version).
				Set("ffabi", ffabiBytes.Bytes()).
				Set("onchain_location", cd.OnChainLocation),
			func() {
				s.callbacks.UUIDCollectionNSEvent("contract_definitions", fftypes.ChangeEventTypeUpdated, cd.Namespace, cd.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("contract_definitions").
				Columns(contractDefinitionsColumns...).
				Values(
					cd.ID,
					cd.Namespace,
					cd.Name,
					cd.Version,
					ffabiBytes.Bytes(),
					cd.OnChainLocation,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContracts, fftypes.ChangeEventTypeCreated, cd.Namespace, cd.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractDefinitionResult(ctx context.Context, row *sql.Rows) (*fftypes.ContractDefinition, error) {
	var ffabiBytes []byte
	cd := fftypes.ContractDefinition{}
	err := row.Scan(
		&cd.ID,
		&cd.Namespace,
		&cd.Name,
		&cd.Version,
		&ffabiBytes,
		&cd.OnChainLocation,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contract")
	}
	err = json.Unmarshal(ffabiBytes, &cd.FFABI)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contract")
	}
	return &cd, nil
}

func (s *SQLCommon) getContractPred(ctx context.Context, desc string, pred interface{}) (*fftypes.ContractDefinition, error) {
	rows, _, err := s.query(ctx,
		sq.Select(contractDefinitionsColumns...).
			From("contract_definitions").
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

	cd, err := s.contractDefinitionResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return cd, nil
}

func (s *SQLCommon) GetContractDefinitions(ctx context.Context, ns string, filter database.Filter) (contractDefinitions []*fftypes.ContractDefinition, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(contractDefinitionsColumns...).From("contract_definitions").Where(sq.Eq{"namespace": ns}), filter, contractDefinitionsFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	cds := []*fftypes.ContractDefinition{}
	for rows.Next() {
		cd, err := s.contractDefinitionResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		cds = append(cds, cd)
	}

	return cds, s.queryRes(ctx, tx, "contract_definitions", fop, fi), err

}

func (s *SQLCommon) GetContractDefinitionByID(ctx context.Context, id string) (*fftypes.ContractDefinition, error) {
	return s.getContractPred(ctx, id, sq.Eq{"id": id})
}

func (s *SQLCommon) GetContractDefinitionByNameAndVersion(ctx context.Context, ns, name, version string) (*fftypes.ContractDefinition, error) {
	return s.getContractPred(ctx, ns+":"+name+":"+version, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"name": name}, sq.Eq{"version": version}})
}
