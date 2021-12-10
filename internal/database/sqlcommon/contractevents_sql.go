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
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	contractEventColumns = []string{
		"id",
		"namespace",
		"subscription_id",
		"name",
		"outputs",
		"info",
	}
	contractEventFilterFieldMap = map[string]string{
		"subscriptionid": "subscription_id",
	}
)

func (s *SQLCommon) InsertContractEvent(ctx context.Context, event *fftypes.ContractEvent) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	if _, err = s.insertTx(ctx, tx,
		sq.Insert("contractevents").
			Columns(contractEventColumns...).
			Values(
				event.ID,
				event.Namespace,
				event.Subscription,
				event.Name,
				event.Outputs,
				event.Info,
			),
		nil, // no change event
	); err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractEventResult(ctx context.Context, row *sql.Rows) (*fftypes.ContractEvent, error) {
	var event fftypes.ContractEvent
	err := row.Scan(
		&event.ID,
		&event.Namespace,
		&event.Subscription,
		&event.Name,
		&event.Outputs,
		&event.Info,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contractevents")
	}
	return &event, nil
}

func (s *SQLCommon) GetContractEvents(ctx context.Context, filter database.Filter) ([]*fftypes.ContractEvent, *database.FilterResult, error) {
	query, fop, fi, err := s.filterSelect(ctx, "",
		sq.Select(contractEventColumns...).From("contractevents"),
		filter, contractEventFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	events := []*fftypes.ContractEvent{}
	for rows.Next() {
		event, err := s.contractEventResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, event)
	}

	return events, s.queryRes(ctx, tx, "contractevents", fop, fi), err
}
