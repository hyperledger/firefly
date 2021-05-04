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
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/kaleido-io/firefly/internal/batching"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/p2pfs"
	"github.com/kaleido-io/firefly/internal/persistence"
)

type Broadcast interface {
	BroadcastMessage(ctx context.Context, identity string, msg *fftypes.MessageRefsOnly, data ...*fftypes.Data) error
	Close()
}

type broadcast struct {
	ctx         context.Context
	persistence persistence.Plugin
	blockchain  blockchain.Plugin
	p2pfs       p2pfs.Plugin
	batch       batching.BatchManager
}

func NewBroadcast(ctx context.Context, persistence persistence.Plugin, blockchain blockchain.Plugin, p2pfs p2pfs.Plugin, batch batching.BatchManager) (Broadcast, error) {
	if persistence == nil || blockchain == nil || batch == nil || p2pfs == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	b := &broadcast{
		ctx:         ctx,
		persistence: persistence,
		blockchain:  blockchain,
		p2pfs:       p2pfs,
		batch:       batch,
	}
	batch.RegisterDispatcher(fftypes.BatchTypeBroadcast, b.dispatchBatch, batching.BatchOptions{
		BatchMaxSize:   config.GetUint(config.BroadcastBatchSize),
		BatchTimeout:   time.Duration(config.GetUint(config.BroadcastBatchTimeout)) * time.Millisecond,
		DisposeTimeout: time.Duration(config.GetUint(config.BroadcastBatchAgentTimeout)) * time.Millisecond,
	})
	return b, nil
}

func (b *broadcast) dispatchBatch(ctx context.Context, batch *fftypes.Batch) error {
	// Serialize the full payload, which has already been sealed for us by the BatchManager
	payload, err := json.Marshal(batch)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}
	// Write it to IPFS etc. to get a payload reference hash (might not be the sha256 data hash)
	// Note the BatchManager is responsible for writing the updated batch
	batch.PayloadRef, err = b.p2pfs.PublishData(ctx, bytes.NewReader(payload))
	return err
}

func (b *broadcast) BroadcastMessage(ctx context.Context, identity string, msg *fftypes.MessageRefsOnly, data ...*fftypes.Data) error {

	// Load all the data - must all be present for us to send
	for _, dataRef := range msg.Data {
		if dataRef.ID == nil {
			continue
		}
		var supplied bool
		for _, d := range data {
			if d.ID != nil && *d.ID == *dataRef.ID {
				supplied = true
				break
			}
		}
		if !supplied {
			d, err := b.persistence.GetDataById(ctx, dataRef.ID)
			if err != nil {
				return err
			}
			if d == nil {
				return i18n.NewError(ctx, i18n.MsgDataNotFound, dataRef.ID)
			}
			data = append(data, d)
		}
	}

	// Write the message and all the data to a broadcast batch
	batchID, err := b.batch.DispatchMessage(ctx, fftypes.BatchTypeBroadcast, msg, data...)
	if err != nil {
		return err
	}
	log.L(ctx).Infof("Added broadcast message %s to batch %s", msg.Header.ID, batchID)

	// TODO: The blockchain bit

	return nil
}

func (b *broadcast) Close() {}
