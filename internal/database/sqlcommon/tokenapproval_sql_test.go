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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestApprovalE2EWithDB(t *testing.T) {
	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	approval := &core.TokenApproval{
		LocalID:    fftypes.NewUUID(),
		Pool:       fftypes.NewUUID(),
		Connector:  "erc1155",
		Namespace:  "ns1",
		Key:        "0x01",
		Operator:   "0x02",
		Approved:   true,
		ProtocolID: "0001/01/01",
		Subject:    "12345",
		Active:     true,
		TX: core.TransactionRef{
			Type: core.TransactionTypeTokenApproval,
			ID:   fftypes.NewUUID(),
		},
		BlockchainEvent: fftypes.NewUUID(),
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionTokenApprovals, core.ChangeEventTypeCreated, approval.Namespace, approval.LocalID, mock.Anything).
		Return().Once()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionTokenApprovals, core.ChangeEventTypeUpdated, approval.Namespace, approval.LocalID, mock.Anything).
		Return().Once()

	// Initial list is empty
	fb := database.TokenApprovalQueryFactory.NewFilter(ctx)
	approvals, _, err := s.GetTokenApprovals(ctx, fb.And())
	assert.NoError(t, err)
	assert.NotNil(t, approvals)
	assert.Equal(t, 0, len(approvals))

	// Add one approval
	err = s.UpsertTokenApproval(ctx, approval)
	assert.NoError(t, err)
	assert.NotNil(t, approval.Created)
	approvalJson, _ := json.Marshal(&approval)

	// Query back token approval by ID
	approvalRead, err := s.GetTokenApprovalByID(ctx, approval.LocalID)
	assert.NoError(t, err)
	assert.NotNil(t, approvalRead)
	approvalReadJson, _ := json.Marshal(&approvalRead)
	assert.Equal(t, string(approvalJson), string(approvalReadJson))

	// Query back token approval by protocol ID
	approvalRead, err = s.GetTokenApprovalByProtocolID(ctx, approval.Connector, approval.ProtocolID)
	assert.NoError(t, err)
	assert.NotNil(t, approvalRead)
	approvalReadJson, _ = json.Marshal(&approvalRead)
	assert.Equal(t, string(approvalJson), string(approvalReadJson))

	// Query back token approval by query filter
	filter := fb.And(
		fb.Eq("pool", approval.Pool),
		fb.Eq("key", approval.Key),
		fb.Eq("operator", approval.Operator),
		fb.Eq("subject", approval.Subject),
		fb.Eq("created", approval.Created),
	)
	approvals, res, err := s.GetTokenApprovals(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(approvals))
	assert.Equal(t, int64(1), *res.TotalCount)
	approvalReadJson, _ = json.Marshal(&approvalRead)
	assert.Equal(t, string(approvalJson), string(approvalReadJson))

	// Update the token approval (by upsert)
	approval.Approved = false
	err = s.UpsertTokenApproval(ctx, approval)
	assert.NoError(t, err)

	// Update the token approval (by update)
	filter = fb.And(fb.Eq("subject", approval.Subject))
	update := database.TokenApprovalQueryFactory.NewUpdate(ctx).Set("active", false)
	err = s.UpdateTokenApprovals(ctx, filter, update)
	assert.NoError(t, err)
	approval.Active = false

	// Query back token approval by ID
	approvalRead, err = s.GetTokenApprovalByID(ctx, approval.LocalID)
	assert.NoError(t, err)
	assert.NotNil(t, approvalRead)
	approvalJson, _ = json.Marshal(&approval)
	approvalReadJson, _ = json.Marshal(&approvalRead)
	assert.Equal(t, string(approvalJson), string(approvalReadJson))
}

func TestUpsertApprovalFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenApproval(context.Background(), &core.TokenApproval{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertApprovalFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenApproval(context.Background(), &core.TokenApproval{})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertApprovalFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTokenApproval(context.Background(), &core.TokenApproval{})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertApprovalFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"subject"}).AddRow("1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertTokenApproval(context.Background(), &core.TokenApproval{})
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertApprovalFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"subject"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertTokenApproval(context.Background(), &core.TokenApproval{})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetApprovalByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetTokenApprovalByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetApprovalByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"subject"}))
	a, err := s.GetTokenApprovalByID(context.Background(), fftypes.NewUUID())
	assert.NoError(t, err)
	assert.Nil(t, a)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetApprovalByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"subject"}).AddRow("1"))
	_, err := s.GetTokenApprovalByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetApprovalsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.TokenApprovalQueryFactory.NewFilter(context.Background()).Eq("subject", "")
	_, _, err := s.GetTokenApprovals(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestGetApprovalsBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.TokenApprovalQueryFactory.NewFilter(context.Background()).Eq("subject", map[bool]bool{true: false})
	_, _, err := s.GetTokenApprovals(context.Background(), f)
	assert.Regexp(t, "FF00143.*subject", err)
}

func TestGetApprovalsScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"subject"}).AddRow("1"))
	f := database.TokenApprovalQueryFactory.NewFilter(context.Background()).Eq("subject", "")
	_, _, err := s.GetTokenApprovals(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateApprovalsFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	f := database.TokenApprovalQueryFactory.NewFilter(context.Background()).Eq("subject", "test")
	u := database.TokenApprovalQueryFactory.NewUpdate(context.Background()).Set("active", false)
	err := s.UpdateTokenApprovals(context.Background(), f, u)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateApprovalsBuildUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	f := database.TokenApprovalQueryFactory.NewFilter(context.Background()).Eq("subject", "test")
	u := database.TokenApprovalQueryFactory.NewUpdate(context.Background()).Set("active", map[bool]bool{true: false})
	err := s.UpdateTokenApprovals(context.Background(), f, u)
	assert.Regexp(t, "FF00143.*active", err)
}

func TestUpdateApprovalsBuildFilterFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	f := database.TokenApprovalQueryFactory.NewFilter(context.Background()).Eq("subject", map[bool]bool{true: false})
	u := database.TokenApprovalQueryFactory.NewUpdate(context.Background()).Set("active", false)
	err := s.UpdateTokenApprovals(context.Background(), f, u)
	assert.Regexp(t, "FF00143.*subject", err)
}

func TestUpdateApprovalsUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	f := database.TokenApprovalQueryFactory.NewFilter(context.Background()).Eq("subject", "test")
	u := database.TokenApprovalQueryFactory.NewUpdate(context.Background()).Set("active", false)
	err := s.UpdateTokenApprovals(context.Background(), f, u)
	assert.Regexp(t, "FF10117", err)
}
