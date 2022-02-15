// Copyright © 2022 Kaleido, Inc.
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

package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

var msgID = fftypes.NewUUID()
var Message = &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: msgID,
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/abcd",
				Key:    "0x12345",
			},
			Type: "",
		},
	}


var tokenLocalID = fftypes.NewUUID()
var TokenTransfer = &fftypes.TokenTransfer{
		Amount:  *fftypes.NewFFBigInt(1),
		LocalID: tokenLocalID,
		Type:    "",
	}


func newTestMetricsManager(t *testing.T) (*metricsManager, func()) {
	config.Reset()
	Clear()
	Registry()
	ctx, cancel := context.WithCancel(context.Background())
	mmi := NewMetricsManager(ctx)
	mm := mmi.(*metricsManager)
	assert.Equal(t, len(mm.timeMap), 0)
	return mm, cancel
}

func TestStartDoesNothing(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	assert.NoError(t, mm.Start())
}

func TestCountBatchPin(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	mm.CountBatchPin()
}

func TestMessageSubmittedBroadcast(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	Message.Header.Type = fftypes.MessageTypeBroadcast
	mm.MessageSubmitted(Message)
	assert.Equal(t, len(mm.timeMap), 1)
	assert.NotNil(t, mm.timeMap[msgID.String()])
}

func TestMessageSubmittedPrivate(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	Message.Header.Type = fftypes.MessageTypePrivate
	mm.MessageSubmitted(Message)
	assert.Equal(t, len(mm.timeMap), 1)
	assert.NotNil(t, mm.timeMap[msgID.String()])
}

func TestMessageConfirmedBroadcastSuccess(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	Message.Header.Type = fftypes.MessageTypeBroadcast
	mm.timeMap[msgID.String()] = time.Now()
	mm.MessageConfirmed(Message, fftypes.EventTypeMessageConfirmed)
	assert.Equal(t, len(mm.timeMap), 0)
}

func TestMessageConfirmedBroadcastRejected(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	Message.Header.Type = fftypes.MessageTypeBroadcast
	mm.timeMap[msgID.String()] = time.Now()
	mm.MessageConfirmed(Message, fftypes.EventTypeMessageRejected)
	assert.Equal(t, len(mm.timeMap), 0)
}

func TestMessageConfirmedPrivateSuccess(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	Message.Header.Type = fftypes.MessageTypePrivate
	mm.timeMap[msgID.String()] = time.Now()
	mm.MessageConfirmed(Message, fftypes.EventTypeMessageConfirmed)
	assert.Equal(t, len(mm.timeMap), 0)
}

func TestMessageConfirmedPrivateRejected(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	Message.Header.Type = fftypes.MessageTypePrivate
	mm.timeMap[msgID.String()] = time.Now()
	mm.MessageConfirmed(Message, fftypes.EventTypeMessageRejected)
	assert.Equal(t, len(mm.timeMap), 0)
}

func TestTokenSubmittedMint(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	TokenTransfer.Type = fftypes.TokenTransferTypeMint
	mm.TransferSubmitted(TokenTransfer)
	assert.Equal(t, len(mm.timeMap), 1)
	assert.NotNil(t, mm.timeMap[tokenLocalID.String()])
}

func TestTransferSubmittedTransfer(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	TokenTransfer.Type = fftypes.TokenTransferTypeTransfer
	mm.TransferSubmitted(TokenTransfer)
	assert.Equal(t, len(mm.timeMap), 1)
	assert.NotNil(t, mm.timeMap[tokenLocalID.String()])
}

func TestTransferSubmittedBurn(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	TokenTransfer.Type = fftypes.TokenTransferTypeBurn
	mm.TransferSubmitted(TokenTransfer)
	assert.Equal(t, len(mm.timeMap), 1)
	assert.NotNil(t, mm.timeMap[tokenLocalID.String()])
}

func TestTransferConfirmedMintSuccess(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	TokenTransfer.Type = fftypes.TokenTransferTypeMint
	mm.timeMap[tokenLocalID.String()] = time.Now()
	mm.TransferConfirmed(TokenTransfer)
	assert.Equal(t, len(mm.timeMap), 0)
}
func TestTransferConfirmedTransferSuccess(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	TokenTransfer.Type = fftypes.TokenTransferTypeTransfer
	mm.timeMap[tokenLocalID.String()] = time.Now()
	mm.TransferConfirmed(TokenTransfer)
	assert.Equal(t, len(mm.timeMap), 0)
}
func TestTransferConfirmedMintBurn(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	TokenTransfer.Type = fftypes.TokenTransferTypeBurn
	mm.timeMap[tokenLocalID.String()] = time.Now()
	mm.TransferConfirmed(TokenTransfer)
	assert.Equal(t, len(mm.timeMap), 0)
}

func TestIsMetricsEnabledTrue(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	mm.metricsEnabled = true
	assert.Equal(t, mm.IsMetricsEnabled(), true)
}

func TestIsMetricsEnabledFalse(t *testing.T) {
	mm, cancel := newTestMetricsManager(t)
	defer cancel()
	mm.metricsEnabled = false
	assert.Equal(t, mm.IsMetricsEnabled(), false)
}
