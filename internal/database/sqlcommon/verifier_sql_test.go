// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.verifier/licenses/LICENSE-2.0
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
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestVerifiersE2EWithDB(t *testing.T) {
	log.SetLevel("debug")

	s, cleanup := newSQLiteTestProvider(t)
	defer cleanup()
	ctx := context.Background()

	// Create a new verifier entry
	verifierID := fftypes.NewUUID()
	verifier := &fftypes.Verifier{
		ID:        verifierID,
		Type:      fftypes.VerifierTypeEthAddress,
		Identity:  fftypes.NewUUID(),
		Namespace: "ns1",
		Value:     "0x12345",
	}

	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionVerifiers, fftypes.ChangeEventTypeCreated, "ns1", verifierID).Return()
	s.callbacks.On("UUIDCollectionNSEvent", database.CollectionVerifiers, fftypes.ChangeEventTypeUpdated, "ns2", verifierID).Return()

	err := s.UpsertVerifier(ctx, verifier, database.UpsertOptimizationNew)
	assert.NoError(t, err)

	// Check we get the exact same verifier back
	verifierRead, err := s.GetVerifierByID(ctx, verifier.ID)
	assert.NoError(t, err)
	assert.NotNil(t, verifierRead)
	verifierJson, _ := json.Marshal(&verifier)
	verifierReadJson, _ := json.Marshal(&verifierRead)
	assert.Equal(t, string(verifierJson), string(verifierReadJson))

	// Update the verifier (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	verifierUpdated := &fftypes.Verifier{
		ID:        verifierID,
		Type:      fftypes.VerifierTypeFFDXPeerID,
		Identity:  fftypes.NewUUID(),
		Namespace: "ns2",
		Value:     "peer1",
		Created:   verifier.Created,
	}
	err = s.UpsertVerifier(context.Background(), verifierUpdated, database.UpsertOptimizationExisting)
	assert.NoError(t, err)

	// Check we get the exact same data back - note the removal of one of the verifier elements
	verifierRead, err = s.GetVerifierByValue(ctx, verifierUpdated.Type, verifierUpdated.Namespace, verifierUpdated.Value)
	assert.NoError(t, err)
	verifierJson, _ = json.Marshal(&verifierUpdated)
	verifierReadJson, _ = json.Marshal(&verifierRead)
	assert.Equal(t, string(verifierJson), string(verifierReadJson))

	// Query back the verifier
	fb := database.VerifierQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("value", string(verifierUpdated.Value)),
		fb.Eq("namespace", verifierUpdated.Namespace),
	)
	verifierRes, res, err := s.GetVerifiers(ctx, filter.Count(true))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(verifierRes))
	assert.Equal(t, int64(1), *res.TotalCount)
	verifierReadJson, _ = json.Marshal(verifierRes[0])
	assert.Equal(t, string(verifierJson), string(verifierReadJson))

	// Update
	updateTime := fftypes.Now()
	up := database.VerifierQueryFactory.NewUpdate(ctx).Set("created", updateTime)
	err = s.UpdateVerifier(ctx, verifierUpdated.ID, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("value", verifierUpdated.Value),
		fb.Eq("created", updateTime.String()),
	)
	verifiers, _, err := s.GetVerifiers(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(verifiers))

	s.callbacks.AssertExpectations(t)
}

func TestUpsertVerifierFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertVerifier(context.Background(), &fftypes.Verifier{}, database.UpsertOptimizationSkip)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertVerifierFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertVerifier(context.Background(), &fftypes.Verifier{ID: fftypes.NewUUID()}, database.UpsertOptimizationSkip)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertVerifierFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertVerifier(context.Background(), &fftypes.Verifier{ID: fftypes.NewUUID()}, database.UpsertOptimizationSkip)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertVerifierFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"verifier"}).
		AddRow("id1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertVerifier(context.Background(), &fftypes.Verifier{ID: fftypes.NewUUID()}, database.UpsertOptimizationSkip)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertVerifierFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"verifier"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertVerifier(context.Background(), &fftypes.Verifier{ID: fftypes.NewUUID()}, database.UpsertOptimizationSkip)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVerifierByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetVerifierByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVerifierByNameSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetVerifierByValue(context.Background(), fftypes.VerifierTypeEthAddress, "ff_system", "0x12345")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVerifierByVerifierSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetVerifierByValue(context.Background(), fftypes.VerifierTypeEthAddress, "ff_system", "0x12345")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVerifierByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"verifier", "verifier", "verifier"}))
	msg, err := s.GetVerifierByID(context.Background(), fftypes.NewUUID())
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVerifierByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"verifier"}).AddRow("only one"))
	_, err := s.GetVerifierByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVerifierQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.VerifierQueryFactory.NewFilter(context.Background()).Eq("value", "")
	_, _, err := s.GetVerifiers(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVerifierBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.VerifierQueryFactory.NewFilter(context.Background()).Eq("value", map[bool]bool{true: false})
	_, _, err := s.GetVerifiers(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetVerifierReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("only one"))
	f := database.VerifierQueryFactory.NewFilter(context.Background()).Eq("value", "")
	_, _, err := s.GetVerifiers(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifierUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.VerifierQueryFactory.NewUpdate(context.Background()).Set("value", "anything")
	err := s.UpdateVerifier(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10114", err)
}

func TestVerifierUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.VerifierQueryFactory.NewUpdate(context.Background()).Set("value", map[bool]bool{true: false})
	err := s.UpdateVerifier(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10149.*value", err)
}

func TestVerifierUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.VerifierQueryFactory.NewUpdate(context.Background()).Set("value", fftypes.NewUUID())
	err := s.UpdateVerifier(context.Background(), fftypes.NewUUID(), u)
	assert.Regexp(t, "FF10117", err)
}
