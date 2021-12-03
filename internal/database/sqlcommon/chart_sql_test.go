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
	expectedMetricsResult = []*fftypes.ChartHistogram{
		{
			Count:     "123",
			Timestamp: fftypes.UnixTime(1000000000),
		},
	}
	mockMetricInterval = []fftypes.ChartHistogramInterval{
		{
			StartTime: fftypes.UnixTime(1000000000),
			EndTime:   fftypes.UnixTime(1000000001),
		},
	}
	validCollections = []string{
		"events",
		"messages",
		"operations",
		"transactions",
	}
)

func TestGetMetricsInvalidCollectionName(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	_, err := s.GetChartHistogram(context.Background(), []fftypes.ChartHistogramInterval{}, database.CollectionName("abc"))
	assert.Regexp(t, "FF10122", err)
}

func TestGetMetricsValidCollectionName(t *testing.T) {
	for i := range validCollections {
		s, mock := newMockProvider().init()
		mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"case_0"}).AddRow("123"))

		metrics, err := s.GetChartHistogram(context.Background(), mockMetricInterval, database.CollectionName(validCollections[i]))

		assert.NoError(t, err)
		assert.Equal(t, metrics, expectedMetricsResult)
		assert.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestGetMetricsQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT *").WillReturnError(fmt.Errorf("pop"))

	_, err := s.GetChartHistogram(context.Background(), mockMetricInterval, database.CollectionName("messages"))
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMetricsFailNoRows(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"case_0"}))

	_, err := s.GetChartHistogram(context.Background(), mockMetricInterval, database.CollectionName("messages"))
	assert.Regexp(t, "FF10300", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMetricsScanFailTooManyCols(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"case_0", "unexpected_column"}).AddRow("one", "two"))

	_, err := s.GetChartHistogram(context.Background(), mockMetricInterval, database.CollectionName("messages"))
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMetricsSuccess(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"case_0"}).AddRow("123"))

	metrics, err := s.GetChartHistogram(context.Background(), mockMetricInterval, database.CollectionName("messages"))

	assert.NoError(t, err)
	assert.Equal(t, metrics, expectedMetricsResult)
	assert.NoError(t, mock.ExpectationsWereMet())
}
