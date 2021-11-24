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
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (s *SQLCommon) metricResult(ctx context.Context, row *sql.Rows) (*fftypes.MetricCount, error) {
	var batch fftypes.MetricCount
	err := row.Scan(
		&batch.Count,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "metrics")
	}
	return &batch, nil
}

func (s *SQLCommon) GetMetrics(ctx context.Context, interval *fftypes.MetricInterval, tableName string) (count string, err error) {
	rows, _, err := s.query(ctx,
		sq.Select("count(*)").
			From(tableName).
			Where(sq.GtOrEq{"created": interval.StartTime}).
			Where(sq.Lt{"created": interval.EndTime}),
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Error fetching count from '%s' table", tableName)
		return "", nil
	}

	metric, err := s.metricResult(ctx, rows)
	if err != nil {
		return "", err
	}

	return metric.Count, err
}
