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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOperatorAction(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}
	mdi := em.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", em.ctx, "ff_system").Return(&core.Namespace{}, nil)
	mdi.On("UpsertNamespace", em.ctx, mock.MatchedBy(func(ns *core.Namespace) bool {
		return ns.ContractIndex == 1
	}), true).Return(nil)
	mbi.On("Stop").Return()
	mbi.On("Start", 1).Return(nil)

	err := em.BlockchainOperatorAction(mbi, "migrate", "1", &core.VerifierRef{})
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestOperatorActionUnknown(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}

	err := em.BlockchainOperatorAction(mbi, "bad", "", &core.VerifierRef{})
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
}

func TestActionMigrateQueryFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}
	mdi := em.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", em.ctx, "ff_system").Return(nil, fmt.Errorf("pop"))

	err := em.actionMigrate(mbi, "1")
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestActionMigrateBadIndex(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}
	mdi := em.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", em.ctx, "ff_system").Return(&core.Namespace{}, nil)

	err := em.actionMigrate(mbi, "!bad")
	assert.Regexp(t, "Atoi", err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestActionMigrateSkip(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}
	mdi := em.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", em.ctx, "ff_system").Return(&core.Namespace{ContractIndex: 1}, nil)

	err := em.actionMigrate(mbi, "1")
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestActionMigrateStartFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}
	mdi := em.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", em.ctx, "ff_system").Return(&core.Namespace{}, nil)
	mbi.On("Stop").Return(nil)
	mbi.On("Start", 1).Return(fmt.Errorf("pop"))

	err := em.actionMigrate(mbi, "1")
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestActionMigrateUpsertFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mbi := &blockchainmocks.Plugin{}
	mdi := em.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", em.ctx, "ff_system").Return(&core.Namespace{}, nil)
	mdi.On("UpsertNamespace", em.ctx, mock.MatchedBy(func(ns *core.Namespace) bool {
		return ns.ContractIndex == 1
	}), true).Return(fmt.Errorf("pop"))
	mbi.On("Stop").Return(nil)
	mbi.On("Start", 1).Return(nil)

	err := em.actionMigrate(mbi, "1")
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
	mdi.AssertExpectations(t)
}
