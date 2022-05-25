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
	dataColumnsNoValue = []string{
		"id",
		"validator",
		"namespace",
		"datatype_name",
		"datatype_version",
		"hash",
		"created",
		"blob_hash",
		"blob_public",
		"blob_name",
		"blob_size",
		"value_size",
	}
	dataColumnsWithValue = append(append([]string{}, dataColumnsNoValue...), "value")
	dataFilterFieldMap   = map[string]string{
		"validator":        "validator",
		"datatype.name":    "datatype_name",
		"datatype.version": "datatype_version",
		"blob.hash":        "blob_hash",
		"blob.public":      "blob_public",
		"blob.name":        "blob_name",
		"blob.size":        "blob_size",
	}
)

const dataTable = "data"

func (s *SQLCommon) attemptDataUpdate(ctx context.Context, tx *txWrapper, data *core.Data) (int64, error) {
	datatype := data.Datatype
	if datatype == nil {
		datatype = &core.DatatypeRef{}
	}
	blob := data.Blob
	if blob == nil {
		blob = &core.BlobRef{}
	}
	return s.updateTx(ctx, dataTable, tx,
		sq.Update(dataTable).
			Set("validator", string(data.Validator)).
			Set("namespace", data.Namespace).
			Set("datatype_name", datatype.Name).
			Set("datatype_version", datatype.Version).
			Set("hash", data.Hash).
			Set("created", data.Created).
			Set("blob_hash", blob.Hash).
			Set("blob_public", blob.Public).
			Set("blob_name", blob.Name).
			Set("blob_size", blob.Size).
			Set("value_size", data.ValueSize).
			Set("value", data.Value).
			Where(sq.Eq{
				"id":   data.ID,
				"hash": data.Hash,
			}),
		func() {
			s.callbacks.UUIDCollectionNSEvent(database.CollectionData, core.ChangeEventTypeUpdated, data.Namespace, data.ID)
		})
}

func (s *SQLCommon) setDataInsertValues(query sq.InsertBuilder, data *core.Data) sq.InsertBuilder {
	datatype := data.Datatype
	if datatype == nil {
		datatype = &core.DatatypeRef{}
	}
	blob := data.Blob
	if blob == nil {
		blob = &core.BlobRef{}
	}
	return query.Values(
		data.ID,
		string(data.Validator),
		data.Namespace,
		datatype.Name,
		datatype.Version,
		data.Hash,
		data.Created,
		blob.Hash,
		blob.Public,
		blob.Name,
		blob.Size,
		data.ValueSize,
		data.Value,
	)
}

func (s *SQLCommon) attemptDataInsert(ctx context.Context, tx *txWrapper, data *core.Data, requestConflictEmptyResult bool) (int64, error) {
	return s.insertTxExt(ctx, dataTable, tx,
		s.setDataInsertValues(sq.Insert(dataTable).Columns(dataColumnsWithValue...), data),
		func() {
			s.callbacks.UUIDCollectionNSEvent(database.CollectionData, core.ChangeEventTypeCreated, data.Namespace, data.ID)
		}, requestConflictEmptyResult)
}

