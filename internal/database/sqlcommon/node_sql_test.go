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

func TestNodesE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new node entry
	node := &fftypes.Node{
		ID:      fftypes.NewUUID(),
		Message: fftypes.NewUUID(),
		Owner:   "0x23456",
		Name:    "node1",
		Created: fftypes.Now(),
	}
	err := s.UpsertNode(ctx, node, true)
	assert.NoError(t, err)

	// Check we get the exact same node back
	nodeRead, err := s.GetNode(ctx, node.Owner, node.Name)
	assert.NoError(t, err)
	assert.NotNil(t, nodeRead)
	nodeJson, _ := json.Marshal(&node)
	nodeReadJson, _ := json.Marshal(&nodeRead)
	assert.Equal(t, string(nodeJson), string(nodeReadJson))

	// Rejects attempt to update ID
	err = s.UpsertNode(context.Background(), &fftypes.Node{
		ID:    fftypes.NewUUID(),
		Owner: "0x23456",
		Name:  "node1",
	}, true)
	assert.Equal(t, database.IDMismatch, err)

	// Update the node (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	nodeUpdated := &fftypes.Node{
		ID:          nil, // as long as we don't specify one we're fine
		Message:     fftypes.NewUUID(),
		Owner:       "0x23456",
		Name:        "node1",
		Description: "node1",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
		Created: fftypes.Now(),
	}
	err = s.UpsertNode(context.Background(), nodeUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same data back - note the removal of one of the node elements
	nodeRead, err = s.GetNode(ctx, node.Owner, node.Name)
	assert.NoError(t, err)
	nodeJson, _ = json.Marshal(&nodeUpdated)
	nodeReadJson, _ = json.Marshal(&nodeRead)
	assert.Equal(t, string(nodeJson), string(nodeReadJson))

	// Query back the node
	fb := database.NodeQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("description", string(nodeUpdated.Description)),
		fb.Eq("name", nodeUpdated.Name),
	)
	nodeRes, err := s.GetNodes(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(nodeRes))
	nodeReadJson, _ = json.Marshal(nodeRes[0])
	assert.Equal(t, string(nodeJson), string(nodeReadJson))

	// Update
	updateTime := fftypes.Now()
	up := database.NodeQueryFactory.NewUpdate(ctx).Set("created", updateTime)
	err = s.UpdateNode(ctx, nodeUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("name", nodeUpdated.Name),
		fb.Eq("created", updateTime.String()),
	)
	nodes, err := s.GetNodes(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(nodes))
}

func TestUpsertNodeFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertNode(context.Background(), &fftypes.Node{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNodeFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNode(context.Background(), &fftypes.Node{Name: "node1"}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNodeFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNode(context.Background(), &fftypes.Node{Name: "node1"}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNodeFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}).
		AddRow("id1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertNode(context.Background(), &fftypes.Node{Name: "node1"}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertNodeFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertNode(context.Background(), &fftypes.Node{Name: "node1"}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNodeByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetNode(context.Background(), "owner1", "node1")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNodeByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name", "node", "name"}))
	msg, err := s.GetNode(context.Background(), "owner1", "node1")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNodeByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("only one"))
	_, err := s.GetNodeByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNodeQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.NodeQueryFactory.NewFilter(context.Background()).Eq("name", "")
	_, err := s.GetNodes(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNodeBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.NodeQueryFactory.NewFilter(context.Background()).Eq("name", map[bool]bool{true: false})
	_, err := s.GetNodes(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetNodeReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("only one"))
	f := database.NodeQueryFactory.NewFilter(context.Background()).Eq("name", "")
	_, err := s.GetNodes(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNodeUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.NodeQueryFactory.NewUpdate(context.Background()).Set("name", "anything")
	err := s.UpdateNode(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestNodeUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.NodeQueryFactory.NewUpdate(context.Background()).Set("name", map[bool]bool{true: false})
	err := s.UpdateNode(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*name", err)
}

func TestNodeUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.NodeQueryFactory.NewUpdate(context.Background()).Set("name", fftypes.NewUUID())
	err := s.UpdateNode(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}
