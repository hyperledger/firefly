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
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var (
	nodeColumns = []string{
		"id",
		"message_id",
		"owner",
		"name",
		"description",
		"dx_peer",
		"dx_endpoint",
		"created",
	}
	nodeFilterFieldMap = map[string]string{
		"message":     "message_id",
		"dx.peer":     "dx_peer",
		"dx.endpoint": "dx_endpoint",
	}
)

func (s *SQLCommon) UpsertNode(ctx context.Context, node *fftypes.Node, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		nodeRows, err := s.queryTx(ctx, tx,
			sq.Select("id").
				From("nodes").
				Where(sq.Eq{
					"owner": node.Owner,
					"name":  node.Name,
				}),
		)
		if err != nil {
			return err
		}
		existing = nodeRows.Next()

		if existing {
			var id fftypes.UUID
			_ = nodeRows.Scan(&id)
			if node.ID != nil {
				if *node.ID != id {
					nodeRows.Close()
					return database.IDMismatch
				}
			}
			node.ID = &id // Update on returned object
		}
		nodeRows.Close()
	}

	if existing {
		// Update the node
		if err = s.updateTx(ctx, tx,
			sq.Update("nodes").
				// Note we do not update ID
				Set("message_id", node.Message).
				Set("owner", node.Owner).
				Set("name", node.Name).
				Set("description", node.Description).
				Set("dx_peer", node.DX.Peer).
				Set("dx_endpoint", node.DX.Endpoint).
				Set("created", node.Created).
				Where(sq.Eq{"id": node.ID}),
			func() {
				s.callbacks.UUIDCollectionEvent(database.CollectionNodes, fftypes.ChangeEventTypeUpdated, node.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("nodes").
				Columns(nodeColumns...).
				Values(
					node.ID,
					node.Message,
					node.Owner,
					node.Name,
					node.Description,
					node.DX.Peer,
					node.DX.Endpoint,
					node.Created,
				),
			func() {
				s.callbacks.UUIDCollectionEvent(database.CollectionNodes, fftypes.ChangeEventTypeCreated, node.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) nodeResult(ctx context.Context, row *sql.Rows) (*fftypes.Node, error) {
	node := fftypes.Node{}
	err := row.Scan(
		&node.ID,
		&node.Message,
		&node.Owner,
		&node.Name,
		&node.Description,
		&node.DX.Peer,
		&node.DX.Endpoint,
		&node.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "nodes")
	}
	return &node, nil
}

func (s *SQLCommon) getNodePred(ctx context.Context, desc string, pred interface{}) (message *fftypes.Node, err error) {

	rows, err := s.query(ctx,
		sq.Select(nodeColumns...).
			From("nodes").
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Node '%s' not found", desc)
		return nil, nil
	}

	node, err := s.nodeResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (s *SQLCommon) GetNode(ctx context.Context, owner, name string) (message *fftypes.Node, err error) {
	return s.getNodePred(ctx, fmt.Sprintf("%s/%s", owner, name), sq.Eq{"owner": owner, "name": name})
}

func (s *SQLCommon) GetNodeByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Node, err error) {
	return s.getNodePred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetNodes(ctx context.Context, filter database.Filter) (message []*fftypes.Node, err error) {

	query, err := s.filterSelect(ctx, "", sq.Select(nodeColumns...).From("nodes"), filter, nodeFilterFieldMap, []string{"sequence"})
	if err != nil {
		return nil, err
	}

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	node := []*fftypes.Node{}
	for rows.Next() {
		d, err := s.nodeResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		node = append(node, d)
	}

	return node, err

}

func (s *SQLCommon) UpdateNode(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("nodes"), update, nodeFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	err = s.updateTx(ctx, tx, query, nil /* no change events for filter based updates */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
