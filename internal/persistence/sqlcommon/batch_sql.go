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

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
)

var (
	batchColumns = []string{
		"id",
		"author",
		"created",
		"hash",
		"payload",
	}
)

func (s *SQLCommon) UpsertBatch(ctx context.Context, batch *fftypes.Batch) (err error) {
	ctx, tx, err := s.beginTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx)

	// Do a select within the transaction to detemine if the UUID already exists
	batchRows, err := s.queryTx(ctx, tx,
		sq.Select("id").
			From("batches").
			Where(sq.Eq{"id": batch.ID}),
	)
	if err != nil {
		return err
	}
	defer batchRows.Close()

	if batchRows.Next() {
		// Update the batch
		if _, err = s.updateTx(ctx, tx,
			sq.Update("batches").
				Set("author", batch.Author).
				Set("created", batch.Created).
				Set("hash", batch.Hash).
				Set("payload", batch.Payload).
				Where(sq.Eq{"id": batch.ID}),
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("batches").
				Columns(batchColumns...).
				Values(
					batch.ID,
					batch.Author,
					batch.Created,
					batch.Hash,
					batch.Payload,
				),
		); err != nil {
			return err
		}
	}

	if err = s.commitTx(ctx, tx); err != nil {
		return err
	}

	return nil
}

func (s *SQLCommon) batchResult(ctx context.Context, row *sql.Rows) (*fftypes.Batch, error) {
	var batch fftypes.Batch
	err := row.Scan(
		&batch.ID,
		&batch.Author,
		&batch.Created,
		&batch.Hash,
		&batch.Payload,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "batches")
	}
	return &batch, nil
}

func (s *SQLCommon) GetBatchById(ctx context.Context, id *uuid.UUID) (message *fftypes.Batch, err error) {

	rows, err := s.query(ctx,
		sq.Select(batchColumns...).
			From("batches").
			Where(sq.Eq{"id": id}),
	)
	if err != nil {
		return nil, err
	}

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
