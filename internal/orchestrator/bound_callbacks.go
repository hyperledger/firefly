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
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
)

type boundCallbacks struct {
	ss sharedstorage.Plugin
	ei events.EventManager
	om operations.Manager
}

func (bc *boundCallbacks) OperationUpdate(update *core.OperationUpdate) {
	bc.om.SubmitOperationUpdate(update)
}

func (bc *boundCallbacks) SharedStorageBatchDownloaded(payloadRef string, data []byte) (*fftypes.UUID, error) {
	return bc.ei.SharedStorageBatchDownloaded(bc.ss, payloadRef, data)
}

func (bc *boundCallbacks) SharedStorageBlobDownloaded(hash fftypes.Bytes32, size int64, payloadRef string) {
	bc.ei.SharedStorageBlobDownloaded(bc.ss, hash, size, payloadRef)
}
