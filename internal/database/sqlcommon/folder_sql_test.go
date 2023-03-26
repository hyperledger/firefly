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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
)

func TestFoldersE2EWithDB(t *testing.T) {
	dataID := fftypes.NewUUID()
	namespace := "e2e"
	log.SetLevel("debug")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new folder entry
	folder := &core.Folder{
		Namespace:  namespace,
		Hash:       fftypes.NewRandB32(),
		Size:       12345,
		PayloadRef: fftypes.NewRandB32().String(),
		Peer:       "peer1",
		Created:    fftypes.Now(),
		DataID:     dataID,
	}
	err := s.InsertFolder(ctx, folder)
	assert.NoError(t, err)

	// Check we get the exact same folder back
	fb := database.FolderQueryFactory.NewFilter(ctx)
	folders, _, err := s.GetFolders(ctx, namespace, fb.Eq("payloadref", folder.PayloadRef))
	folderRead := folders[0]
	assert.NoError(t, err)
	assert.NotNil(t, folderRead)
	folderJson, _ := json.Marshal(&folder)
	folderReadJson, _ := json.Marshal(&folderRead)
	assert.Equal(t, string(folderJson), string(folderReadJson))

	// Query back the folder
	fb = database.FolderQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("hash", folder.Hash),
		fb.Eq("payloadref", folder.PayloadRef),
		fb.Eq("created", folder.Created),
	)
	folderRes, res, err := s.GetFolders(ctx, namespace, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(folderRes))
	assert.Equal(t, int64(1), *res.TotalCount)
	folderReadJson, _ = json.Marshal(folderRes[0])
	assert.Equal(t, string(folderJson), string(folderReadJson))
	assert.Equal(t, folder.Sequence, folderRes[0].Sequence)

	// Test delete
	err = s.DeleteFolder(ctx, folder.Sequence)
	assert.NoError(t, err)
	folders, _, err = s.GetFolders(ctx, namespace, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(folders))

}

func TestInsertFolderFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertFolder(context.Background(), &core.Folder{})
	assert.Regexp(t, "FF00175", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFolderFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.InsertFolder(context.Background(), &core.Folder{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF00177", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFolderFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertFolder(context.Background(), &core.Folder{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF00180", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFoldersBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertFolders(context.Background(), []*core.Folder{})
	assert.Regexp(t, "FF00175", err)
	assert.NoError(t, mock.ExpectationsWereMet())
	s.callbacks.AssertExpectations(t)
}

func TestInsertFoldersMultiRowOK(t *testing.T) {
	s := newMockProvider()
	s.multiRowInsert = true
	s.fakePSQLInsert = true
	s, mock := s.init()

	folder1 := &core.Folder{Hash: fftypes.NewRandB32(), PayloadRef: "pay1"}
	folder2 := &core.Folder{Hash: fftypes.NewRandB32(), PayloadRef: "pay2"}

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT.*").WillReturnRows(sqlmock.NewRows([]string{s.SequenceColumn()}).
		AddRow(int64(1001)).
		AddRow(int64(1002)),
	)
	mock.ExpectCommit()
	err := s.InsertFolders(context.Background(), []*core.Folder{folder1, folder2})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	s.callbacks.AssertExpectations(t)
}

func TestInsertFoldersMultiRowFail(t *testing.T) {
	s := newMockProvider()
	s.multiRowInsert = true
	s.fakePSQLInsert = true
	s, mock := s.init()
	folder1 := &core.Folder{Hash: fftypes.NewRandB32(), PayloadRef: "pay1"}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT.*").WillReturnError(fmt.Errorf("pop"))
	err := s.InsertFolders(context.Background(), []*core.Folder{folder1})
	assert.Regexp(t, "FF00177", err)
	assert.NoError(t, mock.ExpectationsWereMet())
	s.callbacks.AssertExpectations(t)
}

func TestInsertFoldersSingleRowFail(t *testing.T) {
	s, mock := newMockProvider().init()
	folder1 := &core.Folder{Hash: fftypes.NewRandB32(), PayloadRef: "pay1"}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT.*").WillReturnError(fmt.Errorf("pop"))
	err := s.InsertFolders(context.Background(), []*core.Folder{folder1})
	assert.Regexp(t, "FF00177", err)
	assert.NoError(t, mock.ExpectationsWereMet())
	s.callbacks.AssertExpectations(t)
}

// func TestGetFolderByIDSelectFail(t *testing.T) {
// 	s, mock := newMockProvider().init()
// 	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
// 	_, err := s.GetFolder(context.Background(), "ns1", fftypes.NewUUID(), fftypes.NewRandB32())
// 	assert.Regexp(t, "FF00176", err)
// 	assert.NoError(t, mock.ExpectationsWereMet())
// }

// func TestGetFolderByIDNotFound(t *testing.T) {
// 	s, mock := newMockProvider().init()
// 	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
// 	msg, err := s.GetFolder(context.Background(), "ns1", fftypes.NewUUID(), fftypes.NewRandB32())
// 	assert.NoError(t, err)
// 	assert.Nil(t, msg)
// 	assert.NoError(t, mock.ExpectationsWereMet())
// }

// func TestGetFolderByIDScanFail(t *testing.T) {
// 	s, mock := newMockProvider().init()
// 	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
// 	_, err := s.GetFolder(context.Background(), "ns1", fftypes.NewUUID(), fftypes.NewRandB32())
// 	assert.Regexp(t, "FF10121", err)
// 	assert.NoError(t, mock.ExpectationsWereMet())
// }

func TestGetFolderQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.FolderQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, _, err := s.GetFolders(context.Background(), "ns1", f)
	assert.Regexp(t, "FF00176", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFolderBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.FolderQueryFactory.NewFilter(context.Background()).Eq("hash", map[bool]bool{true: false})
	_, _, err := s.GetFolders(context.Background(), "ns1", f)
	assert.Regexp(t, "FF00143.*type", err)
}

func TestGetFolderReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
	f := database.FolderQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, _, err := s.GetFolders(context.Background(), "ns1", f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFolderDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteFolder(context.Background(), 12345)
	assert.Regexp(t, "FF00175", err)
}

func TestFolderDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteFolder(context.Background(), 12345)
	assert.Regexp(t, "FF00179", err)
}
