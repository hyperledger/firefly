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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func compactJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		panic(err)
	}
	return buf.String()
}

func TestGetTransactionStatusBatchPinSuccess(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeBatchPin,
	}
	ops := []*fftypes.Operation{
		{
			Status:  fftypes.OpStatusSucceeded,
			ID:      fftypes.NewUUID(),
			Type:    fftypes.OpTypeBlockchainBatchPin,
			Updated: fftypes.UnixTime(0),
			Output:  fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	events := []*fftypes.BlockchainEvent{
		{
			Name:      "BatchPin",
			ID:        fftypes.NewUUID(),
			Timestamp: fftypes.UnixTime(1),
			Info:      fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	batches := []*fftypes.BatchPersisted{
		{
			BatchHeader: fftypes.BatchHeader{
				ID:   fftypes.NewUUID(),
				Type: fftypes.BatchTypeBroadcast,
			},
			Confirmed: fftypes.UnixTime(2),
		},
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetBatches", mock.Anything, mock.Anything).Return(batches, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Succeeded",
		"details": [
			{
				"type": "Batch",
				"subtype": "broadcast",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:02Z",
				"id": "` + batches[0].ID.String() + `"
			},
			{
				"type": "BlockchainEvent",
				"subtype": "BatchPin",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:01Z",
				"id": "` + events[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "Operation",
				"subtype": "blockchain_batch_pin",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + ops[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusBatchPinFail(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeBatchPin,
	}
	ops := []*fftypes.Operation{
		{
			Status: fftypes.OpStatusFailed,
			ID:     fftypes.NewUUID(),
			Type:   fftypes.OpTypeBlockchainBatchPin,
			Error:  "complete failure",
		},
	}
	events := []*fftypes.BlockchainEvent{}
	batches := []*fftypes.BatchPersisted{}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetBatches", mock.Anything, mock.Anything).Return(batches, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Failed",
		"details": [
			{
				"type": "Operation",
				"subtype": "blockchain_batch_pin",
				"status": "Failed",
				"id": "` + ops[0].ID.String() + `",
				"error": "complete failure"
			},
			{
				"type": "BlockchainEvent",
				"status": "Pending"
			},
			{
				"type": "Batch",
				"status": "Pending"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusBatchPinPending(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeBatchPin,
	}
	ops := []*fftypes.Operation{
		{
			Status:  fftypes.OpStatusSucceeded,
			ID:      fftypes.NewUUID(),
			Type:    fftypes.OpTypeBlockchainBatchPin,
			Updated: fftypes.UnixTime(0),
		},
	}
	events := []*fftypes.BlockchainEvent{}
	batches := []*fftypes.BatchPersisted{}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetBatches", mock.Anything, mock.Anything).Return(batches, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Pending",
		"details": [
			{
				"type": "BlockchainEvent",
				"status": "Pending"
			},
			{
				"type": "Batch",
				"status": "Pending"
			},
			{
				"type": "Operation",
				"subtype": "blockchain_batch_pin",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + ops[0].ID.String() + `"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTokenPoolSuccess(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenPool,
	}
	ops := []*fftypes.Operation{
		{
			Status:  fftypes.OpStatusSucceeded,
			ID:      fftypes.NewUUID(),
			Type:    fftypes.OpTypeTokenCreatePool,
			Updated: fftypes.UnixTime(0),
			Output:  fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	events := []*fftypes.BlockchainEvent{
		{
			Name:      "TokenPool",
			ID:        fftypes.NewUUID(),
			Timestamp: fftypes.UnixTime(0),
			Info:      fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	pools := []*fftypes.TokenPool{
		{
			ID:      fftypes.NewUUID(),
			Type:    fftypes.TokenTypeFungible,
			Created: fftypes.UnixTime(0),
			State:   fftypes.TokenPoolStateConfirmed,
		},
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetTokenPools", mock.Anything, mock.Anything).Return(pools, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Succeeded",
		"details": [
			{
				"type": "Operation",
				"subtype": "token_create_pool",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + ops[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "BlockchainEvent",
				"subtype": "TokenPool",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + events[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "TokenPool",
				"subtype": "fungible",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + pools[0].ID.String() + `"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTokenPoolPending(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenPool,
	}
	ops := []*fftypes.Operation{
		{
			Status: fftypes.OpStatusSucceeded,
			ID:     fftypes.NewUUID(),
			Type:   fftypes.OpTypeTokenCreatePool,
			Output: fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	events := []*fftypes.BlockchainEvent{}
	pools := []*fftypes.TokenPool{}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetTokenPools", mock.Anything, mock.Anything).Return(pools, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Pending",
		"details": [
			{
				"type": "Operation",
				"subtype": "token_create_pool",
				"status": "Succeeded",
				"id": "` + ops[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "BlockchainEvent",
				"status": "Pending"
			},
			{
				"type": "TokenPool",
				"status": "Pending"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTokenPoolUnconfirmed(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenPool,
	}
	ops := []*fftypes.Operation{
		{
			Status: fftypes.OpStatusSucceeded,
			ID:     fftypes.NewUUID(),
			Type:   fftypes.OpTypeTokenCreatePool,
			Output: fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	events := []*fftypes.BlockchainEvent{}
	pools := []*fftypes.TokenPool{
		{
			ID:      fftypes.NewUUID(),
			Type:    fftypes.TokenTypeFungible,
			Created: fftypes.UnixTime(0),
			State:   fftypes.TokenPoolStatePending,
		},
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetTokenPools", mock.Anything, mock.Anything).Return(pools, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Pending",
		"details": [
			{
				"type": "Operation",
				"subtype": "token_create_pool",
				"status": "Succeeded",
				"id": "` + ops[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "BlockchainEvent",
				"status": "Pending"
			},
			{
				"type": "TokenPool",
				"subtype": "fungible",
				"status": "Pending",
				"id": "` + pools[0].ID.String() + `"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTokenTransferSuccess(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenTransfer,
	}
	ops := []*fftypes.Operation{
		{
			Status:  fftypes.OpStatusSucceeded,
			ID:      fftypes.NewUUID(),
			Type:    fftypes.OpTypeTokenTransfer,
			Updated: fftypes.UnixTime(0),
			Output:  fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	events := []*fftypes.BlockchainEvent{
		{
			Name:      "Mint",
			ID:        fftypes.NewUUID(),
			Timestamp: fftypes.UnixTime(0),
			Info:      fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	transfers := []*fftypes.TokenTransfer{
		{
			LocalID: fftypes.NewUUID(),
			Type:    fftypes.TokenTransferTypeMint,
			Created: fftypes.UnixTime(0),
		},
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetTokenTransfers", mock.Anything, mock.Anything).Return(transfers, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Succeeded",
		"details": [
			{
				"type": "Operation",
				"subtype": "token_transfer",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + ops[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "BlockchainEvent",
				"subtype": "Mint",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + events[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "TokenTransfer",
				"subtype": "mint",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + transfers[0].LocalID.String() + `"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTokenApprovalSuccess(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenApproval,
	}
	ops := []*fftypes.Operation{
		{
			Status:  fftypes.OpStatusSucceeded,
			ID:      fftypes.NewUUID(),
			Type:    fftypes.OpTypeTokenApproval,
			Updated: fftypes.UnixTime(0),
			Output:  fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	events := []*fftypes.BlockchainEvent{
		{
			ID:        fftypes.NewUUID(),
			Timestamp: fftypes.UnixTime(0),
			Info:      fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	approvals := []*fftypes.TokenApproval{
		{
			LocalID: fftypes.NewUUID(),
			Created: fftypes.UnixTime(0),
		},
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetTokenApprovals", mock.Anything, mock.Anything).Return(approvals, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Succeeded",
		"details": [
			{
				"type": "Operation",
				"subtype": "token_approval",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + ops[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "BlockchainEvent",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + events[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "TokenApproval",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + approvals[0].LocalID.String() + `"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTokenTransferPending(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenTransfer,
	}
	ops := []*fftypes.Operation{
		{
			Status: fftypes.OpStatusSucceeded,
			ID:     fftypes.NewUUID(),
			Type:   fftypes.OpTypeTokenTransfer,
			Output: fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	events := []*fftypes.BlockchainEvent{}
	transfers := []*fftypes.TokenTransfer{}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetTokenTransfers", mock.Anything, mock.Anything).Return(transfers, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Pending",
		"details": [
			{
				"type": "Operation",
				"subtype": "token_transfer",
				"status": "Succeeded",
				"id": "` + ops[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "BlockchainEvent",
				"status": "Pending"
			},
			{
				"type": "TokenTransfer",
				"status": "Pending"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTokenTransferRetry(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenTransfer,
	}
	op1ID := fftypes.NewUUID()
	op2ID := fftypes.NewUUID()
	ops := []*fftypes.Operation{
		{
			Status: fftypes.OpStatusFailed,
			ID:     op1ID,
			Type:   fftypes.OpTypeTokenTransfer,
			Retry:  op2ID,
		},
		{
			Status: fftypes.OpStatusPending,
			ID:     op2ID,
			Type:   fftypes.OpTypeTokenTransfer,
		},
	}
	events := []*fftypes.BlockchainEvent{}
	transfers := []*fftypes.TokenTransfer{}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetTokenTransfers", mock.Anything, mock.Anything).Return(transfers, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Pending",
		"details": [
			{
				"type": "Operation",
				"subtype": "token_transfer",
				"status": "Failed",
				"id": "` + op1ID.String() + `"
			},
			{
				"type": "Operation",
				"subtype": "token_transfer",
				"status": "Pending",
				"id": "` + op2ID.String() + `"
			},
			{
				"type": "BlockchainEvent",
				"status": "Pending"
			},
			{
				"type": "TokenTransfer",
				"status": "Pending"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTokenApprovalPending(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenApproval,
	}
	ops := []*fftypes.Operation{
		{
			Status: fftypes.OpStatusSucceeded,
			ID:     fftypes.NewUUID(),
			Type:   fftypes.OpTypeTokenApproval,
			Output: fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	events := []*fftypes.BlockchainEvent{}
	approvals := []*fftypes.TokenApproval{}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)
	or.mdi.On("GetTokenApprovals", mock.Anything, mock.Anything).Return(approvals, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Pending",
		"details": [
			{
				"type": "Operation",
				"subtype": "token_approval",
				"status": "Succeeded",
				"id": "` + ops[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			},
			{
				"type": "BlockchainEvent",
				"status": "Pending"
			},
			{
				"type": "TokenApproval",
				"status": "Pending"
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusContractInvokeSuccess(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeContractInvoke,
	}
	ops := []*fftypes.Operation{
		{
			Status:  fftypes.OpStatusSucceeded,
			ID:      fftypes.NewUUID(),
			Type:    fftypes.OpTypeBlockchainInvoke,
			Updated: fftypes.UnixTime(0),
			Output:  fftypes.JSONObject{"transactionHash": "0x100"},
		},
	}
	events := []*fftypes.BlockchainEvent{}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(ops, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(events, nil, nil)

	status, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.NoError(t, err)

	expectedStatus := compactJSON(`{
		"status": "Succeeded",
		"details": [
			{
				"type": "Operation",
				"subtype": "blockchain_invoke",
				"status": "Succeeded",
				"timestamp": "1970-01-01T00:00:00Z",
				"id": "` + ops[0].ID.String() + `",
				"info": {"transactionHash": "0x100"}
			}
		]
	}`)
	statusJSON, _ := json.Marshal(status)
	assert.Equal(t, expectedStatus, string(statusJSON))

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTXError(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(nil, fmt.Errorf("pop"))

	_, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.EqualError(t, err, "pop")

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusNotFound(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(nil, nil)

	_, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.Regexp(t, "FF10109", err)

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusOpError(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(&fftypes.Transaction{}, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.EqualError(t, err, "pop")

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusBlockchainEventError(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(&fftypes.Transaction{}, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.EqualError(t, err, "pop")

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusBatchError(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeBatchPin,
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetBatches", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.EqualError(t, err, "pop")

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusPoolError(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenPool,
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetTokenPools", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.EqualError(t, err, "pop")

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusTransferError(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenTransfer,
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetTokenTransfers", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.EqualError(t, err, "pop")

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusApprovalError(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: fftypes.TransactionTypeTokenApproval,
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetTokenApprovals", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.EqualError(t, err, "pop")

	or.mdi.AssertExpectations(t)
}

func TestGetTransactionStatusUnknownType(t *testing.T) {
	or := newTestOrchestrator()

	txID := fftypes.NewUUID()
	tx := &fftypes.Transaction{
		Type: "bad",
	}

	or.mdi.On("GetTransactionByID", mock.Anything, txID).Return(tx, nil)
	or.mdi.On("GetOperations", mock.Anything, mock.Anything).Return(nil, nil, nil)
	or.mdi.On("GetBlockchainEvents", mock.Anything, mock.Anything).Return(nil, nil, nil)

	_, err := or.GetTransactionStatus(context.Background(), "ns1", txID.String())
	assert.Regexp(t, "FF10336", err)

	or.mdi.AssertExpectations(t)
}
