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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/eventmocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBoundCallbacks(t *testing.T) {
	mei := &eventmocks.EventManager{}
	mbi := &blockchainmocks.Plugin{}
	mdx := &dataexchangemocks.Plugin{}
	mti := &tokenmocks.Plugin{}
	mss := &sharedstoragemocks.Plugin{}
	bc := boundCallbacks{bi: mbi, dx: mdx, ei: mei, ss: mss}

	info := fftypes.JSONObject{"hello": "world"}
	batch := &blockchain.BatchPin{TransactionID: fftypes.NewUUID()}
	pool := &tokens.TokenPool{}
	transfer := &tokens.TokenTransfer{}
	approval := &tokens.TokenApproval{}
	hash := fftypes.NewRandB32()
	opID := fftypes.NewUUID()

	mei.On("BatchPinComplete", mbi, batch, &fftypes.VerifierRef{Value: "0x12345", Type: fftypes.VerifierTypeEthAddress}).Return(fmt.Errorf("pop"))
	err := bc.BatchPinComplete(batch, &fftypes.VerifierRef{Value: "0x12345", Type: fftypes.VerifierTypeEthAddress})
	assert.EqualError(t, err, "pop")

	mei.On("OperationUpdate", mbi, opID, fftypes.OpStatusFailed, "0xffffeeee", "error info", info).Return(fmt.Errorf("pop"))
	err = bc.BlockchainOpUpdate(opID, fftypes.OpStatusFailed, "0xffffeeee", "error info", info)
	assert.EqualError(t, err, "pop")

	mei.On("OperationUpdate", mti, opID, fftypes.OpStatusFailed, "0xffffeeee", "error info", info).Return(fmt.Errorf("pop"))
	err = bc.TokenOpUpdate(mti, opID, fftypes.OpStatusFailed, "0xffffeeee", "error info", info)
	assert.EqualError(t, err, "pop")

	mei.On("TransferResult", mdx, "tracking12345", fftypes.OpStatusFailed, mock.Anything).Return(fmt.Errorf("pop"))
	err = bc.TransferResult("tracking12345", fftypes.OpStatusFailed, fftypes.TransportStatusUpdate{
		Error: "error info", Info: info,
	})
	assert.EqualError(t, err, "pop")

	mei.On("PrivateBLOBReceived", mdx, "peer1", *hash, int64(12345), "ns1/id1").Return(fmt.Errorf("pop"))
	err = bc.PrivateBLOBReceived("peer1", *hash, 12345, "ns1/id1")
	assert.EqualError(t, err, "pop")

	mei.On("MessageReceived", mdx, "peer1", []byte{}).Return("manifest data", fmt.Errorf("pop"))
	_, err = bc.MessageReceived("peer1", []byte{})
	assert.EqualError(t, err, "pop")

	mei.On("TokenPoolCreated", mti, pool).Return(fmt.Errorf("pop"))
	err = bc.TokenPoolCreated(mti, pool)
	assert.EqualError(t, err, "pop")

	mei.On("TokensTransferred", mti, transfer).Return(fmt.Errorf("pop"))
	err = bc.TokensTransferred(mti, transfer)
	assert.EqualError(t, err, "pop")

	mei.On("TokensApproved", mti, approval).Return(fmt.Errorf("pop"))
	err = bc.TokensApproved(mti, approval)
	assert.EqualError(t, err, "pop")

	mei.On("BlockchainEvent", mock.AnythingOfType("*blockchain.EventWithSubscription")).Return(fmt.Errorf("pop"))
	err = bc.BlockchainEvent(&blockchain.EventWithSubscription{})
	assert.EqualError(t, err, "pop")

	mei.On("SharedStorageBatchDownloaded", mss, "ns1", "payload1", []byte(`{}`)).Return(nil, fmt.Errorf("pop"))
	_, err = bc.SharedStorageBatchDownloaded("ns1", "payload1", []byte(`{}`))
	assert.EqualError(t, err, "pop")

	mei.On("SharedStorageBLOBDownloaded", mss, *hash, int64(12345), "payload1").Return(fmt.Errorf("pop"))
	err = bc.SharedStorageBLOBDownloaded(*hash, 12345, "payload1")
	assert.EqualError(t, err, "pop")
}
