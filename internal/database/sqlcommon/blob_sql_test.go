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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestBlobsE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new blob entry
	blob := &fftypes.Blob{
		Hash:       fftypes.NewRandB32(),
		PayloadRef: fftypes.NewRandB32().String(),
		Created:    fftypes.Now(),
	}
	err := s.InsertBlob(ctx, blob)
	assert.NoError(t, err)

	// Check we get the exact same blob back
	blobRead, err := s.GetBlobMatchingHash(ctx, blob.Hash)
	assert.NoError(t, err)
	assert.NotNil(t, blobRead)
	blobJson, _ := json.Marshal(&blob)
	blobReadJson, _ := json.Marshal(&blobRead)
	assert.Equal(t, string(blobJson), string(blobReadJson))

	// Query back the blob
	fb := database.BlobQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("hash", blob.Hash),
		fb.Eq("payloadref", blob.PayloadRef),
		fb.Eq("created", blob.Created),
	)
	blobRes, err := s.GetBlobs(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blobRes))
	blobReadJson, _ = json.Marshal(blobRes[0])
	assert.Equal(t, string(blobJson), string(blobReadJson))
	assert.Equal(t, blob.Sequence, blobRes[0].Sequence)

	// Test delete
	err = s.DeleteBlob(ctx, blob.Sequence)
	assert.NoError(t, err)
	blobs, err := s.GetBlobs(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(blobs))

}

func TestUpsertBlobFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertBlob(context.Background(), &fftypes.Blob{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBlobFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.InsertBlob(context.Background(), &fftypes.Blob{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertBlobFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertBlob(context.Background(), &fftypes.Blob{Hash: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlobByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetBlobMatchingHash(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlobByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	msg, err := s.GetBlobMatchingHash(context.Background(), fftypes.NewRandB32())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlobByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
	_, err := s.GetBlobMatchingHash(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlobQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.BlobQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, err := s.GetBlobs(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBlobBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.BlobQueryFactory.NewFilter(context.Background()).Eq("hash", map[bool]bool{true: false})
	_, err := s.GetBlobs(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetBlobReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
	f := database.BlobQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, err := s.GetBlobs(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBlobDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteBlob(context.Background(), 12345)
	assert.Regexp(t, "FF10114", err)
}

func TestBlobDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteBlob(context.Background(), 12345)
	assert.Regexp(t, "FF10118", err)
}
