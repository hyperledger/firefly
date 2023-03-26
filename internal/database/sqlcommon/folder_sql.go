// Copyright © 2023 Kaleido, Inc.
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
	"github.com/hyperledger/firefly-common/pkg/dbsql"
	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

var (
	folderColumns = []string{
		"id",
		"name",
		"path",
	}
	folderFilterFieldMap = map[string]string{}
)

const foldersTable = "folders"

func (s *SQLCommon) EnsureFolder(ctx context.Context, path string) (err error) {
	ctx, tx, autoCommit, err := s.BeginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.RollbackTx(ctx, tx, autoCommit)

	err = s.attemptFolderInsert(ctx, tx, folder)
	if err != nil {
		return err
	}

	return s.CommitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) setFolderInsertValues(query sq.InsertBuilder, folder *core.Folder) sq.InsertBuilder {
	return query.Values(
		folder.Namespace,
		folder.Hash,
		folder.PayloadRef,
		folder.Created,
		folder.Peer,
		folder.Size,
		folder.DataID,
	)
}

func (s *SQLCommon) attemptFolderInsert(ctx context.Context, tx *dbsql.TXWrapper, folder *core.Folder) (err error) {
	folder.Sequence, err = s.InsertTx(ctx, foldersTable, tx,
		s.setFolderInsertValues(sq.Insert(foldersTable).Columns(folderColumns...), folder),
		nil, // no change events for folders
	)
	return err
}

func (s *SQLCommon) InsertFolders(ctx context.Context, folders []*core.Folder) (err error) {

	ctx, tx, autoCommit, err := s.BeginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.RollbackTx(ctx, tx, autoCommit)

	if s.Features().MultiRowInsert {
		query := sq.Insert(foldersTable).Columns(folderColumns...)
		for _, folder := range folders {
			query = s.setFolderInsertValues(query, folder)
		}
		sequences := make([]int64, len(folders))
		err := s.InsertTxRows(ctx, foldersTable, tx, query,
			nil, /* no change events for folders */
			sequences,
			true /* we want the caller to be able to retry with individual upserts */)
		if err != nil {
			return err
		}
	} else {
		// Fall back to individual inserts grouped in a TX
		for _, folder := range folders {
			err := s.attemptFolderInsert(ctx, tx, folder)
			if err != nil {
				return err
			}
		}
	}

	return s.CommitTx(ctx, tx, autoCommit)

}

func (s *SQLCommon) folderResult(ctx context.Context, row *sql.Rows) (*core.Folder, error) {
	folder := core.Folder{}
	err := row.Scan(
		&folder.Namespace,
		&folder.Hash,
		&folder.PayloadRef,
		&folder.Created,
		&folder.Peer,
		&folder.Size,
		&folder.DataID,
		&folder.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, foldersTable)
	}
	return &folder, nil
}

func (s *SQLCommon) GetFolders(ctx context.Context, namespace string, filter ffapi.Filter) (message []*core.Folder, res *ffapi.FilterResult, err error) {

	cols := append([]string{}, folderColumns...)
	cols = append(cols, s.SequenceColumn())
	query, fop, fi, err := s.FilterSelect(
		ctx,
		"",
		sq.Select(cols...).From(foldersTable), filter, folderFilterFieldMap,
		[]interface{}{"sequence"},
		sq.Eq{"namespace": namespace})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.Query(ctx, foldersTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	folder := []*core.Folder{}
	for rows.Next() {
		d, err := s.folderResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		folder = append(folder, d)
	}

	return folder, s.QueryRes(ctx, foldersTable, tx, fop, fi), err

}

func (s *SQLCommon) DeleteFolder(ctx context.Context, sequence int64) (err error) {

	ctx, tx, autoCommit, err := s.BeginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.RollbackTx(ctx, tx, autoCommit)

	err = s.DeleteTx(ctx, foldersTable, tx, sq.Delete(foldersTable).Where(sq.Eq{
		s.SequenceColumn(): sequence,
	}), nil /* no change events for folders */)
	if err != nil {
		return err
	}

	return s.CommitTx(ctx, tx, autoCommit)
}
