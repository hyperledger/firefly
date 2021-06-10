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
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var (
	groupColumns = []string{
		"message_id",
		"namespace",
		"name",
		"ledger",
		"hash",
		"created",
	}
	groupFilterTypeMap = map[string]string{
		"message": "message_id",
	}
)

func (s *SQLCommon) UpsertGroup(ctx context.Context, group *fftypes.Group, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		groupRows, err := s.queryTx(ctx, tx,
			sq.Select("hash").
				From("groups").
				Where(sq.Eq{"hash": group.Hash}),
		)
		if err != nil {
			return err
		}
		existing = groupRows.Next()
		groupRows.Close()
	}

	if existing {

		// Update the group
		if err = s.updateTx(ctx, tx,
			sq.Update("groups").
				Set("message_id", group.Message).
				Set("namespace", group.Namespace).
				Set("name", group.Name).
				Set("ledger", group.Ledger).
				Set("hash", group.Hash).
				Set("created", group.Created).
				Where(sq.Eq{"hash": group.Hash}),
		); err != nil {
			return err
		}
	} else {
		_, err := s.insertTx(ctx, tx,
			sq.Insert("groups").
				Columns(groupColumns...).
				Values(
					group.Message,
					group.Namespace,
					group.Name,
					group.Ledger,
					group.Hash,
					group.Created,
				),
		)
		if err != nil {
			return err
		}

	}

	if err = s.updateMembers(ctx, tx, group, existing); err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) updateMembers(ctx context.Context, tx *txWrapper, group *fftypes.Group, existing bool) error {

	if existing {
		if err := s.deleteTx(ctx, tx,
			sq.Delete("members").
				Where(sq.And{
					sq.Eq{"group_hash": group.Hash},
				}),
		); err != nil {
			return err
		}
	}

	// Run through the ones in the group, finding ones that already exist, and ones that need to be created
	for requiredIdx, requiredMember := range group.Members {
		if requiredMember == nil || requiredMember.Identity == "" {
			return i18n.NewError(ctx, i18n.MsgEmptyMemberIdentity, requiredIdx)
		}
		if requiredMember.Node == nil {
			return i18n.NewError(ctx, i18n.MsgEmptyMemberNode, requiredIdx)
		}
		if _, err := s.insertTx(ctx, tx,
			sq.Insert("members").
				Columns(
					"group_hash",
					"identity",
					"node_id",
					"idx",
				).
				Values(
					group.Hash,
					requiredMember.Identity,
					requiredMember.Node,
					requiredIdx,
				),
		); err != nil {
			return err
		}
	}

	return nil

}

func (s *SQLCommon) loadMembers(ctx context.Context, groups []*fftypes.Group) error {

	groupIDs := make([]string, len(groups))
	for i, m := range groups {
		if m != nil {
			groupIDs[i] = m.Hash.String()
		}
	}

	members, err := s.query(ctx,
		sq.Select(
			"group_hash",
			"identity",
			"node_id",
			"idx",
		).
			From("members").
			Where(sq.Eq{"group_hash": groupIDs}).
			OrderBy("idx"),
	)
	if err != nil {
		return err
	}
	defer members.Close()

	for members.Next() {
		var groupID fftypes.Bytes32
		member := &fftypes.Member{}
		var idx int
		if err = members.Scan(&groupID, &member.Identity, &member.Node, &idx); err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "members")
		}
		for _, g := range groups {
			if g.Hash.Equals(&groupID) {
				g.Members = append(g.Members, member)
			}
		}
	}
	// Ensure we return an empty array if no entries, and a consistent order for the data
	for _, g := range groups {
		if g.Members == nil {
			g.Members = fftypes.Members{}
		}
	}

	return nil
}

func (s *SQLCommon) groupResult(ctx context.Context, row *sql.Rows) (*fftypes.Group, error) {
	var group fftypes.Group
	err := row.Scan(
		&group.Message,
		&group.Namespace,
		&group.Name,
		&group.Ledger,
		&group.Hash,
		&group.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "groups")
	}
	return &group, nil
}

func (s *SQLCommon) GetGroupByHash(ctx context.Context, hash *fftypes.Bytes32) (group *fftypes.Group, err error) {

	rows, err := s.query(ctx,
		sq.Select(groupColumns...).
			From("groups").
			Where(sq.Eq{"hash": hash}),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Group '%s' not found", hash)
		return nil, nil
	}

	group, err = s.groupResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	rows.Close()
	if err = s.loadMembers(ctx, []*fftypes.Group{group}); err != nil {
		return nil, err
	}

	return group, nil
}

func (s *SQLCommon) getGroupsQuery(ctx context.Context, query sq.SelectBuilder) (group []*fftypes.Group, err error) {
	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := []*fftypes.Group{}
	for rows.Next() {
		group, err := s.groupResult(ctx, rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	rows.Close()
	if len(groups) > 0 {
		if err = s.loadMembers(ctx, groups); err != nil {
			return nil, err
		}
	}

	return groups, err
}

func (s *SQLCommon) GetGroups(ctx context.Context, filter database.Filter) (group []*fftypes.Group, err error) {
	query, err := s.filterSelect(ctx, "", sq.Select(groupColumns...).From("groups"), filter, groupFilterTypeMap)
	if err != nil {
		return nil, err
	}
	return s.getGroupsQuery(ctx, query)
}

func (s *SQLCommon) UpdateGroup(ctx context.Context, hash *fftypes.Bytes32, update database.Update) (err error) {
	return s.UpdateGroups(ctx, database.GroupQueryFactory.NewFilter(ctx).Eq("hash", hash), update)
}

func (s *SQLCommon) UpdateGroups(ctx context.Context, filter database.Filter, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("groups"), update, groupFilterTypeMap)
	if err != nil {
		return err
	}

	query, err = s.filterUpdate(ctx, "", query, filter, opFilterTypeMap)
	if err != nil {
		return err
	}

	err = s.updateTx(ctx, tx, query)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
