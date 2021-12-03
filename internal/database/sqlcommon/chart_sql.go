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
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (s *SQLCommon) getCaseQueries(intervals []fftypes.ChartHistogramInterval) (caseQueries []sq.CaseBuilder) {
	for _, interval := range intervals {
		caseQueries = append(caseQueries, sq.Case().
			When(
				sq.And{
					sq.GtOrEq{"created": interval.StartTime},
					sq.Lt{"created": interval.EndTime},
				},
				"1",
			).
			Else("0"))
	}

	return caseQueries
}

func (s *SQLCommon) getTableNameFromCollection(ctx context.Context, collection database.CollectionName) (tableName string, err error) {
	switch collection {
	case database.CollectionName(database.CollectionMessages):
		return "messages", nil
	case database.CollectionName(database.CollectionTransactions):
		return "transactions", nil
	case database.CollectionName(database.CollectionOperations):
		return "operations", nil
	case database.CollectionName(database.CollectionEvents):
		return "events", nil
	default:
		return "", i18n.NewError(ctx, i18n.MsgUnsupportedCollection, collection)
	}
}

func (s *SQLCommon) histogramResult(ctx context.Context, rows *sql.Rows, cols []interface{}) ([]interface{}, error) {
	results := []interface{}{}

	for i := range cols {
		results = append(results, &cols[i])
	}
	err := rows.Scan(results...)
	if err != nil {
		return nil, i18n.NewError(ctx, i18n.MsgDBReadErr, "histogram")
	}
	return cols, nil
}

func (s *SQLCommon) GetChartHistogram(ctx context.Context, intervals []fftypes.ChartHistogramInterval, collection database.CollectionName) (histogram []*fftypes.ChartHistogram, err error) {
	tableName, err := s.getTableNameFromCollection(ctx, collection)
	if err != nil {
		return nil, err
	}

	cols := []interface{}{}
	qb := sq.Select()

	for i, caseQuery := range s.getCaseQueries(intervals) {
		query, args, _ := caseQuery.ToSql()
		col := fmt.Sprintf("case_%d", i)
		cols = append(cols, "")

		qb = qb.Column(sq.Alias(sq.Expr("SUM("+query+")", args...), col))
	}

	rows, _, err := s.query(ctx, qb.From(tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return []*fftypes.ChartHistogram{}, nil
	}

	res, err := s.histogramResult(ctx, rows, cols)
	if err != nil {
		return nil, err
	}

	for i, interval := range res {
		histogram = append(histogram, &fftypes.ChartHistogram{
			Count:     fmt.Sprintf("%v", interval),
			Timestamp: intervals[i].StartTime,
		})
	}

	return histogram, nil
}
