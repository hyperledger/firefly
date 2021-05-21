// Copyright © 2021 Kaleido, Inc.
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
	dataDefColumns = []string{
		"id",
		"validator",
		"namespace",
		"name",
		"version",
		"hash",
		"created",
		"value",
	}
	dataDefFilterTypeMap = map[string]string{}
)

func (s *SQLCommon) UpsertDataDefinition(ctx context.Context, dataDef *fftypes.DataDefinition, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		dataDefRows, err := s.queryTx(ctx, tx,
			sq.Select("id").
				From("datadefs").
				Where(sq.Eq{"id": dataDef.ID}),
		)
		if err != nil {
			return err
		}
		existing = dataDefRows.Next()
		dataDefRows.Close()
	}

	if existing {

		// Update the dataDef
		if _, err = s.updateTx(ctx, tx,
			sq.Update("datadefs").
				Set("validator", string(dataDef.Validator)).
				Set("namespace", dataDef.Namespace).
				Set("name", dataDef.Name).
				Set("version", dataDef.Version).
				Set("hash", dataDef.Hash).
				Set("created", dataDef.Created).
				Set("value", dataDef.Value).
				Where(sq.Eq{"id": dataDef.ID}),
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("datadefs").
				Columns(dataDefColumns...).
				Values(
					dataDef.ID,
					string(dataDef.Validator),
					dataDef.Namespace,
					dataDef.Name,
					dataDef.Version,
					dataDef.Hash,
					dataDef.Created,
					dataDef.Value,
				),
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) dataDefResult(ctx context.Context, row *sql.Rows) (*fftypes.DataDefinition, error) {
	var dataDef fftypes.DataDefinition
	err := row.Scan(
		&dataDef.ID,
		&dataDef.Validator,
		&dataDef.Namespace,
		&dataDef.Name,
		&dataDef.Version,
		&dataDef.Hash,
		&dataDef.Created,
		&dataDef.Value,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "datadefs")
	}
	return &dataDef, nil
}

func (s *SQLCommon) getDataDefinitionEq(ctx context.Context, eq sq.Eq, textName string) (message *fftypes.DataDefinition, err error) {

	rows, err := s.query(ctx,
		sq.Select(dataDefColumns...).
			From("datadefs").
			Where(eq),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("DataDefinition '%s' not found", textName)
		return nil, nil
	}

	dataDef, err := s.dataDefResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return dataDef, nil
}

func (s *SQLCommon) GetDataDefinitionById(ctx context.Context, id *fftypes.UUID) (message *fftypes.DataDefinition, err error) {
	return s.getDataDefinitionEq(ctx, sq.Eq{"id": id}, id.String())
}

func (s *SQLCommon) GetDataDefinitionByName(ctx context.Context, ns, name string) (message *fftypes.DataDefinition, err error) {
	return s.getDataDefinitionEq(ctx, sq.Eq{"namespace": ns, "name": name}, fmt.Sprintf("%s:%s", ns, name))
}

func (s *SQLCommon) GetDataDefinitions(ctx context.Context, filter database.Filter) (message []*fftypes.DataDefinition, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(dataDefColumns...).From("datadefs"), filter, dataDefFilterTypeMap)
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dataDefs := []*fftypes.DataDefinition{}
	for rows.Next() {
		dataDef, err := s.dataDefResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		dataDefs = append(dataDefs, dataDef)
	}

	return dataDefs, err

}

func (s *SQLCommon) UpdateDataDefinition(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(ctx, "", sq.Update("datadefs"), update, dataDefFilterTypeMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
