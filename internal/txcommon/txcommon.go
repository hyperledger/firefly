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

package txcommon

import (
	"context"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type Helper interface {
	PersistTransaction(ctx context.Context, tx *fftypes.Transaction) (valid bool, err error)
}

type transactionHelper struct {
	database database.Plugin
}

func NewTransactionHelper(di database.Plugin) Helper {
	return &transactionHelper{
		database: di,
	}
}

func subjectMatch(a *fftypes.TransactionSubject, b *fftypes.TransactionSubject) bool {
	return a.Type == b.Type &&
		a.Signer == b.Signer &&
		a.Reference != nil && b.Reference != nil &&
		*a.Reference == *b.Reference &&
		a.Namespace == b.Namespace
}

func (t *transactionHelper) PersistTransaction(ctx context.Context, tx *fftypes.Transaction) (valid bool, err error) {
	if tx.ID == nil {
		log.L(ctx).Errorf("Invalid transaction - ID is nil")
		return false, nil // this is not retryable
	}
	if err := fftypes.ValidateFFNameField(ctx, tx.Subject.Namespace, "namespace"); err != nil {
		log.L(ctx).Errorf("Invalid transaction ID='%s' Reference='%s' - invalid namespace '%s': %a", tx.ID, tx.Subject.Reference, tx.Subject.Namespace, err)
		return false, nil // this is not retryable
	}
	existing, err := t.database.GetTransactionByID(ctx, tx.ID)
	if err != nil {
		return false, err // a persistence failure here is considered retryable (so returned)
	}

	switch {
	case existing == nil:
		// We're the first to write the transaction record on this node
		tx.Created = fftypes.Now()
		tx.Hash = tx.Subject.Hash()

	case subjectMatch(&tx.Subject, &existing.Subject):
		// This is an update to an existing transaction, but the subject is the same
		tx.Created = existing.Created
		tx.Hash = existing.Hash

	default:
		log.L(ctx).Errorf("Invalid transaction ID='%s' Reference='%s' - does not match existing subject", tx.ID, tx.Subject.Reference)
		return false, nil // this is not retryable
	}

	// Upsert the transaction, ensuring the hash does not change
	tx.Status = fftypes.OpStatusSucceeded
	err = t.database.UpsertTransaction(ctx, tx, false)
	if err != nil {
		if err == database.HashMismatch {
			log.L(ctx).Errorf("Invalid transaction ID='%s' Reference='%s' - hash mismatch with existing record '%s'", tx.ID, tx.Subject.Reference, tx.Hash)
			return false, nil // this is not retryable
		}
		log.L(ctx).Errorf("Failed to insert transaction ID='%s' Reference='%s': %a", tx.ID, tx.Subject.Reference, err)
		return false, err // a persistence failure here is considered retryable (so returned)
	}

	return true, nil
}
