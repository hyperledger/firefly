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

	"github.com/hyperledger-labs/firefly/mocks/blockchainmocks"
	"github.com/hyperledger-labs/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger-labs/firefly/mocks/eventmocks"
	"github.com/hyperledger-labs/firefly/mocks/tokenmocks"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestBoundCallbacks(t *testing.T) {
	mei := &eventmocks.EventManager{}
	mbi := &blockchainmocks.Plugin{}
	mdx := &dataexchangemocks.Plugin{}
	mti := &tokenmocks.Plugin{}
	bc := boundCallbacks{bi: mbi, dx: mdx, ei: mei}

	info := fftypes.JSONObject{"hello": "world"}
	batch := &blockchain.BatchPin{TransactionID: fftypes.NewUUID()}
	pool := &fftypes.TokenPool{}
	hash := fftypes.NewRandB32()

	mei.On("BatchPinComplete", mbi, batch, "0x12345", "tx12345", info).Return(fmt.Errorf("pop"))
	err := bc.BatchPinComplete(batch, "0x12345", "tx12345", info)
	assert.EqualError(t, err, "pop")

	mei.On("TxSubmissionUpdate", mbi, "tracking12345", fftypes.OpStatusFailed, "error info", info).Return(fmt.Errorf("pop"))
	err = bc.BlockchainTxUpdate("tracking12345", fftypes.OpStatusFailed, "error info", info)
	assert.EqualError(t, err, "pop")

	mei.On("TxSubmissionUpdate", mti, "tracking12345", fftypes.OpStatusFailed, "error info", info).Return(fmt.Errorf("pop"))
	err = bc.TokensTxUpdate(mti, "tracking12345", fftypes.OpStatusFailed, "error info", info)
	assert.EqualError(t, err, "pop")

	mei.On("TransferResult", mdx, "tracking12345", fftypes.OpStatusFailed, "error info", info).Return(fmt.Errorf("pop"))
	err = bc.TransferResult("tracking12345", fftypes.OpStatusFailed, "error info", info)
	assert.EqualError(t, err, "pop")

	mei.On("BLOBReceived", mdx, "peer1", *hash, "ns1/id1").Return(fmt.Errorf("pop"))
	err = bc.BLOBReceived("peer1", *hash, "ns1/id1")
	assert.EqualError(t, err, "pop")

	mei.On("MessageReceived", mdx, "peer1", []byte{}).Return(fmt.Errorf("pop"))
	err = bc.MessageReceived("peer1", []byte{})
	assert.EqualError(t, err, "pop")

	mei.On("TokenPoolCreated", mti, pool, "0x12345", "tx12345", info).Return(fmt.Errorf("pop"))
	err = bc.TokenPoolCreated(mti, pool, "0x12345", "tx12345", info)
	assert.EqualError(t, err, "pop")
}
