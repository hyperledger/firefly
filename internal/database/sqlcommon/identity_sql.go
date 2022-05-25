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
	identityColumns = []string{
		"id",
		"did",
		"parent",
		"itype",
		"namespace",
		"name",
		"description",
		"profile",
		"messages_claim",
		"messages_verification",
		"messages_update",
		"created",
		"updated",
	}
	identityFilterFieldMap = map[string]string{
		"identity":              "identity_id",
		"type":                  "itype",
		"messages.claim":        "messages_claim",
		"messages.verification": "messages_verification",
		"messages.update":       "messages_update",
	}
)

const identitiesTable = "identities"

func (s *SQLCommon) attemptIdentityUpdate(ctx context.Context, tx *txWrapper, identity *core.Identity) (int64, error) {
	identity.Updated = fftypes.Now()
	return s.updateTx(ctx, identitiesTable, tx,
		sq.Update(identitiesTable).
			Set("did", identity.DID).
			Set("parent", identity.Parent).
			Set("itype", identity.Type).
			Set("namespace", identity.Namespace).
			Set("name", identity.Name).
			Set("description", identity.Description).
			Set("profile", identity.Profile).
			Set("messages_claim", identity.Messages.Claim).
			Set("messages_verification", identity.Messages.Verification).
			Set("messages_update", identity.Messages.Update).
			Set("updated", identity.Updated).
			Where(sq.Eq{
				"id": identity.ID,
			}),
		func() {
			s.callbacks.UUIDCollectionNSEvent(database.CollectionIdentities, core.ChangeEventTypeUpdated, identity.Namespace, identity.ID)
		})
}

func (s *SQLCommon) attemptIdentityInsert(ctx context.Context, tx *txWrapper, identity *core.Identity, requestConflictEmptyResult bool) (err error) {
	identity.Created = fftypes.Now()
	identity.Updated = identity.Created
	_, err = s.insertTxExt(ctx, identitiesTable, tx,
		sq.Insert(identitiesTable).
			Columns(identityColumns...).
			Values(
				identity.ID,
				identity.DID,
				identity.Parent,
				identity.Type,
				identity.Namespace,
				identity.Name,
				identity.Description,
				identity.Profile,
				identity.Messages.Claim,
				identity.Messages.Verification,
				identity.Messages.Update,
				identity.Created,
				identity.Updated,
			),
		func() {
			s.callbacks.UUIDCollectionNSEvent(database.CollectionIdentities, core.ChangeEventTypeCreated, identity.Namespace, identity.ID)
		}, requestConflictEmptyResult)
	return err
}

func (s *SQLCommon) UpsertIdentity(ctx context.Context, identity *core.Identity, optimization database.UpsertOptimization) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	optimized := false
	if optimization == database.UpsertOptimizationNew {
		opErr := s.attemptIdentityInsert(ctx, tx, identity, true /* we want a failure here we can progress past */)
		optimized = opErr == nil
	} else if optimization == database.UpsertOptimizationExisting {
		rowsAffected, opErr := s.attemptIdentityUpdate(ctx, tx, identity)
		optimized = opErr == nil && rowsAffected == 1
	}

	if !optimized {
		// Do a select within the transaction to detemine if the UUID already exists
		msgRows, _, err := s.queryTx(ctx, identitiesTable, tx,
			sq.Select("id").
				From(identitiesTable).
				Where(sq.Eq{"id": identity.ID}),
		)
		if err != nil {
			return err
		}
		existing := msgRows.Next()
		msgRows.Close()

		if existing {
			// Update the identity
			if _, err = s.attemptIdentityUpdate(ctx, tx, identity); err != nil {
				return err
			}
		} else {
			if err = s.attemptIdentityInsert(ctx, tx, identity, false); err != nil {
				return err
			}
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) identityResult(ctx context.Context, row *sql.Rows) (*core.Identity, error) {
	identity := core.Identity{}
	err := row.Scan(
		&identity.ID,
		&identity.DID,
		&identity.Parent,
		&identity.Type,
		&identity.Namespace,
		&identity.Name,
		&identity.Description,
		&identity.Profile,
		&identity.Messages.Claim,
		&identity.Messages.Verification,
		&identity.Messages.Update,
		&identity.Created,
		&identity.Updated,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgDBReadErr, identitiesTable)
	}
	return &identity, nil
}

func (s *SQLCommon) getIdentityPred(ctx context.Context, desc string, pred interface{}) (identity *core.Identity, err error) {

	rows, _, err := s.query(ctx, identitiesTable,
		sq.Select(identityColumns...).
			From(identitiesTable).
			Where(pred),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		log.L(ctx).Debugf("Identity '%s' not found", desc)
		return nil, nil
	}

	return s.identityResult(ctx, rows)
}

func (s *SQLCommon) GetIdentityByName(ctx context.Context, iType core.IdentityType, namespace, name string) (identity *core.Identity, err error) {
	return s.getIdentityPred(ctx, name, sq.Eq{"itype": iType, "namespace": namespace, "name": name})
}

func (s *SQLCommon) GetIdentityByDID(ctx context.Context, did string) (identity *core.Identity, err error) {
	return s.getIdentityPred(ctx, did, sq.Eq{"did": did})
}

func (s *SQLCommon) GetIdentityByID(ctx context.Context, id *fftypes.UUID) (identity *core.Identity, err error) {
	return s.getIdentityPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetIdentities(ctx context.Context, filter database.Filter) (identities []*core.Identity, fr *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(identityColumns...).From(identitiesTable), filter, identityFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, identitiesTable, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	identities = []*core.Identity{}
	for rows.Next() {
		d, err := s.identityResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		identities = append(identities, d)
	}

	return identities, s.queryRes(ctx, identitiesTable, tx, fop, fi), err

}

func (s *SQLCommon) UpdateIdentity(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update(identitiesTable), update, identityFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, identitiesTable, tx, query, nil /* no change events for filter based updates */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
