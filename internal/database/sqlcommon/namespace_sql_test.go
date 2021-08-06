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
	"github.com/stretchr/testify/mock"
)

func TestNamespacesE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new namespace entry
	namespace := &fftypes.Namespace{
		ID:      nil, // generated for us
		Message: fftypes.NewUUID(),
		Type:    fftypes.NamespaceTypeLocal,
		Name:    "namespace1",
		Created: fftypes.Now(),
	}

	s.callbacks.On("UUIDCollectionEvent", database.CollectionNamespaces, fftypes.ChangeEventTypeCreated, mock.Anything, mock.Anything).Return()

	err := s.UpsertNamespace(ctx, namespace, true)
	assert.NoError(t, err)

	// Check we get the exact same namespace back
	namespaceRead, err := s.GetNamespace(ctx, namespace.Name)
	assert.NoError(t, err)
	assert.NotNil(t, namespaceRead)
	namespaceJson, _ := json.Marshal(&namespace)
	namespaceReadJson, _ := json.Marshal(&namespaceRead)
	assert.Equal(t, string(namespaceJson), string(namespaceReadJson))

	// Rejects attempt to update ID
	err = s.UpsertNamespace(context.Background(), &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "namespace1",
	}, true)
	assert.Equal(t, database.IDMismatch, err)

	// Update the namespace (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	namespaceUpdated := &fftypes.Namespace{
		ID:          nil, // as long as we don't specify one we're fine
		Message:     fftypes.NewUUID(),
		Type:        fftypes.NamespaceTypeBroadcast,
		Name:        "namespace1",
		Description: "description1",
		Created:     fftypes.Now(),
	}
	s.callbacks.On("UUIDCollectionEvent", database.CollectionNamespaces, fftypes.ChangeEventTypeUpdated, namespace.ID, mock.Anything).Return()
	err = s.UpsertNamespace(context.Background(), namespaceUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same data back - note the removal of one of the namespace elements
	namespaceRead, err = s.GetNamespace(ctx, namespace.Name)
	assert.NoError(t, err)
	namespaceJson, _ = json.Marshal(&namespaceUpdated)
	namespaceReadJson, _ = json.Marshal(&namespaceRead)
	assert.Equal(t, string(namespaceJson), string(namespaceReadJson))

	// Query back the namespace
	fb := database.NamespaceQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("type", string(namespaceUpdated.Type)),
		fb.Eq("name", namespaceUpdated.Name),
	)
	namespaceRes, res, err := s.GetNamespaces(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(namespaceRes))
	assert.Equal(t, int64(1), *res.TotalCount)
	namespaceReadJson, _ = json.Marshal(namespaceRes[0])
	assert.Equal(t, string(namespaceJson), string(namespaceReadJson))

	// Delete
	s.callbacks.On("UUIDCollectionEvent", database.CollectionNamespaces, fftypes.ChangeEventTypeDeleted, namespace.ID, mock.Anything).Return()
	err = s.DeleteNamespace(ctx, namespaceUpdated.ID)
	assert.NoError(t, err)
	namespaces, _, err := s.GetNamespaces(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(namespaces))

	s.callbacks.AssertExpectations(t)
}

func TestUpsertNamespaceFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNamespaceFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{Name: "name1"}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNamespaceFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{Name: "name1"}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNamespaceFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}).
		AddRow("name1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{Name: "name1"}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNamespaceFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertNamespace(context.Background(), &fftypes.Namespace{Name: "name1"}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetNamespace(context.Background(), "name1")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"ntype", "namespace", "name"}))
	msg, err := s.GetNamespace(context.Background(), "name1")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"ntype"}).AddRow("only one"))
	_, err := s.GetNamespace(context.Background(), "name1")
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.NamespaceQueryFactory.NewFilter(context.Background()).Eq("type", "")
	_, _, err := s.GetNamespaces(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNamespaceBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.NamespaceQueryFactory.NewFilter(context.Background()).Eq("type", map[bool]bool{true: false})
	_, _, err := s.GetNamespaces(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetNamespaceReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"ntype"}).AddRow("only one"))
	f := database.NamespaceQueryFactory.NewFilter(context.Background()).Eq("type", "")
	_, _, err := s.GetNamespaces(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNamespaceDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteNamespace(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10114", err)
}

func TestNamespaceDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteNamespace(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10118", err)
}
