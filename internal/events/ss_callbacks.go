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

package events

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
)

func (em *eventManager) SharedStorageBatchDownloaded(ss sharedstorage.Plugin, ns, payloadRef string, data []byte) (*fftypes.UUID, error) {

	l := log.L(em.ctx)

	// De-serializae the batch
	var batch *fftypes.Batch
	err := json.Unmarshal(data, &batch)
	if err != nil {
		l.Errorf("Invalid batch downloaded from %s '%s': %s", ss.Name(), payloadRef, err)
		return nil, nil
	}
	l.Infof("Shared storage batch downloaded from %s '%s' id=%s (len=%d)", ss.Name(), payloadRef, batch.ID, len(data))

	if batch.Namespace != ns {
		l.Errorf("Invalid batch '%s'. Namespace in batch '%s' does not match pin namespace '%s'", batch.ID, batch.Namespace, ns)
		return nil, nil // This is not retryable. skip this batch
	}

	err = em.retry.Do(em.ctx, "persist batch", func(attempt int) (bool, error) {
		err := em.database.RunAsGroup(em.ctx, func(ctx context.Context) error {
			_, _, err := em.persistBatch(ctx, batch)
			return err
		})
		if err != nil {
			return true, err
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	// Rewind the aggregator to this batch - after the DB updates are complete
	log.L(em.ctx).Errorf("Rewinding for downloaded broadcast batch %s", batch.ID)
	em.aggregator.rewindBatches <- *batch.ID
	return batch.ID, nil
}

func (em *eventManager) SharedStorageBLOBDownloaded(ss sharedstorage.Plugin, hash fftypes.Bytes32, size int64, payloadRef string) error {
	l := log.L(em.ctx)
	l.Infof("Blob received event from public storage %s: Hash='%v'", ss.Name(), hash)

	return em.blobReceivedCommon("", hash, size, payloadRef)
}
