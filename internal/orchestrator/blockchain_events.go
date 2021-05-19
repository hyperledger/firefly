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

package orchestrator

import (
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

func (e *orchestrator) TransactionUpdate(txTrackingID string, txState fftypes.TransactionStatus, protocolTxId, errorMessage string, additionalInfo map[string]interface{}) error {
	return e.events.TransactionUpdate(txTrackingID, txState, protocolTxId, errorMessage, additionalInfo)
}

func (e *orchestrator) SequencedBroadcastBatch(batch *blockchain.BroadcastBatch, author string, protocolTxId string, additionalInfo map[string]interface{}) error {
	return e.events.SequencedBroadcastBatch(batch, author, protocolTxId, additionalInfo)
}
