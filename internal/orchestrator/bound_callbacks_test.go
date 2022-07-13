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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/eventmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/dataexchange"
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
	mom := &operationmocks.Manager{}
	bc := boundCallbacks{dx: mdx, ei: mei, ss: mss, om: mom}

	info := fftypes.JSONObject{"hello": "world"}
	batch := &blockchain.BatchPin{TransactionID: fftypes.NewUUID()}
	pool := &tokens.TokenPool{}
	transfer := &tokens.TokenTransfer{}
	approval := &tokens.TokenApproval{}
	location := fftypes.JSONAnyPtr("{}")
	event := &blockchain.Event{}
	hash := fftypes.NewRandB32()
	opID := fftypes.NewUUID()

	mei.On("BatchPinComplete", batch, &core.VerifierRef{Value: "0x12345", Type: core.VerifierTypeEthAddress}).Return(fmt.Errorf("pop"))
	err := bc.BatchPinComplete(batch, &core.VerifierRef{Value: "0x12345", Type: core.VerifierTypeEthAddress})
	assert.EqualError(t, err, "pop")

	mei.On("BlockchainNetworkAction", "terminate", location, event, &core.VerifierRef{Value: "0x12345", Type: core.VerifierTypeEthAddress}).Return(fmt.Errorf("pop"))
	err = bc.BlockchainNetworkAction("terminate", location, event, &core.VerifierRef{Value: "0x12345", Type: core.VerifierTypeEthAddress})
	assert.EqualError(t, err, "pop")

	nsOpID := "ns1:" + opID.String()
	mom.On("SubmitOperationUpdate", mock.Anything, &operations.OperationUpdate{
		NamespacedOpID: nsOpID,
		Status:         core.OpStatusFailed,
		BlockchainTXID: "0xffffeeee",
		ErrorMessage:   "error info",
		Output:         info,
	}).Return()

	bc.OperationUpdate(mbi, nsOpID, core.OpStatusFailed, "0xffffeeee", "error info", info)

	mde := &dataexchangemocks.DXEvent{}
	mom.On("TransferResult", mdx, mde).Return()
	mei.On("DXEvent", mdx, mde).Return()

	mde.On("Type").Return(dataexchange.DXEventTypeTransferResult).Once()
	bc.DXEvent(mde)

	mde.On("Type").Return(dataexchange.DXEventTypeMessageReceived).Once()
	bc.DXEvent(mde)

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

	mei.On("SharedStorageBatchDownloaded", mss, "payload1", []byte(`{}`)).Return(nil, fmt.Errorf("pop"))
	_, err = bc.SharedStorageBatchDownloaded("payload1", []byte(`{}`))
	assert.EqualError(t, err, "pop")

	mei.On("SharedStorageBlobDownloaded", mss, *hash, int64(12345), "payload1").Return()
	bc.SharedStorageBlobDownloaded(*hash, 12345, "payload1")
}
