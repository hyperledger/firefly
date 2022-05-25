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

package core

import "github.com/hyperledger/firefly-common/pkg/fftypes"

// OpType describes mechanical steps in the process that have to be performed,
// might be asynchronous, and have results in the back-end systems that might need
// to be correlated with messages by operators.
type OpType = FFEnum

var (
	// OpTypeBlockchainPinBatch is a blockchain transaction to pin a batch
	OpTypeBlockchainPinBatch = ffEnum("optype", "blockchain_pin_batch")
	// OpTypeBlockchainInvoke is a smart contract invoke
	OpTypeBlockchainInvoke = ffEnum("optype", "blockchain_invoke")
	// OpTypeSharedStorageUploadBatch is a shared storage operation to upload broadcast data
	OpTypeSharedStorageUploadBatch = ffEnum("optype", "sharedstorage_upload_batch")
	// OpTypeSharedStorageUploadBlob is a shared storage operation to upload blob data
	OpTypeSharedStorageUploadBlob = ffEnum("optype", "sharedstorage_upload_blob")
	// OpTypeSharedStorageDownloadBatch is a shared storage operation to download broadcast data
	OpTypeSharedStorageDownloadBatch = ffEnum("optype", "sharedstorage_download_batch")
	// OpTypeSharedStorageDownloadBlob is a shared storage operation to download broadcast data
	OpTypeSharedStorageDownloadBlob = ffEnum("optype", "sharedstorage_download_blob")
	// OpTypeDataExchangeSendBatch is a private send of a batch
	OpTypeDataExchangeSendBatch = ffEnum("optype", "dataexchange_send_batch")
	// OpTypeDataExchangeSendBlob is a private send of a blob
	OpTypeDataExchangeSendBlob = ffEnum("optype", "dataexchange_send_blob")
	// OpTypeTokenCreatePool is a token pool creation
	OpTypeTokenCreatePool = ffEnum("optype", "token_create_pool")
	// OpTypeTokenActivatePool is a token pool activation
	OpTypeTokenActivatePool = ffEnum("optype", "token_activate_pool")
	// OpTypeTokenTransfer is a token transfer
	OpTypeTokenTransfer = ffEnum("optype", "token_transfer")
	// OpTypeTokenApproval is a token approval
	OpTypeTokenApproval = ffEnum("optype", "token_approval")
)

// OpStatus is the current status of an operation
type OpStatus string

const (
	// OpStatusPending indicates the operation has been submitted, but is not yet confirmed as successful or failed
	OpStatusPending OpStatus = "Pending"
	// OpStatusSucceeded the infrastructure runtime has returned success for the operation
	OpStatusSucceeded OpStatus = "Succeeded"
	// OpStatusFailed happens when an error is reported by the infrastructure runtime
	OpStatusFailed OpStatus = "Failed"
)

type Named interface {
	Name() string
}

// NewOperation creates a new operation in a transaction
func NewOperation(plugin Named, namespace string, tx *fftypes.UUID, opType OpType) *Operation {
	now := fftypes.Now()
	return &Operation{
		ID:          fftypes.NewUUID(),
		Namespace:   namespace,
		Plugin:      plugin.Name(),
		Transaction: tx,
		Type:        opType,
		Status:      OpStatusPending,
		Created:     now,
		Updated:     now,
	}
}

// Operation is a description of an action performed as part of a transaction submitted by this node
type Operation struct {
	ID          *fftypes.UUID      `ffstruct:"Operation" json:"id" ffexcludeinput:"true"`
	Namespace   string             `ffstruct:"Operation" json:"namespace" ffexcludeinput:"true"`
	Transaction *fftypes.UUID      `ffstruct:"Operation" json:"tx" ffexcludeinput:"true"`
	Type        OpType             `ffstruct:"Operation" json:"type" ffenum:"optype" ffexcludeinput:"true"`
	Status      OpStatus           `ffstruct:"Operation" json:"status"`
	Plugin      string             `ffstruct:"Operation" json:"plugin" ffexcludeinput:"true"`
	Input       fftypes.JSONObject `ffstruct:"Operation" json:"input,omitempty" ffexcludeinput:"true"`
	Output      fftypes.JSONObject `ffstruct:"Operation" json:"output,omitempty"`
	Error       string             `ffstruct:"Operation" json:"error,omitempty"`
	Created     *fftypes.FFTime    `ffstruct:"Operation" json:"created,omitempty" ffexcludeinput:"true"`
	Updated     *fftypes.FFTime    `ffstruct:"Operation" json:"updated,omitempty" ffexcludeinput:"true"`
	Retry       *fftypes.UUID      `ffstruct:"Operation" json:"retry,omitempty" ffexcludeinput:"true"`
}

// PreparedOperation is an operation that has gathered all the raw data ready to send to a plugin
// It is never stored, but it should always be possible for the owning Manager to generate a
// PreparedOperation from an Operation. Data is defined by the Manager, but should be JSON-serializable
// to support inspection and debugging.
type PreparedOperation struct {
	ID        *fftypes.UUID `json:"id"`
	Namespace string        `json:"namespace"`
	Type      OpType        `json:"type" ffenum:"optype"`
	Data      interface{}   `json:"data"`
}
