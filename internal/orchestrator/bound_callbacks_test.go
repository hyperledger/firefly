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

package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/eventmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestBoundCallbacks(t *testing.T) (*eventmocks.EventManager, *sharedstoragemocks.Plugin, *operationmocks.Manager, *boundCallbacks) {

	mei := &eventmocks.EventManager{}
	mss := &sharedstoragemocks.Plugin{}
	mom := &operationmocks.Manager{}
	bc := &boundCallbacks{
		o: &orchestrator{
			ctx:       context.Background(),
			namespace: &core.Namespace{Name: "ns1"},
			started:   true,
			events:    mei,
			plugins: &Plugins{
				SharedStorage: SharedStoragePlugin{
					Plugin: mss,
				},
			},
			operations: mom,
		},
	}
	return mei, mss, mom, bc
}

func TestBoundCallbacks(t *testing.T) {

	mei, mss, mom, bc := newTestBoundCallbacks(t)

	mdx := &dataexchangemocks.Plugin{}
	mti := &tokenmocks.Plugin{}
	info := fftypes.JSONObject{"hello": "world"}
	hash := fftypes.NewRandB32()
	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()
	dataID := fftypes.NewUUID()

	update := &core.OperationUpdate{
		NamespacedOpID: nsOpID,
		Status:         core.OpStatusFailed,
		BlockchainTXID: "0xffffeeee",
		ErrorMessage:   "error info",
		Output:         info,
	}
	mom.On("SubmitOperationUpdate", update).Return().Once()
	bc.OperationUpdate(update)

	mei.On("SharedStorageBatchDownloaded", mss, "payload1", []byte(`{}`)).Return(nil, fmt.Errorf("pop"))
	_, err := bc.SharedStorageBatchDownloaded("payload1", []byte(`{}`))
	assert.EqualError(t, err, "pop")

	mei.On("SharedStorageBlobDownloaded", mss, *hash, int64(12345), "payload1", dataID).Return(nil)
	err = bc.SharedStorageBlobDownloaded(*hash, 12345, "payload1", dataID)
	assert.NoError(t, err)

	mei.On("BlockchainEventBatch", []*blockchain.EventToDispatch{{Type: blockchain.EventTypeBatchPinComplete}}).Return(nil)
	err = bc.BlockchainEventBatch([]*blockchain.EventToDispatch{{Type: blockchain.EventTypeBatchPinComplete}})
	assert.NoError(t, err)

	mei.On("DXEvent", mdx, &dataexchangemocks.DXEvent{}).Return(nil)
	err = bc.DXEvent(mdx, &dataexchangemocks.DXEvent{})
	assert.NoError(t, err)

	mei.On("TokenPoolCreated", mock.Anything, mti, &tokens.TokenPool{}).Return(nil)
	err = bc.TokenPoolCreated(context.Background(), mti, &tokens.TokenPool{})
	assert.NoError(t, err)

	mei.On("TokensTransferred", mti, &tokens.TokenTransfer{}).Return(nil)
	err = bc.TokensTransferred(mti, &tokens.TokenTransfer{})
	assert.NoError(t, err)

	mei.On("TokensApproved", mti, &tokens.TokenApproval{}).Return(nil)
	err = bc.TokensApproved(mti, &tokens.TokenApproval{})
	assert.NoError(t, err)

	mei.AssertExpectations(t)
	mss.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestBoundCallbacksStopped(t *testing.T) {

	_, _, _, bc := newTestBoundCallbacks(t)
	bc.o.started = false

	_, err := bc.SharedStorageBatchDownloaded("payload1", []byte(`{}`))
	assert.Regexp(t, "FF10446", err)

	err = bc.SharedStorageBlobDownloaded(*fftypes.NewRandB32(), 12345, "payload1", nil)
	assert.Regexp(t, "FF10446", err)

	err = bc.BlockchainEventBatch([]*blockchain.EventToDispatch{})
	assert.Regexp(t, "FF10446", err)

	err = bc.DXEvent(nil, &dataexchangemocks.DXEvent{})
	assert.Regexp(t, "FF10446", err)

	err = bc.TokenPoolCreated(context.Background(), nil, &tokens.TokenPool{})
	assert.Regexp(t, "FF10446", err)

	err = bc.TokensTransferred(nil, &tokens.TokenTransfer{})
	assert.Regexp(t, "FF10446", err)

	err = bc.TokensApproved(nil, &tokens.TokenApproval{})
	assert.Regexp(t, "FF10446", err)
}
