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
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/events"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type boundCallbacks struct {
	bi blockchain.Plugin
	dx dataexchange.Plugin
	ss sharedstorage.Plugin
	ei events.EventManager
	om operations.Manager
}

func (bc *boundCallbacks) BlockchainOpUpdate(plugin blockchain.Plugin, operationID *fftypes.UUID, txState blockchain.TransactionStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) {
	bc.om.SubmitOperationUpdate(plugin, &operations.OperationUpdate{
		ID:             operationID,
		Status:         txState,
		BlockchainTXID: blockchainTXID,
		ErrorMessage:   errorMessage,
		Output:         opOutput,
	})
}

func (bc *boundCallbacks) TokenOpUpdate(plugin tokens.Plugin, operationID *fftypes.UUID, txState core.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) {
	bc.om.SubmitOperationUpdate(plugin, &operations.OperationUpdate{
		ID:             operationID,
		Status:         txState,
		BlockchainTXID: blockchainTXID,
		ErrorMessage:   errorMessage,
		Output:         opOutput,
	})
}

func (bc *boundCallbacks) BatchPinComplete(batch *blockchain.BatchPin, signingKey *core.VerifierRef) error {
	return bc.ei.BatchPinComplete(bc.bi, batch, signingKey)
}

func (bc *boundCallbacks) DXEvent(event dataexchange.DXEvent) {
	switch event.Type() {
	case dataexchange.DXEventTypeTransferResult:
		bc.om.TransferResult(bc.dx, event)
	default:
		bc.ei.DXEvent(bc.dx, event)
	}
}

func (bc *boundCallbacks) TokenPoolCreated(plugin tokens.Plugin, pool *tokens.TokenPool) error {
	return bc.ei.TokenPoolCreated(plugin, pool)
}

func (bc *boundCallbacks) TokensTransferred(plugin tokens.Plugin, transfer *tokens.TokenTransfer) error {
	return bc.ei.TokensTransferred(plugin, transfer)
}

func (bc *boundCallbacks) BlockchainEvent(event *blockchain.EventWithSubscription) error {
	return bc.ei.BlockchainEvent(event)
}

func (bc *boundCallbacks) TokensApproved(plugin tokens.Plugin, approval *tokens.TokenApproval) error {
	return bc.ei.TokensApproved(plugin, approval)
}

func (bc *boundCallbacks) SharedStorageBatchDownloaded(ns, payloadRef string, data []byte) (*fftypes.UUID, error) {
	return bc.ei.SharedStorageBatchDownloaded(bc.ss, ns, payloadRef, data)
}

func (bc *boundCallbacks) SharedStorageBlobDownloaded(hash fftypes.Bytes32, size int64, payloadRef string) {
	bc.ei.SharedStorageBlobDownloaded(bc.ss, hash, size, payloadRef)
}
