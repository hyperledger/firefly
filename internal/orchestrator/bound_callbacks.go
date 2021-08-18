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
	"github.com/hyperledger-labs/firefly/internal/events"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/dataexchange"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
)

type boundCallbacks struct {
	bi blockchain.Plugin
	dx dataexchange.Plugin
	ei events.EventManager
}

func (bc *boundCallbacks) BlockchainTxUpdate(tx string, txState blockchain.TransactionStatus, errorMessage string, additionalInfo fftypes.JSONObject) error {
	return bc.ei.TxSubmissionUpdate(bc.bi, tx, txState, errorMessage, additionalInfo)
}

func (bc *boundCallbacks) TokensTxUpdate(plugin tokens.Plugin, tx string, txState fftypes.OpStatus, errorMessage string, additionalInfo fftypes.JSONObject) error {
	return bc.ei.TxSubmissionUpdate(plugin, tx, txState, errorMessage, additionalInfo)
}

func (bc *boundCallbacks) BatchPinComplete(batch *blockchain.BatchPin, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error {
	return bc.ei.BatchPinComplete(bc.bi, batch, signingIdentity, protocolTxID, additionalInfo)
}

func (bc *boundCallbacks) TransferResult(trackingID string, status fftypes.OpStatus, info string, additionalInfo fftypes.JSONObject) error {
	return bc.ei.TransferResult(bc.dx, trackingID, status, info, additionalInfo)
}

func (bc *boundCallbacks) BLOBReceived(peerID string, hash fftypes.Bytes32, payloadRef string) error {
	return bc.ei.BLOBReceived(bc.dx, peerID, hash, payloadRef)
}

func (bc *boundCallbacks) MessageReceived(peerID string, data []byte) error {
	return bc.ei.MessageReceived(bc.dx, peerID, data)
}

func (bc *boundCallbacks) TokenPoolCreated(plugin tokens.Plugin, pool *fftypes.TokenPool, signingIdentity string, additionalInfo fftypes.JSONObject) error {
	return bc.ei.TokenPoolCreated(plugin, pool, signingIdentity, additionalInfo)
}