func (s *SQLCommon) UpsertData(ctx context.Context, data *core.Data, optimization database.UpsertOptimization) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	// This is a performance critical function, as we stream data into the database for every message, in every batch.
	//
	// First attempt the operation based on the optimization passed in.
	// The expectation is that the optimization will hit almost all of the time,
	// as only recovery paths require us to go down the un-optimized route.
	optimized := false
	if optimization == database.UpsertOptimizationNew {
		_, opErr := s.attemptDataInsert(ctx, tx, data, true /* we want a failure here we can progress past */)
		optimized = opErr == nil
	} else if optimization == database.UpsertOptimizationExisting {
		rowsAffected, opErr := s.attemptDataUpdate(ctx, tx, data)
		optimized = opErr == nil && rowsAffected == 1
	}

	if !optimized {
		// Do a select within the transaction to determine if the UUID already exists
		dataRows, _, err := s.queryTx(ctx, dataTable, tx,
			sq.Select("hash").
				From(dataTable).
				Where(sq.Eq{"id": data.ID}),
		)
		if err != nil {
			return err
		}

		existing := dataRows.Next()
		if existing {
			var hash *fftypes.Bytes32
			_ = dataRows.Scan(&hash)
			if !fftypes.SafeHashCompare(hash, data.Hash) {
				dataRows.Close()
				log.L(ctx).Errorf("Existing=%s New=%s", hash, data.Hash)
				return database.HashMismatch
			}
		}
		dataRows.Close()

		if existing {
			if _, err = s.attemptDataUpdate(ctx, tx, data); err != nil {
				return err
			}
		} else {
			if _, err = s.attemptDataInsert(ctx, tx, data, false); err != nil {
				return err
			}
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) InsertDataArray(ctx context.Context, dataArray core.DataArray) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	if s.features.MultiRowInsert {
		query := sq.Insert(dataTable).Columns(dataColumnsWithValue...)
		for _, data := range dataArray {
			query = s.setDataInsertValues(query, data)
		}
		sequences := make([]int64, len(dataArray))
		err := s.insertTxRows(ctx, dataTable, tx, query, func() {
			for _, data := range dataArray {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionData, core.ChangeEventTypeCreated, data.Namespace, data.ID)
			}
		}, sequences, true /* we want the caller to be able to retry with individual upserts */)
		if err != nil {
			return err
		}
	} else {
		// Fall back to individual inserts grouped in a TX
		for _, data := range dataArray {
			_, err := s.attemptDataInsert(ctx, tx, data, false)
			if err != nil {
				return err
			}
		}
	}

	return s.commitTx(ctx, tx, autoCommit)

}

func (s *SQLCommon) dataResult(ctx context.Context, row *sql.Rows, withValue bool) (*core.Data, error) {
	data := core.Data{
		Datatype: &core.DatatypeRef{},
		Blob:     &core.BlobRef{},
	}
	results := []interface{}{
		&data.ID,
		&data.Validator,
		&data.Namespace,
		&data.Datatype.Name,
		&data.Datatype.Version,
		&data.Hash,
		&data.Created,
		&data.Blob.Hash,
		&data.Blob.Public,
		&data.Blob.Name,
		&data.Blob.Size,
		&data.ValueSize,
	}
	if withValue {
		results = append(results, &data.Value)
	}
	err := row.Scan(results...)
	if data.Blob.Hash == nil && data.Blob.Public == "" {
		data.Blob = nil
	}
	if data.Datatype.Name == "" && data.Datatype.Version == "" {
		data.Datatype = nil
	}
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, dataTable)
	}
	return &data, nil
}

func (s *SQLCommon) GetDataByID(ctx context.Context, id *fftypes.UUID, withValue bool) (message *core.Data, err error) {

	var cols []string
	if withValue {
		cols = dataColumnsWithValue
	} else {
		cols = dataColumnsNoValue
	}
	rows, _, err := s.query(ctx, dataTable,
		sq.Select(cols...).
			From(dataTable).
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Data '%s' not found", id)
		return nil, nil
	}

	data, err := s.dataResult(ctx, rows, withValue)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *SQLCommon) GetData(ctx context.Context, filter database.Filter) (message core.DataArray, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(dataColumnsWithValue...).From(dataTable), filter, dataFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, dataTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	data := core.DataArray{}
	for rows.Next() {
		d, err := s.dataResult(ctx, rows, true)
		if err != nil {
			return nil, nil, err
		}
		data = append(data, d)
	}

	return data, s.queryRes(ctx, dataTable, tx, fop, fi), err

}

func (s *SQLCommon) GetDataRefs(ctx context.Context, filter database.Filter) (message core.DataRefs, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select("id", "hash").From(dataTable), filter, dataFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, dataTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	refs := core.DataRefs{}
	for rows.Next() {
		ref := core.DataRef{}
		err := rows.Scan(
			&ref.ID,
			&ref.Hash,
		)
		if err != nil {
			return nil, nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, dataTable)
		}
		refs = append(refs, &ref)
	}

	return refs, s.queryRes(ctx, dataTable, tx, fop, fi), err

}

func (s *SQLCommon) UpdateData(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update(dataTable), update, dataFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, dataTable, tx, query, nil /* no change events for filter based updates */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
