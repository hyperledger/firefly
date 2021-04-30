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

package broadcast

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/persistence"
)

type Broadcast struct {
	ctx         context.Context
	persistence persistence.Plugin
	blockchain  blockchain.Plugin
}

var instance *Broadcast

func Init(ctx context.Context, persistence persistence.Plugin, blockchain blockchain.Plugin) error {
	instance = &Broadcast{
		ctx:         ctx,
		persistence: persistence,
		blockchain:  blockchain,
	}

	return nil
}

func GetInstance() *Broadcast {
	return instance
}

func (b *Broadcast) BroadcastMessage(ctx context.Context, identity string, msg *fftypes.MessageRefsOnly) error {

	// TODO: Port batching to Go channels
	batchID := uuid.New()
	batch := &fftypes.Batch{
		ID:      &batchID,
		Author:  identity,
		Created: time.Now().UnixNano(),
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.MessageRefsOnly{
				msg,
			},
		},
	}
	batch.Hash = batch.Payload.Hash()

	if err := b.persistence.UpsertBatch(ctx, batch); err != nil {
		return err
	}

	_, err := b.blockchain.SubmitBroadcastBatch(ctx, batch.Author, &blockchain.BroadcastBatch{
		Timestamp:      batch.Created,
		BatchID:        fftypes.HexUUIDFromUUID(*batch.ID),
		BatchPaylodRef: *batch.Hash, // TODO: This will move to being an IPFS ID, that's obtained by a publish to IPFS
	})
	if err != nil {
		return err
	}

	return nil
}
