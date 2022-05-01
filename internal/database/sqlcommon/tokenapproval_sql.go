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
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/i18n"
	"github.com/hyperledger/firefly/pkg/log"
)

var (
	tokenApprovalColumns = []string{
		"local_id",
		"protocol_id",
		"subject",
		"active",
		"key",
		"operator_key",
		"pool_id",
		"connector",
		"namespace",
		"approved",
		"info",
		"tx_type",
		"tx_id",
		"blockchain_event",
		"created",
	}
	tokenApprovalFilterFieldMap = map[string]string{
		"localid":         "local_id",
		"protocolid":      "protocol_id",
		"pool":            "pool_id",
		"approved":        "approved",
		"key":             "key",
		"operator":        "operator_key",
		"tx.type":         "tx_type",
		"tx.id":           "tx_id",
		"blockchainevent": "blockchain_event",
		"created":         "created",
	}
)

const tokenapprovalTable = "tokenapproval"

func (s *SQLCommon) UpsertTokenApproval(ctx context.Context, approval *fftypes.TokenApproval) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}

	defer s.rollbackTx(ctx, tx, autoCommit)
	rows, _, err := s.queryTx(ctx, tokenapprovalTable, tx,
		sq.Select("seq").
			From(tokenapprovalTable).
			Where(sq.Eq{"protocol_id": approval.ProtocolID}),
	)
	if err != nil {
		return err
	}
	existing := rows.Next()
	rows.Close()

	if existing {
		if _, err = s.updateTx(ctx, tokenapprovalTable, tx,
			sq.Update(tokenapprovalTable).
				Set("local_id", approval.LocalID).
				Set("subject", approval.Subject).
				Set("active", approval.Active).
				Set("key", approval.Key).
				Set("operator_key", approval.Operator).
				Set("pool_id", approval.Pool).
				Set("connector", approval.Connector).
				Set("namespace", approval.Namespace).
				Set("approved", approval.Approved).
				Set("info", approval.Info).
				Set("tx_type", approval.TX.Type).
				Set("tx_id", approval.TX.ID).
				Set("blockchain_event", approval.BlockchainEvent).
				Where(sq.Eq{"protocol_id": approval.ProtocolID}),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTokenApprovals, fftypes.ChangeEventTypeUpdated, approval.Namespace, approval.LocalID)
			},
		); err != nil {
			return err
		}
	} else {
		approval.Created = fftypes.Now()
		if _, err = s.insertTx(ctx, tokenapprovalTable, tx,
			sq.Insert(tokenapprovalTable).
				Columns(tokenApprovalColumns...).
				Values(
					approval.LocalID,
					approval.ProtocolID,
					approval.Subject,
					approval.Active,
					approval.Key,
					approval.Operator,
					approval.Pool,
					approval.Connector,
					approval.Namespace,
					approval.Approved,
					approval.Info,
					approval.TX.Type,
					approval.TX.ID,
					approval.BlockchainEvent,
					approval.Created,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTokenApprovals, fftypes.ChangeEventTypeCreated, approval.Namespace, approval.LocalID)
			},
		); err != nil {
			return err
		}
	}
	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) tokenApprovalResult(ctx context.Context, row *sql.Rows) (*fftypes.TokenApproval, error) {
	approval := fftypes.TokenApproval{}
	err := row.Scan(
		&approval.LocalID,
		&approval.ProtocolID,
		&approval.Subject,
		&approval.Active,
		&approval.Key,
		&approval.Operator,
		&approval.Pool,
		&approval.Connector,
		&approval.Namespace,
		&approval.Approved,
		&approval.Info,
		&approval.TX.Type,
		&approval.TX.ID,
		&approval.BlockchainEvent,
		&approval.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, tokenapprovalTable)
	}
	return &approval, nil
}

func (s *SQLCommon) getTokenApprovalPred(ctx context.Context, desc string, pred interface{}) (*fftypes.TokenApproval, error) {
	rows, _, err := s.query(ctx, tokenapprovalTable,
		sq.Select(tokenApprovalColumns...).
			From(tokenapprovalTable).
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Token approval '%s' not found", desc)
		return nil, nil
	}

	approval, err := s.tokenApprovalResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return approval, nil
}

func (s *SQLCommon) GetTokenApprovalByID(ctx context.Context, localID *fftypes.UUID) (*fftypes.TokenApproval, error) {
	return s.getTokenApprovalPred(ctx, localID.String(), sq.Eq{"local_id": localID})
}

func (s *SQLCommon) GetTokenApprovalByProtocolID(ctx context.Context, poolID *fftypes.UUID, protocolID string) (*fftypes.TokenApproval, error) {
	return s.getTokenApprovalPred(ctx, protocolID, sq.And{
		sq.Eq{"pool_id": poolID},
		sq.Eq{"protocol_id": protocolID},
	})
}

func (s *SQLCommon) GetTokenApprovals(ctx context.Context, filter database.Filter) (approvals []*fftypes.TokenApproval, fr *database.FilterResult, err error) {
	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(tokenApprovalColumns...).From(tokenapprovalTable), filter, tokenApprovalFilterFieldMap, []interface{}{"seq"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, tokenapprovalTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	approvals = []*fftypes.TokenApproval{}
	for rows.Next() {
		d, err := s.tokenApprovalResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		approvals = append(approvals, d)
	}

	return approvals, s.queryRes(ctx, tokenapprovalTable, tx, fop, fi), err
}

func (s *SQLCommon) UpdateTokenApprovals(ctx context.Context, filter database.Filter, update database.Update) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update(tokenapprovalTable), update, tokenApprovalFilterFieldMap)
	if err != nil {
		return err
	}

	query, err = s.filterUpdate(ctx, query, filter, tokenApprovalFilterFieldMap)
	if err != nil {
		return err
	}

	_, err = s.updateTx(ctx, tokenapprovalTable, tx, query, nil /* no change events filter based update */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
