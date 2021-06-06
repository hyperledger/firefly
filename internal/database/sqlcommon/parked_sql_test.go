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
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestParkedsE2EWithDB(t *testing.T) {
	log.SetLevel("trace")

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new parked entry
	parked := &fftypes.Parked{
		Pin:     fftypes.NewRandB32(),
		Batch:   fftypes.NewUUID(),
		Created: fftypes.Now(),
	}
	err := s.InsertParked(ctx, parked)
	assert.NoError(t, err)

	// Query back the parked
	fb := database.ParkedQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("pin", parked.Pin),
		fb.Eq("ledger", parked.Ledger),
		fb.Eq("batch", parked.Batch),
		fb.Gt("created", 0),
	)
	parkedRes, err := s.GetParked(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(parkedRes))

	// Test delete
	err = s.DeleteParked(ctx, parkedRes[0].Sequence)
	assert.NoError(t, err)
	p, err := s.GetParked(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(p))

}

func TestInsertParkedFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertParked(context.Background(), &fftypes.Parked{})
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertParkedFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.InsertParked(context.Background(), &fftypes.Parked{Sequence: 12345})
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertParkedFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.InsertParked(context.Background(), &fftypes.Parked{Sequence: 12345})
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetParkedQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.ParkedQueryFactory.NewFilter(context.Background()).Eq("pin", "")
	_, err := s.GetParked(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetParkedBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.ParkedQueryFactory.NewFilter(context.Background()).Eq("pin", map[bool]bool{true: false})
	_, err := s.GetParked(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetParkedReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"pin"}).AddRow("only one"))
	f := database.ParkedQueryFactory.NewFilter(context.Background()).Eq("pin", "")
	_, err := s.GetParked(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParkedDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteParked(context.Background(), 12345)
	assert.Regexp(t, "FF10114", err)
}

func TestParkedDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.DeleteParked(context.Background(), 12345)
	assert.Regexp(t, "FF10118", err)
}
