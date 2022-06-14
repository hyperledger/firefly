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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

var (
	ffiEventsColumns = []string{
		"id",
		"interface_id",
		"namespace",
		"name",
		"pathname",
		"description",
		"params",
		"details",
	}
	ffiEventFilterFieldMap = map[string]string{
		"interface": "interface_id",
	}
)

const ffieventsTable = "ffievents"

func (s *SQLCommon) UpsertFFIEvent(ctx context.Context, event *core.FFIEvent) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, ffieventsTable, tx,
		sq.Select("id").
			From(ffieventsTable).
			Where(sq.And{sq.Eq{"interface_id": event.Interface}, sq.Eq{"namespace": event.Namespace}, sq.Eq{"pathname": event.Pathname}}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if _, err = s.updateTx(ctx, ffieventsTable, tx,
			sq.Update(ffieventsTable).
				Set("params", event.Params).
				Where(sq.And{sq.Eq{"interface_id": event.Interface}, sq.Eq{"namespace": event.Namespace}, sq.Eq{"pathname": event.Pathname}}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionFFIEvents, core.ChangeEventTypeUpdated, event.Namespace, event.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, ffieventsTable, tx,
			sq.Insert(ffieventsTable).
				Columns(ffiEventsColumns...).
				Values(
					event.ID,
					event.Interface,
					event.Namespace,
					event.Name,
					event.Pathname,
					event.Description,
					event.Params,
					event.Details,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionFFIEvents, core.ChangeEventTypeCreated, event.Namespace, event.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) ffiEventResult(ctx context.Context, row *sql.Rows) (*core.FFIEvent, error) {
	event := core.FFIEvent{}
	err := row.Scan(
		&event.ID,
		&event.Interface,
		&event.Namespace,
		&event.Name,
		&event.Pathname,
		&event.Description,
		&event.Params,
		&event.Details,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, ffieventsTable)
	}
	return &event, nil
}

func (s *SQLCommon) getFFIEventPred(ctx context.Context, desc string, pred interface{}) (*core.FFIEvent, error) {
	rows, _, err := s.query(ctx, ffieventsTable,
		sq.Select(ffiEventsColumns...).
			From(ffieventsTable).
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("FFI event '%s' not found", desc)
		return nil, nil
	}

	ci, err := s.ffiEventResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return ci, nil
}

func (s *SQLCommon) GetFFIEvents(ctx context.Context, filter database.Filter) (events []*core.FFIEvent, res *database.FilterResult, err error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(ffiEventsColumns...).From(ffieventsTable), filter, ffiEventFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, ffieventsTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ci, err := s.ffiEventResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, ci)
	}

	return events, s.queryRes(ctx, ffieventsTable, tx, fop, fi), err

}

func (s *SQLCommon) GetFFIEvent(ctx context.Context, ns string, interfaceID *fftypes.UUID, pathName string) (*core.FFIEvent, error) {
	return s.getFFIEventPred(ctx, ns+":"+pathName, sq.And{sq.Eq{"namespace": ns}, sq.Eq{"interface_id": interfaceID}, sq.Eq{"pathname": pathName}})
}

func (s *SQLCommon) GetFFIEventByID(ctx context.Context, id *fftypes.UUID) (*core.FFIEvent, error) {
	return s.getFFIEventPred(ctx, id.String(), sq.Eq{"id": id})
}
