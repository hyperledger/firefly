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
	"strconv"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (or *orchestrator) getMetricIntervals(ctx context.Context, startTime string, endTime string, numPeriods int64) ([]*fftypes.MetricInterval, error) {
	startInt, err := strconv.ParseInt(startTime, 10, 64)
	if err != nil {
		return nil, err
	}
	endInt, err := strconv.ParseInt(endTime, 10, 64)
	if err != nil {
		return nil, err
	}

	intervals := []*fftypes.MetricInterval{}

	timeIntervalLength := (endInt - startInt) / numPeriods

	for i := startInt; i < endInt; i += timeIntervalLength {
		intervals = append(intervals, &fftypes.MetricInterval{
			StartTime: fftypes.UnixTime(i),
			EndTime:   fftypes.UnixTime(i + timeIntervalLength),
		})
	}

	return intervals, nil
}

func (or *orchestrator) GetMetrics(ctx context.Context, ns string, startTime string, endTime string, periods string, tableName string) ([]*fftypes.Metric, error) {
	numPeriods, err := strconv.ParseInt(periods, 10, 64)
	if err != nil {
		return nil, err
	}
	if numPeriods > 100 {
		return nil, i18n.NewError(ctx, i18n.MsgInvalidNumberOfIntervals, "100")
	}
	metricIntervals, err := or.getMetricIntervals(ctx, startTime, endTime, numPeriods)
	if err != nil {
		return nil, err
	}

	metrics := []*fftypes.Metric{}

	for _, interval := range metricIntervals {
		count, err := or.database.GetMetrics(ctx, interval, tableName)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, &fftypes.Metric{
			Count:     count,
			Timestamp: interval.StartTime,
		})
		if err != nil {
			return nil, err
		}
	}

	return metrics, nil
}
