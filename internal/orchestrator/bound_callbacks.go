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

package orchestrator

import (
	"github.com/hyperledger/firefly/internal/events"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type boundCallbacks struct {
	bi blockchain.Plugin
	dx dataexchange.Plugin
	ei events.EventManager
}

func (bc *boundCallbacks) BlockchainOpUpdate(operationID *fftypes.UUID, txState blockchain.TransactionStatus, errorMessage string, opOutput fftypes.JSONObject) error {
	return bc.ei.OperationUpdate(bc.bi, operationID, txState, errorMessage, opOutput)
}

func (bc *boundCallbacks) TokenOpUpdate(plugin tokens.Plugin, operationID *fftypes.UUID, txState fftypes.OpStatus, errorMessage string, opOutput fftypes.JSONObject) error {
	return bc.ei.OperationUpdate(plugin, operationID, txState, errorMessage, opOutput)
}

func (bc *boundCallbacks) BatchPinComplete(batch *blockchain.BatchPin, author string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	return bc.ei.BatchPinComplete(bc.bi, batch, author, protocolTxID, additionalInfo)
}

func (bc *boundCallbacks) TransferResult(trackingID string, status fftypes.OpStatus, update fftypes.TransportStatusUpdate) error {
	return bc.ei.TransferResult(bc.dx, trackingID, status, update)
}

func (bc *boundCallbacks) BLOBReceived(peerID string, hash fftypes.Bytes32, size int64, payloadRef string) error {
	return bc.ei.BLOBReceived(bc.dx, peerID, hash, size, payloadRef)
}

func (bc *boundCallbacks) MessageReceived(peerID string, data []byte) (manifest string, err error) {
	return bc.ei.MessageReceived(bc.dx, peerID, data)
}

func (bc *boundCallbacks) TokenPoolCreated(plugin tokens.Plugin, pool *tokens.TokenPool, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	return bc.ei.TokenPoolCreated(plugin, pool, protocolTxID, additionalInfo)
}

func (bc *boundCallbacks) TokensTransferred(plugin tokens.Plugin, poolProtocolID string, transfer *fftypes.TokenTransfer, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	return bc.ei.TokensTransferred(plugin, poolProtocolID, transfer, protocolTxID, additionalInfo)
}
