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
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTokenPoolE2EWithDB(t *testing.T) {

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new token pool entry
	poolID := fftypes.NewUUID()
	pool := &fftypes.TokenPool{
		ID:        poolID,
		Namespace: "ns1",
		Name:      "my-pool",
		Type:      fftypes.TokenTypeFungible,
		PoolID:    "12345",
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionTransactions, fftypes.ChangeEventTypeCreated, "ns1", poolID, mock.Anything).Return()

	err := s.UpsertTokenPool(ctx, pool, true)
	assert.NoError(t, err)

	// Check we get the exact same token pool back
	poolRead, err := s.GetTokenPoolByID(ctx, pool.ID)
	assert.NoError(t, err)
	assert.NotNil(t, poolRead)
	poolJson, _ := json.Marshal(&pool)
	poolReadJson, _ := json.Marshal(&poolRead)
	assert.Equal(t, string(poolJson), string(poolReadJson))
}

func TestUpsertTokenPoolFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenPool(context.Background(), &fftypes.TokenPool{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenPoolFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenPool(context.Background(), &fftypes.TokenPool{}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenPoolFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTokenPool(context.Background(), &fftypes.TokenPool{}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenPoolFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTokenPool(context.Background(), &fftypes.TokenPool{}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenPoolFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenPool(context.Background(), &fftypes.TokenPool{}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertTokenPoolInsertSuccess(t *testing.T) {
	s, db := newMockProvider().init()
	callbacks := &databasemocks.Callbacks{}
	s.SQLCommon.callbacks = callbacks
	poolID := fftypes.NewUUID()
	pool := &fftypes.TokenPool{
		ID:        poolID,
		Namespace: "ns1",
		Type:      fftypes.TokenTypeNonFungible,
	}

	db.ExpectBegin()
	db.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	db.ExpectExec("INSERT .*").
		WithArgs(poolID, "ns1", "", "", "nonfungible").
		WillReturnResult(sqlmock.NewResult(1, 1))
	db.ExpectCommit()
	callbacks.On("UUIDCollectionNSEvent", database.CollectionTransactions, fftypes.ChangeEventTypeCreated, "ns1", poolID, mock.Anything).Return()
	err := s.UpsertTokenPool(context.Background(), pool, true)
	assert.NoError(t, err)
	assert.NoError(t, db.ExpectationsWereMet())
}

func TestUpsertTokenPoolUpdateSuccess(t *testing.T) {
	s, db := newMockProvider().init()
	callbacks := &databasemocks.Callbacks{}
	s.SQLCommon.callbacks = callbacks
	poolID := fftypes.NewUUID()
	pool := &fftypes.TokenPool{
		ID:        poolID,
		Namespace: "ns1",
	}

	db.ExpectBegin()
	db.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))
	db.ExpectExec("UPDATE .*").WillReturnResult(sqlmock.NewResult(1, 1))
	db.ExpectCommit()
	callbacks.On("UUIDCollectionNSEvent", database.CollectionTransactions, fftypes.ChangeEventTypeUpdated, "ns1", poolID, mock.Anything).Return()
	err := s.UpsertTokenPool(context.Background(), pool, true)
	assert.NoError(t, err)
	assert.NoError(t, db.ExpectationsWereMet())
}
