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

func TestOrganizationsE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new organization entry
	organization := &fftypes.Organization{
		ID:       fftypes.NewUUID(),
		Message:  fftypes.NewUUID(),
		Name:     "org1",
		Identity: "0x12345",
		Created:  fftypes.Now(),
	}
	err := s.UpsertOrganization(ctx, organization, true)
	assert.NoError(t, err)

	// Check we get the exact same organization back
	organizationRead, err := s.GetOrganizationByIdentity(ctx, organization.Identity)
	assert.NoError(t, err)
	assert.NotNil(t, organizationRead)
	organizationJson, _ := json.Marshal(&organization)
	organizationReadJson, _ := json.Marshal(&organizationRead)
	assert.Equal(t, string(organizationJson), string(organizationReadJson))

	// Rejects attempt to update ID
	err = s.UpsertOrganization(context.Background(), &fftypes.Organization{
		ID:       fftypes.NewUUID(),
		Identity: "0x12345",
	}, true)
	assert.Equal(t, database.IDMismatch, err)

	// Update the organization (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	organizationUpdated := &fftypes.Organization{
		ID:          nil, // as long as we don't specify one we're fine
		Message:     fftypes.NewUUID(),
		Name:        "org1",
		Parent:      "0x23456",
		Identity:    "0x12345",
		Description: "organization1",
		Profile:     fftypes.JSONObject{"some": "info"},
		Created:     fftypes.Now(),
	}
	err = s.UpsertOrganization(context.Background(), organizationUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same data back - note the removal of one of the organization elements
	organizationRead, err = s.GetOrganizationByName(ctx, organization.Name)
	assert.NoError(t, err)
	organizationJson, _ = json.Marshal(&organizationUpdated)
	organizationReadJson, _ = json.Marshal(&organizationRead)
	assert.Equal(t, string(organizationJson), string(organizationReadJson))

	// Query back the organization
	fb := database.OrganizationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("description", string(organizationUpdated.Description)),
		fb.Eq("identity", organizationUpdated.Identity),
	)
	organizationRes, err := s.GetOrganizations(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(organizationRes))
	organizationReadJson, _ = json.Marshal(organizationRes[0])
	assert.Equal(t, string(organizationJson), string(organizationReadJson))

	// Update
	updateTime := fftypes.Now()
	up := database.OrganizationQueryFactory.NewUpdate(ctx).Set("created", updateTime)
	err = s.UpdateOrganization(ctx, organizationUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("identity", organizationUpdated.Identity),
		fb.Eq("created", updateTime.String()),
	)
	organizations, err := s.GetOrganizations(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(organizations))
}

func TestUpsertOrganizationFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertOrganization(context.Background(), &fftypes.Organization{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertOrganizationFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertOrganization(context.Background(), &fftypes.Organization{Identity: "id1"}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertOrganizationFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertOrganization(context.Background(), &fftypes.Organization{Identity: "id1"}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertOrganizationFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"identity"}).
		AddRow("id1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertOrganization(context.Background(), &fftypes.Organization{Identity: "id1"}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertOrganizationFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"identity"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertOrganization(context.Background(), &fftypes.Organization{Identity: "id1"}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetOrganizationByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationByNameSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetOrganizationByName(context.Background(), "org1")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationByIdentitySelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetOrganizationByIdentity(context.Background(), "id1")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"identity", "organization", "identity"}))
	msg, err := s.GetOrganizationByID(context.Background(), fftypes.NewUUID())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"identity"}).AddRow("only one"))
	_, err := s.GetOrganizationByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.OrganizationQueryFactory.NewFilter(context.Background()).Eq("identity", "")
	_, err := s.GetOrganizations(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.OrganizationQueryFactory.NewFilter(context.Background()).Eq("identity", map[bool]bool{true: false})
	_, err := s.GetOrganizations(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetOrganizationReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"identity"}).AddRow("only one"))
	f := database.OrganizationQueryFactory.NewFilter(context.Background()).Eq("identity", "")
	_, err := s.GetOrganizations(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrganizationUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.OrganizationQueryFactory.NewUpdate(context.Background()).Set("identity", "anything")
	err := s.UpdateOrganization(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestOrganizationUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.OrganizationQueryFactory.NewUpdate(context.Background()).Set("identity", map[bool]bool{true: false})
	err := s.UpdateOrganization(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*identity", err)
}

func TestOrganizationUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.OrganizationQueryFactory.NewUpdate(context.Background()).Set("identity", fftypes.NewUUID())
	err := s.UpdateOrganization(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}
