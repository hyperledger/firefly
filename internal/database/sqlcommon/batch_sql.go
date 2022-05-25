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
	batchColumns = []string{
		"id",
		"btype",
		"namespace",
		"author",
		"key",
		"group_hash",
		"created",
		"hash",
		"manifest",
		"confirmed",
		"tx_type",
		"tx_id",
		"node_id",
	}
	batchFilterFieldMap = map[string]string{
		"type":    "btype",
		"tx.type": "tx_type",
		"tx.id":   "tx_id",
		"group":   "group_hash",
		"node":    "node_id",
	}
)

const batchesTable = "batches"

func (s *SQLCommon) UpsertBatch(ctx context.Context, batch *core.BatchPersisted) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	// Do a select within the transaction to detemine if the UUID already exists
	batchRows, _, err := s.queryTx(ctx, batchesTable, tx,
		sq.Select("hash").
			From(batchesTable).
			Where(sq.Eq{"id": batch.ID}),
	)
	if err != nil {
		return err
	}

	existing := batchRows.Next()
	if existing {
		var hash *fftypes.Bytes32
		_ = batchRows.Scan(&hash)
		if !fftypes.SafeHashCompare(hash, batch.Hash) {
			batchRows.Close()
			log.L(ctx).Errorf("Existing=%s New=%s", hash, batch.Hash)
			return database.HashMismatch
		}
	}
	batchRows.Close()

	if existing {

		// Update the batch
		if _, err = s.updateTx(ctx, batchesTable, tx,
			sq.Update(batchesTable).
				Set("btype", string(batch.Type)).
				Set("namespace", batch.Namespace).
				Set("author", batch.Author).
				Set("key", batch.Key).
				Set("group_hash", batch.Group).
				Set("created", batch.Created).
				Set("hash", batch.Hash).
				Set("manifest", batch.Manifest).
				Set("confirmed", batch.Confirmed).
				Set("tx_type", batch.TX.Type).
				Set("tx_id", batch.TX.ID).
				Set("node_id", batch.Node).
				Where(sq.Eq{"id": batch.ID}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionBatches, core.ChangeEventTypeUpdated, batch.Namespace, batch.ID)
			},
		); err != nil {
			return err
		}
	} else {

		if _, err = s.insertTx(ctx, batchesTable, tx,
			sq.Insert(batchesTable).
				Columns(batchColumns...).
				Values(
					batch.ID,
					string(batch.Type),
					batch.Namespace,
					batch.Author,
					batch.Key,
					batch.Group,
					batch.Created,
					batch.Hash,
					batch.Manifest,
					batch.Confirmed,
					batch.TX.Type,
					batch.TX.ID,
					batch.Node,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionBatches, core.ChangeEventTypeCreated, batch.Namespace, batch.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) batchResult(ctx context.Context, row *sql.Rows) (*core.BatchPersisted, error) {
	var batch core.BatchPersisted
	err := row.Scan(
		&batch.ID,
		&batch.Type,
		&batch.Namespace,
		&batch.Author,
		&batch.Key,
		&batch.Group,
		&batch.Created,
		&batch.Hash,
		&batch.Manifest,
		&batch.Confirmed,
		&batch.TX.Type,
		&batch.TX.ID,
		&batch.Node,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, batchesTable)
	}
	return &batch, nil
}

func (s *SQLCommon) GetBatchByID(ctx context.Context, id *fftypes.UUID) (message *core.BatchPersisted, err error) {

	rows, _, err := s.query(ctx, batchesTable,
		sq.Select(batchColumns...).
			From(batchesTable).
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Batch '%s' not found", id)
		return nil, nil
	}

	batch, err := s.batchResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return batch, nil
}

func (s *SQLCommon) GetBatches(ctx context.Context, filter database.Filter) (message []*core.BatchPersisted, res *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(batchColumns...).From(batchesTable), filter, batchFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, batchesTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	batches := []*core.BatchPersisted{}
	for rows.Next() {
		batch, err := s.batchResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		batches = append(batches, batch)
	}

	return batches, s.queryRes(ctx, batchesTable, tx, fop, fi), err

}

func (s *SQLCommon) UpdateBatch(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update(batchesTable), update, batchFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, batchesTable, tx, query, nil /* no change events on filter update */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
