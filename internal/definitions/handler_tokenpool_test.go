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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/assetmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newPoolAnnouncement() *core.TokenPoolAnnouncement {
	pool := &core.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "name1",
		Type:      core.TokenTypeFungible,
		Locator:   "12345",
		Symbol:    "COIN",
		TX: core.TransactionRef{
			Type: core.TransactionTypeTokenPool,
			ID:   fftypes.NewUUID(),
		},
		Connector: "remote1",
	}
	return &core.TokenPoolAnnouncement{
		Pool: pool,
	}
}

func buildPoolDefinitionMessage(announce *core.TokenPoolAnnouncement) (*core.Message, core.DataArray, error) {
	msg := &core.Message{
		Header: core.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: core.SystemTagDefinePool,
		},
	}
	b, err := json.Marshal(announce)
	if err != nil {
		return nil, nil, err
	}
	data := core.DataArray{{
		Value: fftypes.JSONAnyPtrBytes(b),
	}}
	return msg, data, nil
}

func TestHandleDefinitionBroadcastTokenPoolActivateOK(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mam := sh.assets.(*assetmocks.Manager)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", pool.ID).Return(nil, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID && p.Connector == "connector1"
	})).Return(nil)
	mam.On("ActivateTokenPool", context.Background(), mock.AnythingOfType("*core.TokenPool")).Return(nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionWait, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	err = bs.RunPreFinalize(context.Background())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolBadConnector(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	pool.Name = "//bad"
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mam := sh.assets.(*assetmocks.Manager)
	mam.On("ActivateTokenPool", context.Background(), mock.AnythingOfType("*core.TokenPool")).Return(nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject, CustomCorrelator: pool.ID}, action)
	assert.Regexp(t, "FF10403", err)

	err = bs.RunPreFinalize(context.Background())
	assert.NoError(t, err)
}

func TestHandleDefinitionBroadcastTokenPoolGetPoolFail(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", pool.ID).Return(nil, fmt.Errorf("pop"))

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionBroadcastTokenPoolExisting(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mam := sh.assets.(*assetmocks.Manager)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", pool.ID).Return(&core.TokenPool{}, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(nil)
	mam.On("ActivateTokenPool", context.Background(), mock.AnythingOfType("*core.TokenPool")).Return(nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionWait, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	err = bs.RunPreFinalize(context.Background())
	assert.NoError(t, err)

}

func TestHandleDefinitionBroadcastTokenPoolExistingConfirmed(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)
	existing := &core.TokenPool{
		State: core.TokenPoolStateConfirmed,
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", pool.ID).Return(existing, nil)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolIDMismatch(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", pool.ID).Return(nil, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(database.IDMismatch)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject, CustomCorrelator: pool.ID}, action)
	assert.Error(t, err)

	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionBroadcastTokenPoolFailUpsert(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", pool.ID).Return(nil, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(fmt.Errorf("pop"))

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionRetry}, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionBroadcastTokenPoolActivateFail(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	announce := newPoolAnnouncement()
	pool := announce.Pool
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	mdi := sh.database.(*databasemocks.Plugin)
	mam := sh.assets.(*assetmocks.Manager)
	mdi.On("GetTokenPoolByID", context.Background(), "ns1", pool.ID).Return(nil, nil)
	mdi.On("UpsertTokenPool", context.Background(), mock.MatchedBy(func(p *core.TokenPool) bool {
		return *p.ID == *pool.ID && p.Message == msg.Header.ID
	})).Return(nil)
	mam.On("ActivateTokenPool", context.Background(), mock.AnythingOfType("*core.TokenPool")).Return(fmt.Errorf("pop"))

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionWait, CustomCorrelator: pool.ID}, action)
	assert.NoError(t, err)

	err = bs.RunPreFinalize(context.Background())
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastTokenPoolValidateFail(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	announce := &core.TokenPoolAnnouncement{
		Pool: &core.TokenPool{},
	}
	msg, data, err := buildPoolDefinitionMessage(announce)
	assert.NoError(t, err)

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, data, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)
	bs.assertNoFinalizers()
}

func TestHandleDefinitionBroadcastTokenPoolBadMessage(t *testing.T) {
	sh, bs := newTestDefinitionHandler(t)

	msg := &core.Message{
		Header: core.MessageHeader{
			ID:  fftypes.NewUUID(),
			Tag: core.SystemTagDefinePool,
		},
	}

	action, err := sh.HandleDefinitionBroadcast(context.Background(), &bs.BatchState, msg, nil, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.Error(t, err)
	bs.assertNoFinalizers()
}
