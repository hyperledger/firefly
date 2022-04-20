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
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
	"github.com/hyperledger/firefly/pkg/log"
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
		"created",
	}
	contractListenerFilterFieldMap = map[string]string{
		"interface": "interface_id",
		"backendid": "backend_id",
	}
)

const contractlistenersTable = "contractlisteners"

func (s *SQLCommon) UpsertContractListener(ctx context.Context, listener *fftypes.ContractListener) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, contractlistenersTable, tx,
		sq.Select("seq").
			From(contractlistenersTable).
			Where(sq.Eq{"backend_id": listener.BackendID}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	var interfaceID *fftypes.UUID
	if listener.Interface != nil {
		interfaceID = listener.Interface.ID
	}

	if existing {
		if _, err = s.updateTx(ctx, contractlistenersTable, tx,
			sq.Update(contractlistenersTable).
				Set("id", listener.ID).
				Set("interface_id", interfaceID).
				Set("event", listener.Event).
				Set("namespace", listener.Namespace).
				Set("name", listener.Name).
				Set("location", listener.Location).
				Set("signature", listener.Signature).
				Set("topic", listener.Topic).
				Set("options", listener.Options).
				Where(sq.Eq{"backend_id": listener.BackendID}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractListeners, fftypes.ChangeEventTypeUpdated, listener.Namespace, listener.ID)
			},
		); err != nil {
			return err
		}
	} else {
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
					listener.Created,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractListeners, fftypes.ChangeEventTypeCreated, listener.Namespace, listener.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractListenerResult(ctx context.Context, row *sql.Rows) (*fftypes.ContractListener, error) {
	listener := fftypes.ContractListener{
		Interface: &fftypes.FFIReference{},
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
		&listener.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, contractlistenersTable)
	}
	return &listener, nil
}

func (s *SQLCommon) getContractListenerPred(ctx context.Context, desc string, pred interface{}) (*fftypes.ContractListener, error) {
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

func (s *SQLCommon) GetContractListener(ctx context.Context, ns, name string) (sub *fftypes.ContractListener, err error) {
	return s.getContractListenerPred(ctx, fmt.Sprintf("%s:%s", ns, name), sq.Eq{"namespace": ns, "name": name})
}

func (s *SQLCommon) GetContractListenerByID(ctx context.Context, id *fftypes.UUID) (sub *fftypes.ContractListener, err error) {
	return s.getContractListenerPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetContractListenerByBackendID(ctx context.Context, id string) (sub *fftypes.ContractListener, err error) {
	return s.getContractListenerPred(ctx, id, sq.Eq{"backend_id": id})
}

func (s *SQLCommon) GetContractListeners(ctx context.Context, filter database.Filter) ([]*fftypes.ContractListener, *database.FilterResult, error) {
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

	subs := []*fftypes.ContractListener{}
	for rows.Next() {
		sub, err := s.contractListenerResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		subs = append(subs, sub)
	}

	return subs, s.queryRes(ctx, contractlistenersTable, tx, fop, fi), err
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
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractListeners, fftypes.ChangeEventTypeDeleted, sub.Namespace, sub.ID)
			},
		)
		if err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}
