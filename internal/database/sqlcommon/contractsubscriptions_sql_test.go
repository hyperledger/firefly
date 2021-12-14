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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestContractSubscriptionE2EWithDB(t *testing.T) {
	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new contract subscription entry
	location := fftypes.JSONObject{"path": "my-api"}
	locationJson, _ := json.Marshal(location)
	sub := &fftypes.ContractSubscription{
		ID:         fftypes.NewUUID(),
		Interface:  fftypes.NewUUID(),
		Event:      fftypes.NewUUID(),
		Namespace:  "ns",
		Name:       "sub1",
		ProtocolID: "sb-123",
		Location:   locationJson,
	}

	err := s.UpsertContractSubscription(ctx, sub)
	assert.NotNil(t, sub.Created)
	assert.NoError(t, err)
	subJson, _ := json.Marshal(&sub)

	// Query back the subscription (by query filter)
	fb := database.ContractSubscriptionQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("protocolid", sub.ProtocolID),
	)
	subs, res, err := s.GetContractSubscriptions(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, int64(1), *res.TotalCount)
	subReadJson, _ := json.Marshal(subs[0])
	assert.Equal(t, string(subJson), string(subReadJson))

	// Query back the subscription (by name)
	subRead, err := s.GetContractSubscription(ctx, "ns", "sub1")
	assert.NoError(t, err)
	subReadJson, _ = json.Marshal(subRead)
	assert.Equal(t, string(subJson), string(subReadJson))

	// Query back the subscription (by ID)
	subRead, err = s.GetContractSubscriptionByID(ctx, sub.ID)
	assert.NoError(t, err)
	subReadJson, _ = json.Marshal(subRead)
	assert.Equal(t, string(subJson), string(subReadJson))

	// Query back the subscription (by protocol ID)
	subRead, err = s.GetContractSubscriptionByProtocolID(ctx, sub.ProtocolID)
	assert.NoError(t, err)
	subReadJson, _ = json.Marshal(subRead)
	assert.Equal(t, string(subJson), string(subReadJson))

	// Update the subscription
	sub.Location = []byte("{}")
	subJson, _ = json.Marshal(&sub)
	err = s.UpsertContractSubscription(ctx, sub)
	assert.NoError(t, err)

	// Query back the subscription (by query filter)
	filter = fb.And(
		fb.Eq("protocolid", sub.ProtocolID),
	)
	subs, res, err = s.GetContractSubscriptions(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, int64(1), *res.TotalCount)
	subReadJson, _ = json.Marshal(subs[0])
	assert.Equal(t, string(subJson), string(subReadJson))

	// Test delete, and refind no return
	err = s.DeleteContractSubscriptionByID(ctx, sub.ID)
	assert.NoError(t, err)
	subs, _, err = s.GetContractSubscriptions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(subs))
}

func TestUpsertContractSubscriptionFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertContractSubscription(context.Background(), &fftypes.ContractSubscription{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContractSubscriptionFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertContractSubscription(context.Background(), &fftypes.ContractSubscription{})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContractSubscriptionFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertContractSubscription(context.Background(), &fftypes.ContractSubscription{})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContractSubscriptionFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}).AddRow("1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertContractSubscription(context.Background(), &fftypes.ContractSubscription{})
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertContractSubscriptionFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertContractSubscription(context.Background(), &fftypes.ContractSubscription{})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractSubscriptionByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetContractSubscriptionByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractSubscriptionByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}))
	msg, err := s.GetContractSubscriptionByID(context.Background(), fftypes.NewUUID())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractSubscriptionByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}).AddRow("only one"))
	_, err := s.GetContractSubscriptionByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractSubscriptionsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.ContractSubscriptionQueryFactory.NewFilter(context.Background()).Eq("protocolid", "")
	_, _, err := s.GetContractSubscriptions(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetContractSubscriptionsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.ContractSubscriptionQueryFactory.NewFilter(context.Background()).Eq("protocolid", map[bool]bool{true: false})
	_, _, err := s.GetContractSubscriptions(context.Background(), f)
	assert.Regexp(t, "FF10149.*id", err)
}

func TestGetContractSubscriptionsScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"protocolid"}).AddRow("only one"))
	f := database.ContractSubscriptionQueryFactory.NewFilter(context.Background()).Eq("protocolid", "")
	_, _, err := s.GetContractSubscriptions(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContractSubscriptionDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteContractSubscriptionByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10114", err)
}

func TestContractSubscriptionDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(contractSubscriptionColumns).AddRow(
		fftypes.NewUUID(), nil, fftypes.NewUUID(), "ns1", "sub1", "123", "{}", fftypes.Now()),
	)
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteContractSubscriptionByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10118", err)
}
