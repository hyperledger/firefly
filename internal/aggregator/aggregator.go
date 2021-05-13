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

package aggregator

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/gofrs/uuid"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/p2pfs"
	"github.com/kaleido-io/firefly/internal/retry"
)

type Aggregator interface {
	blockchain.Events
	// TODO: Add dataexchange events
}

type aggregator struct {
	ctx   context.Context
	p2pfs p2pfs.Plugin
	retry retry.Retry
}

func NewAggregator(ctx context.Context, p2pfs p2pfs.Plugin) Aggregator {
	return &aggregator{
		ctx:   log.WithLogField(ctx, "role", "aggregator"),
		p2pfs: p2pfs,
		retry: retry.Retry{
			InitialDelay: time.Duration(config.GetUint(config.AggregatorDataReadRetryDelayMS)),
			MaximumDelay: time.Duration(config.GetUint(config.AggregatorDataReadRetryMaxDelayMS)),
			Factor:       config.GetFloat64(config.AggregatorDataReadRetryMaxDelayMS),
		},
	}
}

func (a *aggregator) TransactionUpdate(txTrackingID string, txState fftypes.TransactionStatus, protocolTxId, errorMessage string, additionalInfo map[string]interface{}) error {
	return nil
}

// SequencedBroadcastBatch is called in-line with a particular ledger's stream of events, so while we
// block here this blockchain event remains un-acknowledged, and no further events will arrive from this
// particular ledger.
//
// We must block here long enough to get the payload from the p2pfs, persist the messages in the correct
// sequence, and also persist all the data.
func (a *aggregator) SequencedBroadcastBatch(batch *blockchain.BroadcastBatch, author string, protocolTxId string, additionalInfo map[string]interface{}) error {

	var batchID uuid.UUID
	copy(batchID[:], batch.BatchID[0:16])

	var body io.ReadCloser
	if err := a.retry.Do(a.ctx, func(attempt int) (retry bool, err error) {
		body, err = a.p2pfs.RetrieveData(a.ctx, batch.BatchPaylodRef)
		return err != nil, err // retry indefinitely (until context closes)
	}); err != nil {
		return err
	}
	defer body.Close()

	var batchBody fftypes.Batch
	err := json.NewDecoder(body).Decode(&batchBody)
	if err != nil {
		log.L(a.ctx).Errorf("Failed to parse payload referred in batch ID '%s' from transaction '%s'", batchID, protocolTxId)
		return nil // swallow bad data
	}

	return nil
}
