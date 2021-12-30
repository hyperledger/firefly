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
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestContractAPIE2EWithDB(t *testing.T) {
	log.SetLevel("trace")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new contract API
	apiID := fftypes.NewUUID()
	interfaceID := fftypes.NewUUID()

	contractAPI := &fftypes.ContractAPI{
		ID:        apiID,
		Namespace: "ns1",
		Name:      "banana",
		Interface: &fftypes.FFIReference{
			ID:      interfaceID,
			Name:    "banana",
			Version: "v1.0.0",
		},
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionContractAPIs, fftypes.ChangeEventTypeCreated, "ns1", apiID, mock.Anything).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionContractAPIs, fftypes.ChangeEventTypeUpdated, "ns1", apiID, mock.Anything).Return()

	err := s.UpsertContractAPI(ctx, contractAPI, database.UpsertOptimizationSkip)
	assert.NoError(t, err)

	// Check we get the exact same ContractAPI back
	dataRead, err := s.GetContractAPIByID(ctx, apiID)
	assert.NoError(t, err)
	assert.NotNil(t, dataRead)
	assert.Equal(t, *apiID, *dataRead.ID)

	contractAPI.Interface.Version = "v1.1.0"

	err = s.UpsertContractAPI(ctx, contractAPI, database.UpsertOptimizationSkip)
	assert.NoError(t, err)

	// Check we get the exact same ContractAPI back
	dataRead, err = s.GetContractAPIByID(ctx, apiID)
	assert.NoError(t, err)
	assert.NotNil(t, dataRead)
	assert.Equal(t, *apiID, *dataRead.ID)
}

func TestContractAPIDBFailBeginTransaction(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertContractAPI(context.Background(), &fftypes.ContractAPI{}, database.UpsertOptimizationSkip)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContractAPIDBFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertContractAPI(context.Background(), &fftypes.ContractAPI{}, database.UpsertOptimizationSkip)
	assert.Regexp(t, "pop", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContractAPIDBFailInsert(t *testing.T) {
	rows := sqlmock.NewRows([]string{"id", "interface_id", "ledger", "location", "name", "namespace"})
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)
	// mock.ExpectQuery("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{},
	}
	err := s.UpsertContractAPI(context.Background(), api, database.UpsertOptimizationSkip)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContractAPIDBFailUpdate(t *testing.T) {
	rows := sqlmock.NewRows([]string{"id", "interface_id", "ledger", "location", "name", "namespace"}).
		AddRow("7e2c001c-e270-4fd7-9e82-9dacee843dc2", "8fcc4938-7d8b-4c00-a71b-1b46837c8ab1", nil, nil, "banana", "ns1")
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)
	mock.ExpectQuery("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	api := &fftypes.ContractAPI{
		Interface: &fftypes.FFIReference{},
	}
	err := s.UpsertContractAPI(context.Background(), api, database.UpsertOptimizationSkip)
	assert.Regexp(t, "pop", err)
}

func TestContractAPIDBFailScan(t *testing.T) {
	s, mock := newMockProvider().init()
	apiID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("only one"))
	_, err := s.GetContractAPIByID(context.Background(), apiID)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContractAPIDBSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	apiID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetContractAPIByID(context.Background(), apiID)
	assert.Regexp(t, "pop", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContractAPIDBNoRows(t *testing.T) {
	s, mock := newMockProvider().init()
	apiID := fftypes.NewUUID()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"id", "interface_id", "ledger", "location", "name", "namespace"}))
	_, err := s.GetContractAPIByID(context.Background(), apiID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractAPIs(t *testing.T) {
	fb := database.ContractAPIQueryFactory.NewFilter(context.Background())
	s, mock := newMockProvider().init()
	rows := sqlmock.NewRows([]string{"id", "interface_id", "ledger", "location", "name", "namespace"}).
		AddRow("7e2c001c-e270-4fd7-9e82-9dacee843dc2", "8fcc4938-7d8b-4c00-a71b-1b46837c8ab1", nil, nil, "banana", "ns1")
	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)
	_, _, err := s.GetContractAPIs(context.Background(), "ns1", fb.And())
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractAPIsFilterSelectFail(t *testing.T) {
	fb := database.ContractAPIQueryFactory.NewFilter(context.Background())
	s, _ := newMockProvider().init()
	_, _, err := s.GetContractAPIs(context.Background(), "ns1", fb.And(fb.Eq("id", map[bool]bool{true: false})))
	assert.Error(t, err)
}

func TestGetContractAPIsQueryFail(t *testing.T) {
	fb := database.ContractAPIQueryFactory.NewFilter(context.Background())
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, _, err := s.GetContractAPIs(context.Background(), "ns1", fb.And())
	assert.Regexp(t, "pop", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractAPIsQueryResultFail(t *testing.T) {
	fb := database.ContractAPIQueryFactory.NewFilter(context.Background())
	s, mock := newMockProvider().init()
	rows := sqlmock.NewRows([]string{"id", "interface_id", "ledger", "location", "name", "namespace"}).
		AddRow("7e2c001c-e270-4fd7-9e82-9dacee843dc2", "8fcc4938-7d8b-4c00-a71b-1b46837c8ab1", nil, nil, "apple", "ns1").
		AddRow("69851ca3-e9f9-489b-8731-dc6a7d990291", "4db4952e-4669-4243-a387-8f0f609e92bd", nil, nil, "orange", nil)
	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)
	_, _, err := s.GetContractAPIs(context.Background(), "ns1", fb.And())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractAPIByName(t *testing.T) {
	s, mock := newMockProvider().init()
	rows := sqlmock.NewRows([]string{"id", "interface_id", "ledger", "location", "name", "namespace"}).
		AddRow("7e2c001c-e270-4fd7-9e82-9dacee843dc2", "8fcc4938-7d8b-4c00-a71b-1b46837c8ab1", nil, nil, "banana", "ns1")
	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)
	api, err := s.GetContractAPIByName(context.Background(), "ns1", "banana")
	assert.NotNil(t, api)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
