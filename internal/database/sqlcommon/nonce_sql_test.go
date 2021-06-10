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

func TestNoncesE2EWithDB(t *testing.T) {
	log.SetLevel("trace")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new nonce entry
	nonceZero := &fftypes.Nonce{
		Context: fftypes.NewRandB32(),
		Group:   fftypes.NewRandB32(),
		Topic:   "topic12345",
	}
	err := s.UpsertNonceNext(ctx, nonceZero)
	assert.NoError(t, err)

	// Check we get the exact same nonce back
	nonceRead, err := s.GetNonce(ctx, nonceZero.Context)
	assert.NoError(t, err)
	assert.NotNil(t, nonceRead)
	nonceJson, _ := json.Marshal(&nonceZero)
	nonceReadJson, _ := json.Marshal(&nonceRead)
	assert.Equal(t, string(nonceJson), string(nonceReadJson))

	// Update the nonce (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	var nonceUpdated fftypes.Nonce
	nonceUpdated = *nonceZero

	// Increment a couple of times
	err = s.UpsertNonceNext(context.Background(), &nonceUpdated)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), nonceUpdated.Nonce)
	err = s.UpsertNonceNext(context.Background(), &nonceUpdated)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), nonceUpdated.Nonce)

	// Check we get the exact same data back
	nonceRead, err = s.GetNonce(ctx, nonceUpdated.Context)
	assert.NoError(t, err)
	nonceJson, _ = json.Marshal(&nonceUpdated)
	nonceReadJson, _ = json.Marshal(&nonceRead)
	assert.Equal(t, string(nonceJson), string(nonceReadJson))

	// Query back the nonce
	fb := database.NonceQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("context", nonceUpdated.Context),
		fb.Eq("nonce", nonceUpdated.Nonce),
		fb.Eq("group", nonceUpdated.Group),
		fb.Eq("topic", nonceUpdated.Topic),
	)
	nonceRes, err := s.GetNonces(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(nonceRes))
	nonceReadJson, _ = json.Marshal(nonceRes[0])
	assert.Equal(t, string(nonceJson), string(nonceReadJson))

	// Test delete
	err = s.DeleteNonce(ctx, nonceUpdated.Context)
	assert.NoError(t, err)
	nonces, err := s.GetNonces(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(nonces))

}

func TestUpsertNonceFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertNonceNext(context.Background(), &fftypes.Nonce{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNonceFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNonceNext(context.Background(), &fftypes.Nonce{Context: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNonceFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNonceNext(context.Background(), &fftypes.Nonce{Context: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNonceFailScan(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}).AddRow())
	mock.ExpectRollback()
	err := s.UpsertNonceNext(context.Background(), &fftypes.Nonce{Context: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNonceFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"nonce", "sequence"}).AddRow(int64(12345), int64(11111)))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNonceNext(context.Background(), &fftypes.Nonce{Context: fftypes.NewRandB32()})
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNonceSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetNonce(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNonceNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context", "nonce", "group_hash", "topic"}))
	msg, err := s.GetNonce(context.Background(), fftypes.NewRandB32())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNonceScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context"}).AddRow("only one"))
	_, err := s.GetNonce(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNonceQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.NonceQueryFactory.NewFilter(context.Background()).Eq("context", "")
	_, err := s.GetNonces(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNonceBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.NonceQueryFactory.NewFilter(context.Background()).Eq("context", map[bool]bool{true: false})
	_, err := s.GetNonces(context.Background(), f)
	assert.Regexp(t, "FF10149.*context", err)
}

func TestGetNonceReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"context"}).AddRow("only one"))
	f := database.NonceQueryFactory.NewFilter(context.Background()).Eq("topic", "")
	_, err := s.GetNonces(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNonceDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteNonce(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10114", err)
}

func TestNonceDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteNonce(context.Background(), fftypes.NewRandB32())
	assert.Regexp(t, "FF10118", err)
}
