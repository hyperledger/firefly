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

	"github.com/kaleido-io/firefly/mocks/blockchainmocks"
	"github.com/kaleido-io/firefly/mocks/eventmocks"
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestBoundBlockchainCallbacks(t *testing.T) {
	mei := &eventmocks.EventManager{}
	mbi := &blockchainmocks.Plugin{}
	bbc := boundBlockchainCallbacks{bi: mbi, ei: mei}

	info := fftypes.JSONObject{"hello": "world"}
	batch := &blockchain.BatchPin{TransactionID: fftypes.NewUUID()}

	mei.On("BatchPinComplete", mbi, batch, "0x12345", "tx12345", info).Return(fmt.Errorf("pop"))
	err := bbc.BatchPinComplete(batch, "0x12345", "tx12345", info)
	assert.EqualError(t, err, "pop")

	mei.On("TransactionUpdate", mbi, "tracking12345", fftypes.OpStatusFailed, "tx12345", "error info", info).Return(fmt.Errorf("pop"))
	err = bbc.TransactionUpdate("tracking12345", fftypes.OpStatusFailed, "tx12345", "error info", info)
	assert.EqualError(t, err, "pop")
}
