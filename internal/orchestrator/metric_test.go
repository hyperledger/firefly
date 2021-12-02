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

package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func makeTestIntervals(start int, numIntervals int) (intervals []fftypes.MetricInterval) {
	for i := 0; i < numIntervals; i++ {
		intervals = append(intervals, fftypes.MetricInterval{
			StartTime: fftypes.UnixTime(int64(start + i)),
			EndTime:   fftypes.UnixTime(int64(start + i + 1)),
		})
	}
	return intervals
}

func TestGetMetricsBadIntervalMin(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetMetrics(context.Background(), "ns1", 1234567890, 9876543210, fftypes.MetricMinBuckets-1, database.CollectionName("test"))
	assert.Regexp(t, "FF10298", err)
}

func TestGetMetricsBadIntervalMax(t *testing.T) {
	or := newTestOrchestrator()
	_, err := or.GetMetrics(context.Background(), "ns1", 1234567890, 9876543210, fftypes.MetricMaxBuckets+1, database.CollectionName("test"))
	assert.Regexp(t, "FF10298", err)
}

func TestGetMetricsFailDB(t *testing.T) {
	or := newTestOrchestrator()
	intervals := makeTestIntervals(1000000000, 10)
	or.mdi.On("GetMetrics", mock.Anything, intervals, database.CollectionName("test")).Return(nil, fmt.Errorf("pop"))
	_, err := or.GetMetrics(context.Background(), "ns1", 1000000000, 1000000010, 10, database.CollectionName("test"))
	assert.EqualError(t, err, "pop")
}

func TestGetMetricsSuccess(t *testing.T) {
	or := newTestOrchestrator()
	intervals := makeTestIntervals(1000000000, 10)
	mockMetrics := []*fftypes.Metric{}

	or.mdi.On("GetMetrics", mock.Anything, intervals, database.CollectionName("test")).Return(mockMetrics, nil)
	_, err := or.GetMetrics(context.Background(), "ns1", 1000000000, 1000000010, 10, database.CollectionName("test"))
	assert.NoError(t, err)
}
