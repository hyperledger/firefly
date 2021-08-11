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

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var (
	tokenPoolColumns = []string{
		"id",
		"namespace",
		"name",
		"pool_id",
		"is_fungible",
	}
)

func (s *SQLCommon) UpsertTokenPool(ctx context.Context, pool *fftypes.TokenPool, allowExisting bool) (err error) {
	ctx, tx, autoCommit, err := s.beginOrUseTx(ctx)
	if err != nil {
		return err
	}
	defer s.rollbackTx(ctx, tx, autoCommit)

	existing := false
	if allowExisting {
		transactionRows, _, err := s.queryTx(ctx, tx,
			sq.Select("seq").
				From("tokenpool").
				Where(sq.Eq{"id": pool.ID}),
		)
		if err != nil {
			return err
		}
		existing = transactionRows.Next()
		transactionRows.Close()
	}

	isFungible := 0
	if pool.Type == fftypes.TokenTypeFungible {
		isFungible = 1
	}

	if existing {
		if err = s.updateTx(ctx, tx,
			sq.Update("tokenpool").
				Set("id", pool.ID).
				Set("namespace", pool.Namespace).
				Set("name", pool.Name).
				Set("pool_id", pool.PoolID).
				Set("is_fungible", isFungible),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTransactions, fftypes.ChangeEventTypeUpdated, pool.Namespace, pool.ID)
			},
		); err != nil {
			return err
		}
	} else {
		if _, err = s.insertTx(ctx, tx,
			sq.Insert("tokenpool").
				Columns(tokenPoolColumns...).
				Values(
					pool.ID,
					pool.Namespace,
					pool.Name,
					pool.PoolID,
					isFungible,
				),
			func() {
				s.callbacks.UUIDCollectionNSEvent(database.CollectionTransactions, fftypes.ChangeEventTypeCreated, pool.Namespace, pool.ID)
			},
		); err != nil {
			return err
		}
	}

	return s.commitTx(ctx, tx, autoCommit)
}
