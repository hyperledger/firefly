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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestUpsertGroupE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new group
	groupHash := fftypes.NewRandB32()
	group := &fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Name:      "group1",
			Namespace: "ns1",
			Members: fftypes.Members{
				{Identity: "0x12345", Node: fftypes.NewUUID()},
				{Identity: "0x23456", Node: fftypes.NewUUID()},
			},
		},
		Hash:    groupHash,
		Created: fftypes.Now(),
	}
	err := s.UpsertGroup(ctx, group, true)
	assert.NoError(t, err)

	// Check we get the exact same group back
	groupRead, err := s.GetGroupByHash(ctx, group.Hash)
	assert.NoError(t, err)
	groupJson, _ := json.Marshal(&group)
	groupReadJson, _ := json.Marshal(&groupRead)
	assert.Equal(t, string(groupJson), string(groupReadJson))

	// Update the group (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	groupUpdated := &fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Name:      "group1",
			Namespace: "ns1",
			Members: fftypes.Members{
				{Identity: "0x12345", Node: fftypes.NewUUID()},
				group.Members[0],
			},
			Ledger: fftypes.NewUUID(),
		},
		Created: fftypes.Now(),
		Message: fftypes.NewUUID(),
		Hash:    groupHash,
	}

	err = s.UpsertGroup(context.Background(), groupUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same group back - note the removal of one of the data elements
	groupRead, err = s.GetGroupByHash(ctx, group.Hash)
	assert.NoError(t, err)
	groupJson, _ = json.Marshal(&groupUpdated)
	groupReadJson, _ = json.Marshal(&groupRead)
	assert.Equal(t, string(groupJson), string(groupReadJson))

	// Query back the group
	fb := database.GroupQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("hash", groupUpdated.Hash),
		fb.Eq("namespace", groupUpdated.Namespace),
		fb.Eq("message", groupUpdated.Message),
		fb.Eq("ledger", groupUpdated.Ledger),
		fb.Gt("created", "0"),
	)
	groups, err := s.GetGroups(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	groupReadJson, _ = json.Marshal(groups[0])
	assert.Equal(t, string(groupJson), string(groupReadJson))

	// Negative test on filter
	filter = fb.And(
		fb.Eq("hash", groupUpdated.Hash.String()),
		fb.Eq("created", "0"),
	)
	groups, err = s.GetGroups(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(groups))

	// Update (testing what's possible at the DB layer)
	newHash := fftypes.NewRandB32()
	up := database.GroupQueryFactory.NewUpdate(ctx).
		Set("hash", newHash)
	err = s.UpdateGroup(ctx, group.Hash, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("hash", newHash),
	)
	groups, err = s.GetGroups(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(groups))

	s.callbacks.AssertExpectations(t)
}

func TestUpsertGroupFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertGroup(context.Background(), &fftypes.Group{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertGroupFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	groupID := fftypes.NewRandB32()
	err := s.UpsertGroup(context.Background(), &fftypes.Group{Hash: groupID}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertGroupFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	groupID := fftypes.NewRandB32()
	err := s.UpsertGroup(context.Background(), &fftypes.Group{Hash: groupID}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertGroupFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow(groupID.String()))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertGroup(context.Background(), &fftypes.Group{Hash: groupID}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertGroupFailUpdateMembers(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow(groupID.String()))
	mock.ExpectExec("UPDATE .*").WillReturnResult(driver.ResultNoRows)
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertGroup(context.Background(), &fftypes.Group{Hash: groupID}, true)
	assert.Regexp(t, "FF10118", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertGroupFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertGroup(context.Background(), &fftypes.Group{Hash: groupID}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMembersMissingOrg(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	err := s.updateMembers(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Group{
		Hash: groupID,
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{{Node: fftypes.NewUUID()}},
		},
	}, false)
	assert.Regexp(t, "FF10220", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMembersMissingNode(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	err := s.updateMembers(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Group{
		Hash: groupID,
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{{Identity: "0x12345"}},
		},
	}, false)
	assert.Regexp(t, "FF10221", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateGroupDataDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	err := s.updateMembers(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Group{
		Hash: groupID,
	}, true)
	assert.Regexp(t, "FF10118", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateGroupDataAddFail(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectBegin()
	tx, _ := s.db.Begin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.updateMembers(context.Background(), &txWrapper{sqlTX: tx}, &fftypes.Group{
		Hash: groupID,
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{{Identity: "0x12345", Node: fftypes.NewUUID()}},
		},
	}, false)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMembersQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	err := s.loadMembers(context.Background(), []*fftypes.Group{{Hash: groupID}})
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMembersScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"identity"}).AddRow("only one"))
	err := s.loadMembers(context.Background(), []*fftypes.Group{{Hash: groupID}})
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadMembersEmpty(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	group := &fftypes.Group{Hash: groupID}
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"identity"}))
	err := s.loadMembers(context.Background(), []*fftypes.Group{group})
	assert.NoError(t, err)
	assert.Equal(t, fftypes.Members{}, group.Members)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetGroupByHash(context.Background(), groupID)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}))
	group, err := s.GetGroupByHash(context.Background(), groupID)
	assert.NoError(t, err)
	assert.Nil(t, group)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
	_, err := s.GetGroupByHash(context.Background(), groupID)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupByIDLoadMembersFail(t *testing.T) {
	s, mock := newMockProvider().init()
	groupID := fftypes.NewRandB32()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(groupColumns).
		AddRow(nil, "ns1", "name1", fftypes.NewUUID(), fftypes.NewRandB32(), fftypes.Now()))
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetGroupByHash(context.Background(), groupID)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupsBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.GroupQueryFactory.NewFilter(context.Background()).Eq("hash", map[bool]bool{true: false})
	_, err := s.GetGroups(context.Background(), f)
	assert.Regexp(t, "FF10149.*hash", err)
}

func TestGetGroupsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.GroupQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, err := s.GetGroups(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupsReadGroupFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"hash"}).AddRow("only one"))
	f := database.GroupQueryFactory.NewFilter(context.Background()).Eq("hash", "")
	_, err := s.GetGroups(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupsLoadMembersFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows(groupColumns).
		AddRow(nil, "ns1", "group1", fftypes.NewUUID(), fftypes.NewRandB32(), fftypes.Now()))
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.GroupQueryFactory.NewFilter(context.Background()).Gt("created", "0")
	_, err := s.GetGroups(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.GroupQueryFactory.NewUpdate(context.Background()).Set("hash", "anything")
	err := s.UpdateGroup(context.Background(), fftypes.NewRandB32(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestGroupUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.GroupQueryFactory.NewUpdate(context.Background()).Set("hash", map[bool]bool{true: false})
	err := s.UpdateGroup(context.Background(), fftypes.NewRandB32(), u)
	assert.Regexp(t, "FF10149.*hash", err)
}

func TestGroupsUpdateBuildFilterFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	f := database.GroupQueryFactory.NewFilter(context.Background()).Eq("hash", map[bool]bool{true: false})
	u := database.GroupQueryFactory.NewUpdate(context.Background()).Set("description", "my desc")
	err := s.UpdateGroups(context.Background(), f, u)
	assert.Regexp(t, "FF10149.*hash", err)
}

func TestGroupUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.GroupQueryFactory.NewUpdate(context.Background()).Set("description", fftypes.NewUUID())
	err := s.UpdateGroup(context.Background(), fftypes.NewRandB32(), u)
	assert.Regexp(t, "FF10117", err)
}
