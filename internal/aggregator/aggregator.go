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
	"time"

	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/database"
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
	ctx      context.Context
	p2pfs    p2pfs.Plugin
	database database.Plugin
	retry    retry.Retry
}

func NewAggregator(ctx context.Context, p2pfs p2pfs.Plugin, database database.Plugin) Aggregator {
	return &aggregator{
		ctx:      log.WithLogField(ctx, "role", "aggregator"),
		p2pfs:    p2pfs,
		database: database,
		retry: retry.Retry{
			InitialDelay: time.Duration(config.GetUint(config.AggregatorDataReadRetryDelayMS)) * time.Millisecond,
			MaximumDelay: time.Duration(config.GetUint(config.AggregatorDataReadRetryMaxDelayMS)) * time.Millisecond,
			Factor:       config.GetFloat64(config.AggregatorDataReadRetryMaxDelayMS),
		},
	}
}

func (a *aggregator) TransactionUpdate(txTrackingID string, txState fftypes.TransactionStatus, protocolTxId, errorMessage string, additionalInfo map[string]interface{}) error {
	return nil
}
