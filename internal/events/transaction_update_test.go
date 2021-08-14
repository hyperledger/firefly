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

	"github.com/hyperledger-labs/firefly/mocks/blockchainmocks"
	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTransactionLookupSuccess(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	opID := fftypes.NewUUID()
	mbi.On("Name").Return("ut")
	mdi.On("GetOperations", em.ctx, mock.Anything).Return([]*fftypes.Operation{
		{ID: opID},
	}, nil, nil)
	mdi.On("UpdateOperation", em.ctx, uuidMatches(opID), mock.Anything).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TxSubmissionUpdate(mbi, "tracking12345", fftypes.OpStatusFailed, "tx12345", "some error", info)
	assert.NoError(t, err)
}

func TestTransactionLookupNoResults(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	opID := fftypes.NewUUID()
	mbi.On("Name").Return("ut")
	mdi.On("GetOperations", em.ctx, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)
	mdi.On("UpdateOperation", em.ctx, uuidMatches(opID), mock.Anything).Return(nil)

	info := fftypes.JSONObject{"some": "info"}
	err := em.TxSubmissionUpdate(mbi, "tracking12345", fftypes.OpStatusFailed, "tx12345", "some error", info)
	assert.NoError(t, err)
}

func TestTransactionLookupNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	mbi.On("Name").Return("ut")
	mdi.On("GetOperations", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop")).Once()

	info := fftypes.JSONObject{"some": "info"}
	err := em.TxSubmissionUpdate(mbi, "tracking12345", fftypes.OpStatusFailed, "tx12345", "some error", info)
	assert.NoError(t, err) // swallowed after logging
}

func TestTxSubmissionUpdateError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	mdi := em.database.(*databasemocks.Plugin)
	mbi := &blockchainmocks.Plugin{}

	opID := fftypes.NewUUID()
	mbi.On("Name").Return("ut")
	mdi.On("GetOperations", em.ctx, mock.Anything).Return([]*fftypes.Operation{
		{ID: opID},
	}, nil, nil)
	mdi.On("UpdateOperation", em.ctx, uuidMatches(opID), mock.Anything).Return(fmt.Errorf("pop"))

	info := fftypes.JSONObject{"some": "info"}
	err := em.TxSubmissionUpdate(mbi, "tracking12345", fftypes.OpStatusFailed, "tx12345", "some error", info)
	assert.EqualError(t, err, "pop")
}
