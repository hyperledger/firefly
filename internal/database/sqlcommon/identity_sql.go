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
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var (
	identityColumns = []string{
		"id",
		"message_id",
		"name",
		"parent",
		"identity",
		"description",
		"profile",
		"created",
	}
	identityFilterFieldMap = map[string]string{
		"message": "message_id",
	}
)

func (s *SQLCommon) UpsertIdentity(ctx context.Context, identity *fftypes.Identity, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		// Do a select within the transaction to detemine if the UUID already exists
		identityRows, _, err := s.queryTx(ctx, tx,
			sq.Select("id").
				From("identities").
				Where(sq.Eq{"identity": identity.Identity}),
		)
		if err != nil {
			return err
		}
		existing = identityRows.Next()

		if existing {
			var id fftypes.UUID
			_ = identityRows.Scan(&id)
			if identity.ID != nil {
				if *identity.ID != id {
					identityRows.Close()
					return database.IDMismatch
				}
			}
			identity.ID = &id // Update on returned object
		}
		identityRows.Close()
	}

	if existing {
		// Update the identity
		if _, err = s.updateTx(ctx, tx,
			sq.Update("identities").
				// Note we do not update ID
				Set("message_id", identity.Message).
				Set("parent", identity.Parent).
				Set("identity", identity.Identity).
				Set("description", identity.Description).
				Set("profile", identity.Profile).
				Set("created", identity.Created).
				Where(sq.Eq{"identity": identity.Identity}),
			func() {
				s.callbacks.UUIDCollectionEvent(database.CollectionIdentitys, fftypes.ChangeEventTypeUpdated, identity.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("identities").
				Columns(identityColumns...).
				Values(
					identity.ID,
					identity.Message,
					identity.Name,
					identity.Parent,
					identity.Identity,
					identity.Description,
					identity.Profile,
					identity.Created,
				),
			func() {
				s.callbacks.UUIDCollectionEvent(database.CollectionIdentitys, fftypes.ChangeEventTypeCreated, identity.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}

func (s *SQLCommon) identityResult(ctx context.Context, row *sql.Rows) (*fftypes.Identity, error) {
	identity := fftypes.Identity{}
	err := row.Scan(
		&identity.ID,
		&identity.Message,
		&identity.Name,
		&identity.Parent,
		&identity.Identity,
		&identity.Description,
		&identity.Profile,
		&identity.Created,
	)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgDBReadErr, "identities")
	}
	return &identity, nil
}

func (s *SQLCommon) getIdentityPred(ctx context.Context, desc string, pred interface{}) (message *fftypes.Identity, err error) {

	rows, _, err := s.query(ctx,
		sq.Select(identityColumns...).
			From("identities").
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

	identity, err := s.identityResult(ctx, rows)
	if err != nil {
		return nil, err
	}

	return identity, nil
}

func (s *SQLCommon) GetIdentityByName(ctx context.Context, name string) (message *fftypes.Identity, err error) {
	return s.getIdentityPred(ctx, name, sq.Eq{"name": name})
}

func (s *SQLCommon) GetIdentityByIdentity(ctx context.Context, identity string) (message *fftypes.Identity, err error) {
	return s.getIdentityPred(ctx, identity, sq.Eq{"identity": identity})
}

func (s *SQLCommon) GetIdentityByID(ctx context.Context, id *fftypes.UUID) (message *fftypes.Identity, err error) {
	return s.getIdentityPred(ctx, id.String(), sq.Eq{"id": id})
}

func (s *SQLCommon) GetIdentitys(ctx context.Context, filter database.Filter) (message []*fftypes.Identity, fr *database.FilterResult, err error) {

	query, fop, fi, err := s.filterSelect(ctx, "", sq.Select(identityColumns...).From("identities"), filter, identityFilterFieldMap, []interface{}{"sequence"})
	if err != nil {
		return nil, nil, err
	}

	rows, tx, err := s.query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	identity := []*fftypes.Identity{}
	for rows.Next() {
		d, err := s.identityResult(ctx, rows)
		if err != nil {
			return nil, nil, err
		}
		identity = append(identity, d)
	}

	return identity, s.queryRes(ctx, tx, "identities", fop, fi), err

}

func (s *SQLCommon) UpdateIdentity(ctx context.Context, id *fftypes.UUID, update database.Update) (err error) {

	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	query, err := s.buildUpdate(sq.Update("identities"), update, identityFilterFieldMap)
	if err != nil {
		return err
	}
	query = query.Where(sq.Eq{"id": id})

	_, err = s.updateTx(ctx, tx, query, nil /* no change events for filter based updates */)
	if err != nil {
		return err
	}

	return s.commitTx(ctx, tx, autoCommit)
}
