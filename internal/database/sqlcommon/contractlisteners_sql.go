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

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

var (
	contractListenerColumns = []string{
		"id",
		"interface_id",
		"event",
		"namespace",
		"name",
		"backend_id",
		"location",
		"signature",
		"topic",
		"options",
		"state",
		"created",
	}
	contractListenerFilterFieldMap = map[string]string{
		"interface": "interface_id",
		"backendid": "backend_id",
	}
)

const contractlistenersTable = "contractlisteners"

func (s *SQLCommon) InsertContractListener(ctx context.Context, listener *core.ContractListener) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	var interfaceID *fftypes.UUID
	if listener.Interface != nil {
		interfaceID = listener.Interface.ID
	}

	listener.Created = fftypes.Now()
	if _, err = s.insertTx(ctx, contractlistenersTable, tx,
		sq.Insert(contractlistenersTable).
			Columns(contractListenerColumns...).
			Values(
				listener.ID,
				interfaceID,
				listener.Event,
				listener.Namespace,
				listener.Name,
				listener.BackendID,
				listener.Location,
				listener.Signature,
				listener.Topic,
				listener.Options,
				listener.State,
				listener.Created,
			),
		func() {
			s.callbacks.UUIDCollectionNSEvent(database.CollectionContractListeners, core.ChangeEventTypeCreated, listener.Namespace, listener.ID)
		},
	); err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractListenerResult(ctx context.Context, row *sql.Rows) (*core.ContractListener, error) {
	listener := core.ContractListener{
		Interface: &core.FFIReference{},
	}
	err := row.Scan(
		&listener.ID,
		&listener.Interface.ID,
		&listener.Event,
		&listener.Namespace,
		&listener.Name,
		&listener.BackendID,
		&listener.Location,
		&listener.Signature,
		&listener.Topic,
		&listener.Options,
		&listener.State,
		&listener.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, contractlistenersTable)
	}
	return &listener, nil
}

func (s *SQLCommon) getContractListenerPred(ctx context.Context, desc string, pred interface{}) (*core.ContractListener, error) {
	rows, _, err := s.query(ctx, contractlistenersTable,
		sq.Select(contractListenerColumns...).
			From(contractlistenersTable).
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Contract listener '%s' not found", desc)
		return nil, nil
	}

	sub, err := s.contractListenerResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *SQLCommon) GetContractListener(ctx context.Context, ns, name string) (sub *core.ContractListener, err error) {
	return s.getContractListenerPred(ctx, fmt.Sprintf("%s:%s", ns, name), sq.Eq{"namespace": ns, "name": name})
}

func (s *SQLCommon) GetContractListenerByID(ctx context.Context, id *fftypes.UUID) (sub *core.ContractListener, err error) {
	return s.getContractListenerPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetContractListenerByBackendID(ctx context.Context, id string) (sub *core.ContractListener, err error) {
	return s.getContractListenerPred(ctx, id, sq.Eq{"backend_id": id})
}

func (s *SQLCommon) GetContractListeners(ctx context.Context, filter database.Filter) ([]*core.ContractListener, *database.FilterResult, error) {
	query, fop, fi, err := s.filterSelect(ctx, "",
		sq.Select(contractListenerColumns...).From(contractlistenersTable),
		filter, contractListenerFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, contractlistenersTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	subs := []*core.ContractListener{}
	for rows.Next() {
		sub, err := s.contractListenerResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		subs = append(subs, sub)
	}

	return subs, s.queryRes(ctx, contractlistenersTable, tx, fop, fi), err
}

func (s *SQLCommon) UpdateContractListener(ctx context.Context, ns string, id *fftypes.UUID, dto *core.ContractListenerUpdateDTO) error {
	update := database.ContractListenerQueryFactory.NewUpdate(ctx).S()
	if dto.State != nil {
		update = update.Set("state", dto.State)
	}
	return s.updateContractListener(ctx, ns, id, update)
}

func (s *SQLCommon) updateContractListener(ctx context.Context, ns string, id *fftypes.UUID, update database.Update) error {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update(contractlistenersTable), update, opFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Set("updated", fftypes.Now())
	query = query.Where(sq.And{
		sq.Eq{"id": id},
		sq.Eq{"namespace": ns},
	})

	ra, err := s.updateTx(ctx, contractlistenersTable, tx, query, func() {
		s.callbacks.UUIDCollectionNSEvent(database.CollectionContractListeners, core.ChangeEventTypeUpdated, ns, id)
	})
	if err != nil {
		return err
	}
	if ra < 1 {
		return i18n.NewError(ctx, coremsgs.Msg404NoResult)
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) DeleteContractListenerByID(ctx context.Context, id *fftypes.UUID) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	sub, err := s.GetContractListenerByID(ctx, id)
	if err == nil && sub != nil {
		err = s.deleteTx(ctx, contractlistenersTable, tx, sq.Delete(contractlistenersTable).Where(sq.Eq{"id": id}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractListeners, core.ChangeEventTypeDeleted, sub.Namespace, sub.ID)
			},
		)
		if err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}
