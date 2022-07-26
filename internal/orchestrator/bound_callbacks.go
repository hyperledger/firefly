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
	"context"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/events"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
)

type boundCallbacks struct {
	dx dataexchange.Plugin
	ss sharedstorage.Plugin
	ei events.EventManager
	om operations.Manager
}

func (bc *boundCallbacks) OperationUpdate(plugin core.Named, nsOpID string, status core.OpStatus, blockchainTXID, errorMessage string, opOutput fftypes.JSONObject) {
	bc.om.SubmitOperationUpdate(plugin, &operations.OperationUpdate{
		NamespacedOpID: nsOpID,
		Status:         status,
		BlockchainTXID: blockchainTXID,
		ErrorMessage:   errorMessage,
		Output:         opOutput,
	})
}

func (bc *boundCallbacks) DXEvent(ctx context.Context, event dataexchange.DXEvent) {
	switch event.Type() {
	case dataexchange.DXEventTypeTransferResult:
		tr := event.TransferResult()
		log.L(ctx).Infof("Transfer result %s=%s error='%s' manifest='%s' info='%s'", tr.TrackingID, tr.Status, tr.Error, tr.Manifest, tr.Info)

		opUpdate := &operations.OperationUpdate{
			NamespacedOpID: event.NamespacedID(),
			Status:         tr.Status,
			VerifyManifest: bc.dx.Capabilities().Manifest,
			ErrorMessage:   tr.Error,
			Output:         tr.Info,
			OnComplete: func() {
				event.Ack()
			},
		}

		// Pass manifest verification code to the background worker, for once it has loaded the operation
		if opUpdate.VerifyManifest {
			if tr.Manifest != "" {
				// For batches DX passes us a manifest to compare.
				opUpdate.DXManifest = tr.Manifest
			} else if tr.Hash != "" {
				// For blobs DX passes us a hash to compare.
				opUpdate.DXHash = tr.Hash
			}
		}

		bc.om.SubmitOperationUpdate(bc.dx, opUpdate)
	default:
		bc.ei.DXEvent(bc.dx, event)
	}
}

func (bc *boundCallbacks) SharedStorageBatchDownloaded(payloadRef string, data []byte) (*fftypes.UUID, error) {
	return bc.ei.SharedStorageBatchDownloaded(bc.ss, payloadRef, data)
}

func (bc *boundCallbacks) SharedStorageBlobDownloaded(hash fftypes.Bytes32, size int64, payloadRef string) {
	bc.ei.SharedStorageBlobDownloaded(bc.ss, hash, size, payloadRef)
}
