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

package definitions

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newPoolAnnouncement() *fftypes.TokenPoolAnnouncement {
	pool := &fftypes.TokenPool{
		ID:         fftypes.NewUUID(),
		Namespace:  "ns1",
		Name:       "name1",
		Type:       fftypes.TokenTypeFungible,
		ProtocolID: "12345",
		Symbol:     "COIN",
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
	}
	return &fftypes.TokenPoolAnnouncement{
		Pool:  pool,
		Event: &fftypes.BlockchainEvent{},
	}
}

func buildPoolDefinitionMessage(announce *fftypes.TokenPoolAnnouncement) (*fftypes.Message, []*fftypes.Data, error) {
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}
	b, err := json.Marshal(announce)
	if err != nil {
		return nil, nil, err
	}
	data := []*fftypes.Data{{
		Value: fftypes.JSONAnyPtrBytes(b),
	}}
	return msg, data, nil
}

func TestHandleDefinitionBroadcastTokenPoolActivateOK(t *testing.T) {
	sh := newTestDefinitionHandlers(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mam := sh.assets.(*assetmocks.Manager)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(nil, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(nil)
	mam.On("ActivateTokenPool", context.Background(), mock.AnythingOfType("*fftypes.TokenPool"), mock.AnythingOfType("*fftypes.BlockchainEvent")).Return(nil)

	action, ba, err := sh.HandleDefinitionBroadcast(context.Background(), msg, data)
	assert.Equal(t, ActionWait, action)
	assert.NoError(t, err)

	err = ba.PreFinalize(context.Background())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolGetPoolFail(t *testing.T) {
	sh := newTestDefinitionHandlers(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(nil, fmt.Errorf("pop"))

	action, _, err := sh.HandleDefinitionBroadcast(context.Background(), msg, data)
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolExisting(t *testing.T) {
	sh := newTestDefinitionHandlers(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mam := sh.assets.(*assetmocks.Manager)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(&fftypes.TokenPool{}, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(nil)
	mam.On("ActivateTokenPool", context.Background(), mock.AnythingOfType("*fftypes.TokenPool"), mock.AnythingOfType("*fftypes.BlockchainEvent")).Return(nil)

	action, _, err := sh.HandleDefinitionBroadcast(context.Background(), msg, data)
	assert.Equal(t, ActionWait, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolExistingConfirmed(t *testing.T) {
	sh := newTestDefinitionHandlers(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)
	existing := &fftypes.TokenPool{
		State: fftypes.TokenPoolStateConfirmed,
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(existing, nil)

	action, _, err := sh.HandleDefinitionBroadcast(context.Background(), msg, data)
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolIDMismatch(t *testing.T) {
	sh := newTestDefinitionHandlers(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(nil, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(database.IDMismatch)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *fftypes.Event) bool {
		return *event.Reference == *pool.ID && event.Namespace == pool.Namespace && event.Type == fftypes.EventTypePoolRejected
	})).Return(nil)

	action, _, err := sh.HandleDefinitionBroadcast(context.Background(), msg, data)
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolFailUpsert(t *testing.T) {
	sh := newTestDefinitionHandlers(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(nil, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(fmt.Errorf("pop"))

	action, _, err := sh.HandleDefinitionBroadcast(context.Background(), msg, data)
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolActivateFail(t *testing.T) {
	sh := newTestDefinitionHandlers(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mam := sh.assets.(*assetmocks.Manager)
	mdi.On("GetTokenPoolByID", context.Background(), pool.ID).Return(nil, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *fftypes.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(nil)
	mam.On("ActivateTokenPool", context.Background(), mock.AnythingOfType("*fftypes.TokenPool"), mock.AnythingOfType("*fftypes.BlockchainEvent")).Return(fmt.Errorf("pop"))

	action, batchAction, err := sh.HandleDefinitionBroadcast(context.Background(), msg, data)
	assert.Equal(t, ActionWait, action)
	assert.NoError(t, err)

	err = batchAction.PreFinalize(context.Background())
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolValidateFail(t *testing.T) {
	sh := newTestDefinitionHandlers(t)

	announce := &fftypes.TokenPoolAnnouncement{
		Pool:  &fftypes.TokenPool{},
		Event: &fftypes.BlockchainEvent{},
	}
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("InsertEvent", context.Background(), mock.MatchedBy(func(event *fftypes.Event) bool {
		return event.Type == fftypes.EventTypePoolRejected
	})).Return(nil)

	action, _, err := sh.HandleDefinitionBroadcast(context.Background(), msg, data)
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolBadMessage(t *testing.T) {
	sh := newTestDefinitionHandlers(t)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: string(fftypes.SystemTagDefinePool),
		},
	}

	action, _, err := sh.HandleDefinitionBroadcast(context.Background(), msg, nil)
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}
