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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestPersistTransactionNoID(t *testing.T) {
	mdb := &databasemocks.Plugin{}

	tx := &fftypes.Transaction{}

	valid, err := PersistTransaction(context.Background(), mdb, tx)
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestPersistTransactionNoNamespace(t *testing.T) {
	mdb := &databasemocks.Plugin{}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
	}

	valid, err := PersistTransaction(context.Background(), mdb, tx)
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestPersistTransactionGetTransactionError(t *testing.T) {
	mdb := &databasemocks.Plugin{}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
		},
	}

	mdb.On("GetTransactionByID", context.Background(), tx.ID).Return(nil, fmt.Errorf("pop"))

	valid, err := PersistTransaction(context.Background(), mdb, tx)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistTransactionSubjectMismatch(t *testing.T) {
	mdb := &databasemocks.Plugin{}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: fftypes.NewUUID(),
		},
	}
	existing := &fftypes.Transaction{
		ID: tx.ID,
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: fftypes.NewUUID(),
		},
	}

	mdb.On("GetTransactionByID", context.Background(), tx.ID).Return(existing, nil)

	valid, err := PersistTransaction(context.Background(), mdb, tx)
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestPersistTransactionUpsertFail(t *testing.T) {
	mdb := &databasemocks.Plugin{}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: fftypes.NewUUID(),
		},
	}
	existing := &fftypes.Transaction{
		ID: tx.ID,
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: tx.Subject.Reference,
		},
	}

	mdb.On("GetTransactionByID", context.Background(), tx.ID).Return(existing, nil)
	mdb.On("UpsertTransaction", context.Background(), tx, false).Return(fmt.Errorf("pop"))

	valid, err := PersistTransaction(context.Background(), mdb, tx)
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")
}

func TestPersistTransactionHashMismatch(t *testing.T) {
	mdb := &databasemocks.Plugin{}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: fftypes.NewUUID(),
		},
	}
	existing := &fftypes.Transaction{
		ID: tx.ID,
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: tx.Subject.Reference,
		},
	}

	mdb.On("GetTransactionByID", context.Background(), tx.ID).Return(existing, nil)
	mdb.On("UpsertTransaction", context.Background(), tx, false).Return(database.HashMismatch)

	valid, err := PersistTransaction(context.Background(), mdb, tx)
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestPersistTransactionUpdateOk(t *testing.T) {
	mdb := &databasemocks.Plugin{}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: fftypes.NewUUID(),
		},
	}
	existing := &fftypes.Transaction{
		ID: tx.ID,
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: tx.Subject.Reference,
		},
	}

	mdb.On("GetTransactionByID", context.Background(), tx.ID).Return(existing, nil)
	mdb.On("UpsertTransaction", context.Background(), tx, false).Return(nil)

	valid, err := PersistTransaction(context.Background(), mdb, tx)
	assert.True(t, valid)
	assert.NoError(t, err)
}

func TestPersistTransactionCreateOk(t *testing.T) {
	mdb := &databasemocks.Plugin{}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: "ns1",
			Reference: fftypes.NewUUID(),
		},
	}

	mdb.On("GetTransactionByID", context.Background(), tx.ID).Return(nil, nil)
	mdb.On("UpsertTransaction", context.Background(), tx, false).Return(nil)

	valid, err := PersistTransaction(context.Background(), mdb, tx)
	assert.True(t, valid)
	assert.NoError(t, err)

	assert.NotNil(t, tx.Created)
	assert.NotNil(t, tx.Hash)
}
