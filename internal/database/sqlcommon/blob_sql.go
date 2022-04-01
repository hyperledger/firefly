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
	blobColumns = []string{
		"hash",
		"payload_ref",
		"peer",
		"created",
		"size",
	}
	blobFilterFieldMap = map[string]string{
		"payloadref": "payload_ref",
	}
)

func (s *SQLCommon) InsertBlob(ctx context.Context, blob *fftypes.Blob) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.attemptBlobInsert(ctx, tx, blob)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) setBlobInsertValues(query sq.InsertBuilder, blob *fftypes.Blob) sq.InsertBuilder {
	return query.Values(
		blob.Hash,
		blob.PayloadRef,
		blob.Peer,
		blob.Created,
		blob.Size,
	)
}

func (s *SQLCommon) attemptBlobInsert(ctx context.Context, tx *txWrapper, blob *fftypes.Blob) (err error) {
	blob.Sequence, err = s.insertTx(ctx, tx,
		s.setBlobInsertValues(sq.Insert("blobs").Columns(blobColumns...), blob),
		nil, // no change events for blobs
	)
	return err
}

func (s *SQLCommon) InsertBlobs(ctx context.Context, blobs []*fftypes.Blob) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	if s.features.MultiRowInsert {
		query := sq.Insert("blobs").Columns(dataColumnsWithValue...)
		for _, blob := range blobs {
			query = s.setBlobInsertValues(query, blob)
		}
		sequences := make([]int64, len(blobs))
		err := s.insertTxRows(ctx, tx, query,
			nil, /* no change events for blobs */
			sequences,
			true /* we want the caller to be able to retry with individual upserts */)
		if err != nil {
			return err
		}
	} else {
		// Fall back to individual inserts grouped in a TX
		for _, blob := range blobs {
			err := s.attemptBlobInsert(ctx, tx, blob)
			if err != nil {
				return err
			}
		}
	}

	return s.commitTx(ctx, tx, autoCommit)

}

func (s *SQLCommon) blobResult(ctx context.Context, row *sql.Rows) (*fftypes.Blob, error) {
	blob := fftypes.Blob{}
	err := row.Scan(
		&blob.Hash,
		&blob.PayloadRef,
		&blob.Peer,
		&blob.Created,
		&blob.Size,
		&blob.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "blobs")
	}
	return &blob, nil
}

func (s *SQLCommon) getBlobPred(ctx context.Context, desc string, pred interface{}) (message *fftypes.Blob, err error) {
	cols := append([]string{}, blobColumns...)
	cols = append(cols, sequenceColumn)
	rows, _, err := s.query(ctx,
		sq.Select(cols...).
			From("blobs").
			Where(pred).
			Limit(1),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Blob '%s' not found", desc)
		return nil, nil
	}

	blob, err := s.blobResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return blob, nil
}

func (s *SQLCommon) GetBlobMatchingHash(ctx context.Context, hash *fftypes.Bytes32) (message *fftypes.Blob, err error) {
	return s.getBlobPred(ctx, hash.String(), sq.Eq{
		"hash": hash,
	})
}

func (s *SQLCommon) GetBlobs(ctx context.Context, filter database.Filter) (message []*fftypes.Blob, res *database.FilterResult, err error) {

	cols := append([]string{}, blobColumns...)
	cols = append(cols, sequenceColumn)
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(cols...).From("blobs"), filter, blobFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	blob := []*fftypes.Blob{}
	for rows.Next() {
		d, err := s.blobResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		blob = append(blob, d)
	}

	return blob, s.queryRes(ctx, tx, "blobs", fop, fi), err

}

func (s *SQLCommon) DeleteBlob(ctx context.Context, sequence int64) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	err = s.deleteTx(ctx, tx, sq.Delete("blobs").Where(sq.Eq{
		sequenceColumn: sequence,
	}), nil /* no change events for blobs */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
