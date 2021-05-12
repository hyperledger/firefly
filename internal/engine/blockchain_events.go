// Copyright © 2021 Kaleido, Inc.
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

package engine

import (
	"context"

	"github.com/kaleido-io/firefly/internal/blockchain"
)

type blockchainEvents struct {
	ctx context.Context
	e   *engine
}

func (be *blockchainEvents) TransactionUpdate(txTrackingID string, txState blockchain.TransactionStatus, protocolTxId, errorMessage string, additionalInfo map[string]interface{}) {

}

func (be *blockchainEvents) SequencedBroadcastBatch(batch *blockchain.BroadcastBatch, author string, protocolTxId string, additionalInfo map[string]interface{}) {

}
