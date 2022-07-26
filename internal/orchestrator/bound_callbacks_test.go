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
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/eventmocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBoundCallbacks(t *testing.T) {
	mei := &eventmocks.EventManager{}
	mbi := &blockchainmocks.Plugin{}
	mdx := &dataexchangemocks.Plugin{}
	mss := &sharedstoragemocks.Plugin{}
	mom := &operationmocks.Manager{}
	bc := boundCallbacks{dx: mdx, ei: mei, ss: mss, om: mom}

	info := fftypes.JSONObject{"hello": "world"}
	hash := fftypes.NewRandB32()
	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()

	mom.On("SubmitOperationUpdate", mock.Anything, &operations.OperationUpdate{
		NamespacedOpID: nsOpID,
		Status:         core.OpStatusFailed,
		BlockchainTXID: "0xffffeeee",
		ErrorMessage:   "error info",
		Output:         info,
	}).Return().Once()
	bc.OperationUpdate(mbi, nsOpID, core.OpStatusFailed, "0xffffeeee", "error info", info)

	mei.On("SharedStorageBatchDownloaded", mss, "payload1", []byte(`{}`)).Return(nil, fmt.Errorf("pop"))
	_, err := bc.SharedStorageBatchDownloaded("payload1", []byte(`{}`))
	assert.EqualError(t, err, "pop")

	mei.On("SharedStorageBlobDownloaded", mss, *hash, int64(12345), "payload1").Return()
	bc.SharedStorageBlobDownloaded(*hash, 12345, "payload1")

	mei.AssertExpectations(t)
	mbi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mss.AssertExpectations(t)
	mom.AssertExpectations(t)
}

func TestBoundCallbacksDXEvent(t *testing.T) {
	mei := &eventmocks.EventManager{}
	mdx := &dataexchangemocks.Plugin{}
	mss := &sharedstoragemocks.Plugin{}
	mom := &operationmocks.Manager{}
	bc := boundCallbacks{dx: mdx, ei: mei, ss: mss, om: mom}

	ctx := context.Background()
	info := fftypes.JSONObject{"hello": "world"}
	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()

	mdx.On("Capabilities").Return(&dataexchange.Capabilities{
		Manifest: true,
	})

	event1 := &dataexchangemocks.DXEvent{}
	mei.On("DXEvent", mdx, event1).Return().Once()
	event1.On("Type").Return(dataexchange.DXEventTypeMessageReceived).Once()
	bc.DXEvent(ctx, event1)
	event1.AssertExpectations(t)

	event2 := &dataexchangemocks.DXEvent{}
	event2.On("Type").Return(dataexchange.DXEventTypeTransferResult).Once()
	event2.On("TransferResult").Return(&dataexchange.TransferResult{
		TrackingID: opID.String(),
		Status:     core.OpStatusSucceeded,
		TransportStatusUpdate: core.TransportStatusUpdate{
			Info:     info,
			Manifest: "Sally",
		},
	})
	event2.On("NamespacedID").Return(nsOpID)
	event2.On("Ack").Return()
	mom.On("SubmitOperationUpdate", mock.Anything, mock.MatchedBy(func(update *operations.OperationUpdate) bool {
		if update.NamespacedOpID == nsOpID &&
			update.Status == core.OpStatusSucceeded &&
			update.VerifyManifest &&
			update.DXManifest == "Sally" &&
			update.DXHash == "" {
			update.OnComplete()
			return true
		}
		return false
	})).Return().Once()
	bc.DXEvent(ctx, event2)
	event2.AssertExpectations(t)

	event3 := &dataexchangemocks.DXEvent{}
	event3.On("Type").Return(dataexchange.DXEventTypeTransferResult).Once()
	event3.On("TransferResult").Return(&dataexchange.TransferResult{
		TrackingID: opID.String(),
		Status:     core.OpStatusSucceeded,
		TransportStatusUpdate: core.TransportStatusUpdate{
			Info: info,
			Hash: "hash1",
		},
	})
	event3.On("NamespacedID").Return(nsOpID)
	event3.On("Ack").Return()
	mom.On("SubmitOperationUpdate", mock.Anything, mock.MatchedBy(func(update *operations.OperationUpdate) bool {
		if update.NamespacedOpID == nsOpID &&
			update.Status == core.OpStatusSucceeded &&
			update.VerifyManifest &&
			update.DXManifest == "" &&
			update.DXHash == "hash1" {
			update.OnComplete()
			return true
		}
		return false
	})).Return().Once()
	bc.DXEvent(ctx, event3)
	event3.AssertExpectations(t)

	mei.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mss.AssertExpectations(t)
	mom.AssertExpectations(t)
}
