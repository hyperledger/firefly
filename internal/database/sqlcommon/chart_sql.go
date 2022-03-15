// Copyright © 2022 Kaleido, Inc.
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
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (s *SQLCommon) getCaseQueries(ns string, dataTypes []string, interval fftypes.ChartHistogramInterval, typeColName string) (caseQueries []sq.CaseBuilder) {
	for _, dataType := range dataTypes {
		caseQueries = append(caseQueries, sq.Case().
			When(
				sq.And{
					sq.GtOrEq{"created": interval.StartTime},
					sq.Lt{"created": interval.EndTime},
					sq.Eq{typeColName: dataType},
					sq.Eq{"namespace": ns},
				},
				"1",
			).
			Else("0"))
	}

	return caseQueries
}

func (s *SQLCommon) getTableNameFromCollection(ctx context.Context, collection database.CollectionName) (tableName string, fieldMap map[string]string, err error) {
	switch collection {
	case database.CollectionName(database.CollectionMessages):
		return "messages", msgFilterFieldMap, nil
	case database.CollectionName(database.CollectionTransactions):
		return "transactions", transactionFilterFieldMap, nil
	case database.CollectionName(database.CollectionOperations):
		return "operations", opFilterFieldMap, nil
	case database.CollectionName(database.CollectionEvents):
		return "events", eventFilterFieldMap, nil
	case database.CollectionName(database.CollectionTokenTransfers):
		return "tokentransfers", tokenTransferFilterFieldMap, nil
	default:
		return "", nil, i18n.NewError(ctx, i18n.MsgUnsupportedCollection, collection)
	}
}

func (s *SQLCommon) getDistinctTypesFromTable(ctx context.Context, tableName string, fieldMap map[string]string) ([]string, error) {
	qb := sq.Select(fieldMap["type"]).Distinct().From(tableName)

	rows, _, err := s.query(ctx, qb.From(tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dataTypes []string
	for rows.Next() {
		var dataType string
		err := rows.Scan(&dataType)
		if err != nil {
			return []string{}, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, tableName)
		}
		dataTypes = append(dataTypes, dataType)
	}
	rows.Close()

	return dataTypes, nil
}

func (s *SQLCommon) histogramResult(ctx context.Context, rows *sql.Rows, cols []*fftypes.ChartHistogramType, tableName string) ([]*fftypes.ChartHistogramType, error) {
	results := []interface{}{}

	for i := range cols {
		results = append(results, &cols[i].Count)
	}
	err := rows.Scan(results...)
	if err != nil {
		return nil, i18n.NewError(ctx, i18n.MsgDBReadErr, tableName)
	}

	return cols, nil
}

func (s *SQLCommon) getBucketTotal(typeBuckets []*fftypes.ChartHistogramType) (string, error) {
	total := 0
	for _, typeBucket := range typeBuckets {
		typeBucketInt, err := strconv.Atoi(typeBucket.Count)
		if err != nil {
			return "", err
		}
		total += typeBucketInt
	}
	return strconv.Itoa(total), nil
}

func (s *SQLCommon) GetChartHistogram(ctx context.Context, ns string, intervals []fftypes.ChartHistogramInterval, collection database.CollectionName) (histogramList []*fftypes.ChartHistogram, err error) {
	tableName, fieldMap, err := s.getTableNameFromCollection(ctx, collection)
	if err != nil {
		return nil, err
	}

	dataTypes, err := s.getDistinctTypesFromTable(ctx, tableName, fieldMap)
	if err != nil {
		return nil, err
	}

	for _, interval := range intervals {
		qb := sq.Select()
		histogramTypes := make([]*fftypes.ChartHistogramType, 0)

		for i, caseQuery := range s.getCaseQueries(ns, dataTypes, interval, fieldMap["type"]) {
			query, args, _ := caseQuery.ToSql()
			histogramTypes = append(histogramTypes, &fftypes.ChartHistogramType{
				Count: "",
				Type:  dataTypes[i],
			})

			qb = qb.Column(sq.Alias(sq.Expr("SUM("+query+")", args...), fmt.Sprintf("case_%d", i)))
		}

		rows, _, err := s.query(ctx, qb.From(tableName))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		if rows.Next() {
			hist, err := s.histogramResult(ctx, rows, histogramTypes, tableName)
			rows.Close()
			if err != nil {
				return nil, err
			}

			total, err := s.getBucketTotal(hist)
			if err != nil {
				return nil, err
			}

			histogramList = append(histogramList, &fftypes.ChartHistogram{
				Count:     total,
				Timestamp: interval.StartTime,
				Types:     hist,
			})
		} else {
			histogramList = append(histogramList, &fftypes.ChartHistogram{
				Count:     "0",
				Timestamp: interval.StartTime,
				Types:     make([]*fftypes.ChartHistogramType, 0),
			})
		}
	}

	return histogramList, nil
}
