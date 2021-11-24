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

package broadcast

import (
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastDefinitionAsNodeConfirm(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mdi := bm.database.(*databasemocks.Plugin)
	msa := bm.syncasync.(*syncasyncmocks.Bridge)
	mim := bm.identity.(*identitymanagermocks.Manager)

	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mim.On("ResolveInputIdentity", mock.Anything, mock.Anything).Return(nil)
	msa.On("WaitForMessage", bm.ctx, "ff_system", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := bm.BroadcastDefinitionAsNode(bm.ctx, &fftypes.Namespace{}, fftypes.SystemTagDefineNamespace, true)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	msa.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBroadcastDefinitionAsNodeUpsertFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))
	mim := bm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", mock.Anything, mock.Anything).Return(nil)
	_, err := bm.BroadcastDefinitionAsNode(bm.ctx, &fftypes.Namespace{}, fftypes.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)
}

func TestBroadcastDefinitionBadIdentity(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mim := bm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := bm.BroadcastDefinition(bm.ctx, &fftypes.Namespace{}, &fftypes.Identity{
		Author: "wrong",
		Key:    "wrong",
	}, fftypes.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)
}

func TestBroadcastRootOrgDefinitionPassedThroughAnyIdentity(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mim := bm.identity.(*identitymanagermocks.Manager)
	mim.On("OrgDID", mock.Anything, mock.Anything).Return("did:firefly:org/12345", nil)
	// Should call through to upsert data, stop test there
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	_, err := bm.BroadcastRootOrgDefinition(bm.ctx, &fftypes.Organization{
		ID: fftypes.NewUUID(),
	}, &fftypes.Identity{
		Author: "anything - overridden",
		Key:    "0x12345",
	}, fftypes.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
}
