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

package utdbql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/mocks/blockchainmocks"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("utdbql_unit_tests")

func resetConf() {
	config.Reset()
	u := &UTDBQL{}
	u.InitPrefix(utConfPrefix)
}

func TestInit(t *testing.T) {
	u := &UTDBQL{}

	resetConf()
	utConfPrefix.Set(UTDBQLConfURL, "memory://")

	err := u.Init(context.Background(), utConfPrefix, &blockchainmocks.Callbacks{})
	assert.NoError(t, err)

	assert.Equal(t, "utdbql", u.Name())
	assert.NotNil(t, u.Capabilities())
	u.Close()
}

func TestInitBadURL(t *testing.T) {
	u := &UTDBQL{}

	resetConf()
	utConfPrefix.Set(UTDBQLConfURL, "!badness://")

	err := u.Init(context.Background(), utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Error(t, err)
}

func TestVerifyIdentitySyntaxOK(t *testing.T) {
	u := &UTDBQL{}
	err := u.VerifyIdentitySyntax(context.Background(), &fftypes.Identity{OnChain: "good"})
	assert.NoError(t, err)
}

func TestVerifyIdentitySyntaxFail(t *testing.T) {
	u := &UTDBQL{}
	err := u.VerifyIdentitySyntax(context.Background(), &fftypes.Identity{OnChain: "!bad"})
	assert.Regexp(t, "FF10131", err)
}

func TestVerifyBroadcastBatchTXCycle(t *testing.T) {
	u := &UTDBQL{}
	me := &blockchainmocks.Callbacks{}

	sbbEv := make(chan bool, 1)
	sbb := me.On("BatchPinComplete", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	sbb.RunFn = func(a mock.Arguments) {
		sbbEv <- true
	}

	txEv := make(chan bool, 1)
	tx := me.On("TxSubmissionUpdate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	tx.RunFn = func(a mock.Arguments) {
		txEv <- true
	}

	resetConf()
	utConfPrefix.Set(UTDBQLConfURL, "memory://")

	err := u.Init(context.Background(), utConfPrefix, me)
	assert.NoError(t, err)
	defer u.Close()

	u.Start()

	trackingID, err := u.SubmitBatchPin(context.Background(), nil, &fftypes.Identity{OnChain: "id1"}, &blockchain.BatchPin{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: fftypes.NewRandB32(),
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, trackingID)

	if err == nil {
		<-txEv
		<-sbbEv
	}

}

func TestCloseOnEventDispatchError(t *testing.T) {
	u := &UTDBQL{}
	me := &blockchainmocks.Callbacks{}

	sbbEv := make(chan bool, 1)
	sbb := me.On("BatchPinComplete", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("Pop"))
	sbb.RunFn = func(a mock.Arguments) {
		sbbEv <- true
	}

	me.On("TxSubmissionUpdate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	resetConf()
	utConfPrefix.Set(UTDBQLConfURL, "memory://")

	err := u.Init(context.Background(), utConfPrefix, me)
	assert.NoError(t, err)
	defer u.Close()

	u.Start()

	trackingID, err := u.SubmitBatchPin(context.Background(), nil, &fftypes.Identity{OnChain: "id1"}, &blockchain.BatchPin{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: fftypes.NewRandB32(),
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, trackingID)

	for !u.closed {
		time.Sleep(1 * time.Microsecond)
	}
}

func TestVerifyBroadcastDBError(t *testing.T) {
	u := &UTDBQL{}
	me := &blockchainmocks.Callbacks{}

	resetConf()
	utConfPrefix.Set(UTDBQLConfURL, "memory://")

	err := u.Init(context.Background(), utConfPrefix, me)
	assert.NoError(t, err)
	u.Close()

	_, err = u.SubmitBatchPin(context.Background(), nil, &fftypes.Identity{OnChain: "id1"}, &blockchain.BatchPin{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: fftypes.NewRandB32(),
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
		},
	})
	assert.Error(t, err)

}

func TestVerifyEventLoopCancelledContext(t *testing.T) {
	u := &UTDBQL{}
	me := &blockchainmocks.Callbacks{}

	resetConf()
	utConfPrefix.Set(UTDBQLConfURL, "memory://")

	err := u.Init(context.Background(), utConfPrefix, me)
	assert.NoError(t, err)
	defer u.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	u.ctx = ctx
	u.eventLoop() // Just confirming it exits
}

func TestVerifyDispatchEventBadData(t *testing.T) {
	u := &UTDBQL{}
	me := &blockchainmocks.Callbacks{}

	resetConf()
	utConfPrefix.Set(UTDBQLConfURL, "memory://")

	err := u.Init(context.Background(), utConfPrefix, me)
	assert.NoError(t, err)
	defer u.Close()

	u.dispatchEvent(&utEvent{
		txType: utDBQLEventTypeBatchPinComplete,
		data:   []byte(`!json`),
	}) // Just confirming it handles it
}
