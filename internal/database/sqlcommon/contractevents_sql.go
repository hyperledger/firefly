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

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
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
		"timestamp",
	}
	contractEventFilterFieldMap = map[string]string{
		"subscription": "subscription_id",
	}
)

func (s *SQLCommon) InsertContractEvent(ctx context.Context, event *fftypes.ContractEvent) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	if event.Sequence, err = s.insertTx(ctx, tx,
		sq.Insert("contractevents").
			Columns(contractEventColumns...).
			Values(
				event.ID,
				event.Namespace,
				event.Subscription,
				event.Name,
				event.Outputs,
				event.Info,
				event.Timestamp,
			),
		func() {
			s.callbacks.OrderedUUIDCollectionNSEvent(database.CollectionContractEvents, fftypes.ChangeEventTypeCreated, event.Namespace, event.ID, event.Sequence)
		},
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
		&event.Timestamp,
		// Must be added to the list of columns in all selects
		&event.Sequence,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contractevents")
	}
	return &event, nil
}

func (s *SQLCommon) getContractEventPred(ctx context.Context, desc string, pred interface{}) (*fftypes.ContractEvent, error) {
	cols := append([]string{}, contractEventColumns...)
	cols = append(cols, sequenceColumn)

	rows, _, err := s.query(ctx,
		sq.Select(cols...).
			From("contractevents").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Contract event '%s' not found", desc)
		return nil, nil
	}

	event, err := s.contractEventResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *SQLCommon) GetContractEventByID(ctx context.Context, id *fftypes.UUID) (*fftypes.ContractEvent, error) {
	return s.getContractEventPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetContractEvents(ctx context.Context, filter database.Filter) ([]*fftypes.ContractEvent, *database.FilterResult, error) {
	cols := append([]string{}, contractEventColumns...)
	cols = append(cols, sequenceColumn)

	query, fop, fi, err := s.filterSelect(ctx, "",
		sq.Select(cols...).From("contractevents"),
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
