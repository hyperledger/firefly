// Copyright © 2021 Kaleido, Inc.
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
	"github.com/kaleido-io/firefly/internal/database"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/stretchr/testify/assert"
)

func TestNamespacesE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s := &SQLCommon{}
	ctx := context.Background()
	InitSQLCommon(ctx, s, ensureTestDB(t), nil, &database.Capabilities{}, testSQLOptions())

	// Create a new namespace entry
	namespace := &fftypes.Namespace{
		ID:      fftypes.NewUUID(),
		Type:    fftypes.NamespaceTypeStaticLocal,
		Name:    "namespace1",
		Created: fftypes.Now(),
	}
	err := s.UpsertNamespace(ctx, namespace)
	assert.NoError(t, err)

	// Check we get the exact same namespace back
	namespaceRead, err := s.GetNamespace(ctx, namespace.Name)
	assert.NoError(t, err)
	assert.NotNil(t, namespaceRead)
	namespaceJson, _ := json.Marshal(&namespace)
	namespaceReadJson, _ := json.Marshal(&namespaceRead)
	assert.Equal(t, string(namespaceJson), string(namespaceReadJson))

	// Update the namespace (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	namespaceUpdated := &fftypes.Namespace{
		ID:          fftypes.NewUUID(), // with namespaces we query them always by name, the id is secondary
		Type:        fftypes.NamespaceTypeStaticBroadcast,
		Name:        "namespace1",
		Description: "description1",
		Created:     fftypes.Now(),
		Confirmed:   fftypes.Now(),
	}
	err = s.UpsertNamespace(context.Background(), namespaceUpdated)
	assert.NoError(t, err)

	// Check we get the exact same data back - note the removal of one of the namespace elements
	namespaceRead, err = s.GetNamespace(ctx, namespace.Name)
	assert.NoError(t, err)
	namespaceJson, _ = json.Marshal(&namespaceUpdated)
	namespaceReadJson, _ = json.Marshal(&namespaceRead)
	assert.Equal(t, string(namespaceJson), string(namespaceReadJson))

	// Query back the namespace
	fb := database.NamespaceQueryFactory.NewFilter(ctx, 0)
	filter := fb.And(
		fb.Eq("type", string(namespaceUpdated.Type)),
		fb.Eq("name", namespaceUpdated.Name),
	)
	namespaceRes, err := s.GetNamespaces(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(namespaceRes))
	namespaceReadJson, _ = json.Marshal(namespaceRes[0])
	assert.Equal(t, string(namespaceJson), string(namespaceReadJson))

	// Update
	updateTime := fftypes.Now()
	up := database.NamespaceQueryFactory.NewUpdate(ctx).Set("confirmed", updateTime)
	err = s.UpdateNamespace(ctx, namespaceUpdated.Name, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("name", namespaceUpdated.Name),
		fb.Eq("confirmed", updateTime.String()),
	)
	namespaces, err := s.GetNamespaces(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(namespaces))
}

func TestUpsertNamespaceFailBegin(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{})
	assert.Regexp(t, "FF10114", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNamespaceFailSelect(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{Name: "name1"})
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNamespaceFailInsert(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{Name: "name1"})
	assert.Regexp(t, "FF10116", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNamespaceFailUpdate(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}).
		AddRow("name1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{Name: "name1"})
	assert.Regexp(t, "FF10117", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNamespaceFailCommit(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{Name: "name1"})
	assert.Regexp(t, "FF10119", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceByIdSelectFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetNamespace(context.Background(), "name1")
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceByIdNotFound(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"ntype", "namespace", "name"}))
	msg, err := s.GetNamespace(context.Background(), "name1")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceByIdScanFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"ntype"}).AddRow("only one"))
	_, err := s.GetNamespace(context.Background(), "name1")
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.NamespaceQueryFactory.NewFilter(context.Background(), 0).Eq("type", "")
	_, err := s.GetNamespaces(context.Background(), f)
	assert.Regexp(t, "FF10115", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceBuildQueryFail(t *testing.T) {
	s, _ := getMockDB()
	f := database.NamespaceQueryFactory.NewFilter(context.Background(), 0).Eq("type", map[bool]bool{true: false})
	_, err := s.GetNamespaces(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err.Error())
}

func TestGetNamespaceReadMessageFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"ntype"}).AddRow("only one"))
	f := database.NamespaceQueryFactory.NewFilter(context.Background(), 0).Eq("type", "")
	_, err := s.GetNamespaces(context.Background(), f)
	assert.Regexp(t, "FF10121", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNamespaceUpdateBeginFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.NamespaceQueryFactory.NewUpdate(context.Background()).Set("name", "anything")
	err := s.UpdateNamespace(context.Background(), "name1", u)
	assert.Regexp(t, "FF10114", err.Error())
}

func TestNamespaceUpdateBuildQueryFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	u := database.NamespaceQueryFactory.NewUpdate(context.Background()).Set("name", map[bool]bool{true: false})
	err := s.UpdateNamespace(context.Background(), "name1", u)
	assert.Regexp(t, "FF10149.*name", err.Error())
}

func TestNamespaceUpdateFail(t *testing.T) {
	s, mock := getMockDB()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.NamespaceQueryFactory.NewUpdate(context.Background()).Set("name", fftypes.NewUUID())
	err := s.UpdateNamespace(context.Background(), "name1", u)
	assert.Regexp(t, "FF10117", err.Error())
}
