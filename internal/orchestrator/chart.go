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

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (or *orchestrator) getHistogramIntervals(startTime int64, endTime int64, numBuckets int64) (intervals []fftypes.ChartHistogramInterval) {
	timeIntervalLength := (endTime - startTime) / numBuckets

	for i := startTime; i < endTime; i += timeIntervalLength {
		intervals = append(intervals, fftypes.ChartHistogramInterval{
			StartTime: fftypes.UnixTime(i),
			EndTime:   fftypes.UnixTime(i + timeIntervalLength),
		})
	}

	return intervals
}

func (or *orchestrator) GetChartHistogram(ctx context.Context, ns string, startTime int64, endTime int64, buckets int64, collection database.CollectionName) ([]*fftypes.ChartHistogram, error) {
	if buckets > fftypes.ChartHistogramMaxBuckets || buckets < fftypes.ChartHistogramMinBuckets {
		return nil, i18n.NewError(ctx, i18n.MsgInvalidNumberOfIntervals, fftypes.ChartHistogramMinBuckets, fftypes.ChartHistogramMaxBuckets)
	}
	if startTime > endTime {
		return nil, i18n.NewError(ctx, i18n.MsgHistogramInvalidTimes)
	}

	intervals := or.getHistogramIntervals(startTime, endTime, buckets)

	histogram, err := or.database.GetChartHistogram(ctx, ns, intervals, collection)
	if err != nil {
		return nil, err
	}

	return histogram, nil
}
