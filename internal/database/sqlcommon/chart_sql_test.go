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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

var (
	emptyHistogramResult = []*fftypes.ChartHistogram{
		{
			Count:     "0",
			Timestamp: fftypes.UnixTime(1000000000),
			Types:     make([]*fftypes.ChartHistogramType, 0),
		},
	}
	expectedHistogramResult = []*fftypes.ChartHistogram{
		{
			Count:     "10",
			Timestamp: fftypes.UnixTime(1000000000),
			Types: []*fftypes.ChartHistogramType{
				{
					Count: "5",
					Type:  "typeA",
				},
				{
					Count: "5",
					Type:  "typeB",
				},
			},
		},
	}
	expectedHistogramResultNoTypes = []*fftypes.ChartHistogram{
		{
			Count:     "10",
			Timestamp: fftypes.UnixTime(1000000000),
			Types:     make([]*fftypes.ChartHistogramType, 0),
		},
	}

	mockHistogramInterval = []fftypes.ChartHistogramInterval{
		{
			StartTime: fftypes.UnixTime(1000000000),
			EndTime:   fftypes.UnixTime(1000000001),
		},
	}
	validCollectionsWithTypes = []string{
		"events",
		"messages",
		"operations",
		"transactions",
		"tokentransfers",
	}
	validCollectionsNoTypes = []string{
		"blockchainevents",
	}
)

func TestGetChartHistogramInvalidCollectionName(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	_, err := s.GetChartHistogram(context.Background(), "ns1", []fftypes.ChartHistogramInterval{}, database.CollectionName("abc"))
	assert.Regexp(t, "FF10301", err)
}

func TestGetChartHistogramValidCollectionNameWithTypes(t *testing.T) {
	for i := range validCollectionsWithTypes {
		s, mock := newMockProvider().init()
		mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"timestamp", "type"}).
			AddRow(fftypes.UnixTime(1000000000), "typeA").
			AddRow(fftypes.UnixTime(1000000000), "typeA").
			AddRow(fftypes.UnixTime(1000000000), "typeA").
			AddRow(fftypes.UnixTime(1000000000), "typeA").
			AddRow(fftypes.UnixTime(1000000000), "typeA").
			AddRow(fftypes.UnixTime(1000000000), "typeB").
			AddRow(fftypes.UnixTime(1000000000), "typeB").
			AddRow(fftypes.UnixTime(1000000000), "typeB").
			AddRow(fftypes.UnixTime(1000000000), "typeB").
			AddRow(fftypes.UnixTime(1000000000), "typeB"))

		histogram, err := s.GetChartHistogram(context.Background(), "ns1", mockHistogramInterval, database.CollectionName(validCollectionsWithTypes[i]))

		assert.NoError(t, err)
		assert.Equal(t, histogram, expectedHistogramResult)
		assert.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestGetChartHistogramValidCollectionNameNoTypes(t *testing.T) {
	for i := range validCollectionsNoTypes {
		s, mock := newMockProvider().init()
		mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"timestamp"}).
			AddRow(fftypes.UnixTime(1000000000)).
			AddRow(fftypes.UnixTime(1000000000)).
			AddRow(fftypes.UnixTime(1000000000)).
			AddRow(fftypes.UnixTime(1000000000)).
			AddRow(fftypes.UnixTime(1000000000)).
			AddRow(fftypes.UnixTime(1000000000)).
			AddRow(fftypes.UnixTime(1000000000)).
			AddRow(fftypes.UnixTime(1000000000)).
			AddRow(fftypes.UnixTime(1000000000)).
			AddRow(fftypes.UnixTime(1000000000)))

		histogram, err := s.GetChartHistogram(context.Background(), "ns1", mockHistogramInterval, database.CollectionName(validCollectionsNoTypes[i]))
		assert.NoError(t, err)
		assert.Equal(t, expectedHistogramResultNoTypes, histogram)
		assert.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestGetChartHistogramsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT *").WillReturnError(fmt.Errorf("pop"))

	_, err := s.GetChartHistogram(context.Background(), "ns1", mockHistogramInterval, database.CollectionName("messages"))
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChartHistogramScanFailInvalidRowTypeOnlyTimestamp(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"timestamp"}).
		AddRow(nil))

	_, err := s.GetChartHistogram(context.Background(), "ns1", mockHistogramInterval, database.CollectionName("blockchainevents"))
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChartHistogramScanFailTooManyColsOnlyTimestamp(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"timestamp", "unexpected_col"}).
		AddRow(fftypes.UnixTime(1000000000), "test"))

	_, err := s.GetChartHistogram(context.Background(), "ns1", mockHistogramInterval, database.CollectionName("blockchainevents"))
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChartHistogramScanFailInvalidRowType(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"timestamp", "type"}).
		AddRow(fftypes.UnixTime(1000000000), nil))

	_, err := s.GetChartHistogram(context.Background(), "ns1", mockHistogramInterval, database.CollectionName("messages"))
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChartHistogramScanFailTooManyCols(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"timestamp", "type", "unexpected_col"}).
		AddRow(fftypes.UnixTime(1000000000), "typeA", "test"))

	_, err := s.GetChartHistogram(context.Background(), "ns1", mockHistogramInterval, database.CollectionName("messages"))
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChartHistogramSuccessNoRows(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"timestamp", "type"}))

	histogram, err := s.GetChartHistogram(context.Background(), "ns1", mockHistogramInterval, database.CollectionName("messages"))
	assert.NoError(t, err)
	assert.Equal(t, emptyHistogramResult, histogram)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChartHistogramSuccessNoRowsOnlyTimestamp(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"timestamp"}))

	histogram, err := s.GetChartHistogram(context.Background(), "ns1", mockHistogramInterval, database.CollectionName("blockchainevents"))
	assert.NoError(t, err)

	assert.Equal(t, emptyHistogramResult, histogram)
	assert.NoError(t, mock.ExpectationsWereMet())
}
