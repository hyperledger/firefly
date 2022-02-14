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

package events

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistBatchFromBroadcastRootOrg(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKeyIdentity", em.ctx, mock.Anything).Return("", nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(fmt.Errorf(("pop")))

	org := fftypes.Organization{
		ID:     fftypes.NewUUID(),
		Name:   "org1",
		Parent: "", // root
	}
	orgBytes, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		ID:        fftypes.NewUUID(),
		Value:     fftypes.JSONAnyPtrBytes(orgBytes),
		Validator: fftypes.MessageTypeDefinition,
	}

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		IdentityRef: fftypes.IdentityRef{
			Author: "did:firefly:org/12345",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID:   fftypes.NewUUID(),
				Type: fftypes.TransactionTypeBatchPin,
			},
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:   fftypes.NewUUID(),
						Type: fftypes.MessageTypeDefinition,
						IdentityRef: fftypes.IdentityRef{
							Author: "did:firefly:org/12345",
							Key:    "0x12345",
						},
					},
					Data: fftypes.DataRefs{
						{
							ID:   data.ID,
							Hash: data.Hash,
						},
					},
				},
			},
			Data: []*fftypes.Data{
				data,
			},
		},
	}
	batch.Hash = batch.Payload.Hash()

	_, err = em.persistBatchFromBroadcast(em.ctx, batch, batch.Hash, "0x12345")
	assert.EqualError(t, err, "pop") // Confirms we got to upserting the batch

}

func TestPersistBatchFromBroadcastRootOrgBadData(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKeyIdentity", em.ctx, mock.Anything).Return("", nil)

	data := &fftypes.Data{
		ID:        fftypes.NewUUID(),
		Value:     fftypes.JSONAnyPtr("!badness"),
		Validator: fftypes.MessageTypeDefinition,
	}

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		IdentityRef: fftypes.IdentityRef{
			Key: "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID:   fftypes.NewUUID(),
				Type: fftypes.TransactionTypeBatchPin,
			},
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						ID:   fftypes.NewUUID(),
						Type: fftypes.MessageTypeDefinition,
						IdentityRef: fftypes.IdentityRef{
							Key: "0x12345",
						},
					},
					Data: fftypes.DataRefs{
						{
							ID:   data.ID,
							Hash: data.Hash,
						},
					},
				},
			},
			Data: []*fftypes.Data{
				data,
			},
		},
	}
	batch.Hash = batch.Payload.Hash()

	valid, err := em.persistBatchFromBroadcast(em.ctx, batch, batch.Hash, "0x12345")
	assert.NoError(t, err)
	assert.False(t, valid)

}

func TestPersistBatchFromBroadcastNoRootOrgBadIdentity(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveSigningKeyIdentity", em.ctx, mock.Anything).Return("", nil)

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		IdentityRef: fftypes.IdentityRef{
			Key: "0x12345",
		},
	}
	batch.Hash = batch.Payload.Hash()

	valid, err := em.persistBatchFromBroadcast(em.ctx, batch, batch.Hash, "0x12345")
	assert.NoError(t, err)
	assert.False(t, valid)

}
