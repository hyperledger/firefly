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

func (or *orchestrator) TransactionUpdate(txTrackingID string, txState fftypes.TransactionStatus, protocolTxID, errorMessage string, additionalInfo map[string]interface{}) error {
	return or.events.TransactionUpdate(txTrackingID, txState, protocolTxID, errorMessage, additionalInfo)
}

func (or *orchestrator) SequencedBroadcastBatch(batch *blockchain.BroadcastBatch, author string, protocolTxID string, additionalInfo map[string]interface{}) error {
	return or.events.SequencedBroadcastBatch(batch, author, protocolTxID, additionalInfo)
}
