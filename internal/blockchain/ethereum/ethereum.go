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

package ethereum

import (
	"context"

	"github.com/kaleido-io/firefly/internal/blockchain"
)

type Ethereum struct {
	// ctx context.Context
	// cfg *Config
}

func (e *Ethereum) ConfigInterface() interface{} { return &Ethereum{} }

func (e *Ethereum) Init(ctx context.Context, config interface{}, events blockchain.Events) (*blockchain.Capabilities, error) {
	return &blockchain.Capabilities{}, nil
}

func (e *Ethereum) SubmitBroadcastBatch(identity string, broadcast blockchain.BroadcastBatch) (txTrackingID string, err error) {
	return "", nil
}
