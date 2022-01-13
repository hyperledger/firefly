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
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	contractSubscriptionColumns = []string{
		"id",
		"interface_id",
		"event",
		"namespace",
		"name",
		"protocol_id",
		"location",
		"created",
	}
	contractSubscriptionFilterFieldMap = map[string]string{
		"interface":  "interface_id",
		"protocolid": "protocol_id",
	}
)

func (s *SQLCommon) UpsertContractSubscription(ctx context.Context, sub *fftypes.ContractSubscription) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	rows, _, err := s.queryTx(ctx, tx,
		sq.Select("seq").
			From("contractsubscriptions").
			Where(sq.Eq{"protocol_id": sub.ProtocolID}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	var interfaceID *fftypes.UUID
	if sub.Interface != nil {
		interfaceID = sub.Interface.ID
	}

	if existing {
		if _, err = s.updateTx(ctx, tx,
			sq.Update("contractsubscriptions").
				Set("id", sub.ID).
				Set("interface_id", interfaceID).
				Set("event", sub.Event).
				Set("namespace", sub.Namespace).
				Set("name", sub.Name).
				Set("location", sub.Location).
				Where(sq.Eq{"protocol_id": sub.ProtocolID}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractSubscriptions, fftypes.ChangeEventTypeUpdated, sub.Namespace, sub.ID)
			},
		); err != nil {
			return err
		}
	} else {
		sub.Created = fftypes.Now()
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("contractsubscriptions").
				Columns(contractSubscriptionColumns...).
				Values(
					sub.ID,
					interfaceID,
					sub.Event,
					sub.Namespace,
					sub.Name,
					sub.ProtocolID,
					sub.Location,
					sub.Created,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractSubscriptions, fftypes.ChangeEventTypeCreated, sub.Namespace, sub.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) contractSubscriptionResult(ctx context.Context, row *sql.Rows) (*fftypes.ContractSubscription, error) {
	sub := fftypes.ContractSubscription{
		Interface: &fftypes.FFIReference{},
	}
	err := row.Scan(
		&sub.ID,
		&sub.Interface.ID,
		&sub.Event,
		&sub.Namespace,
		&sub.Name,
		&sub.ProtocolID,
		&sub.Location,
		&sub.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "contractsubscriptions")
	}
	return &sub, nil
}

func (s *SQLCommon) getContractSubscriptionPred(ctx context.Context, desc string, pred interface{}) (*fftypes.ContractSubscription, error) {
	rows, _, err := s.query(ctx,
		sq.Select(contractSubscriptionColumns...).
			From("contractsubscriptions").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Contract subscription '%s' not found", desc)
		return nil, nil
	}

	sub, err := s.contractSubscriptionResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *SQLCommon) GetContractSubscription(ctx context.Context, ns, name string) (sub *fftypes.ContractSubscription, err error) {
	return s.getContractSubscriptionPred(ctx, fmt.Sprintf("%s:%s", ns, name), sq.Eq{"namespace": ns, "name": name})
}

func (s *SQLCommon) GetContractSubscriptionByID(ctx context.Context, id *fftypes.UUID) (sub *fftypes.ContractSubscription, err error) {
	return s.getContractSubscriptionPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetContractSubscriptionByProtocolID(ctx context.Context, id string) (sub *fftypes.ContractSubscription, err error) {
	return s.getContractSubscriptionPred(ctx, id, sq.Eq{"protocol_id": id})
}

func (s *SQLCommon) GetContractSubscriptions(ctx context.Context, filter database.Filter) ([]*fftypes.ContractSubscription, *database.FilterResult, error) {
	query, fop, fi, err := s.filterSelect(ctx, "",
		sq.Select(contractSubscriptionColumns...).From("contractsubscriptions"),
		filter, contractSubscriptionFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	subs := []*fftypes.ContractSubscription{}
	for rows.Next() {
		sub, err := s.contractSubscriptionResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		subs = append(subs, sub)
	}

	return subs, s.queryRes(ctx, tx, "contractsubscriptions", fop, fi), err
}

func (s *SQLCommon) DeleteContractSubscriptionByID(ctx context.Context, id *fftypes.UUID) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	sub, err := s.GetContractSubscriptionByID(ctx, id)
	if err == nil && sub != nil {
		err = s.deleteTx(ctx, tx, sq.Delete("contractsubscriptions").Where(sq.Eq{"id": id}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionContractSubscriptions, fftypes.ChangeEventTypeDeleted, sub.Namespace, sub.ID)
			},
		)
		if err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}
