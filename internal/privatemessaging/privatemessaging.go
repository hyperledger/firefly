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

package privatemessaging

import (
	"context"

	"github.com/kaleido-io/firefly/internal/batch"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/data"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/kaleido-io/firefly/pkg/identity"
)

type PrivateMessaging interface {
}

type privateMessaging struct {
	groupManager

	ctx          context.Context
	database     database.Plugin
	identity     identity.Plugin
	batch        batch.Manager
	data         data.Manager
	nodeIdentity string
}

func NewPrivateMessaging(ctx context.Context, di database.Plugin, ii identity.Plugin, ba batch.Manager, dm data.Manager) (PrivateMessaging, error) {
	pm := &privateMessaging{
		ctx:      ctx,
		database: di,
		identity: ii,
		batch:    ba,
		data:     dm,
		groupManager: groupManager{
			database: di,
			data:     dm,
		},
		nodeIdentity: config.GetString(config.NodeIdentity),
	}

	bo := batch.Options{
		BatchMaxSize:   config.GetUint(config.PrivateBatchSize),
		BatchTimeout:   config.GetDuration(config.PrivateBatchTimeout),
		DisposeTimeout: config.GetDuration(config.PrivateBatchAgentTimeout),
	}

	ba.RegisterDispatcher([]fftypes.MessageType{
		fftypes.MessageTypeGroupInit,
		fftypes.MessageTypePrivate,
	}, pm.dispatchBatch, bo)

	return pm, nil
}

func (pm *privateMessaging) dispatchBatch(ctx context.Context, batch *fftypes.Batch) error {
	return nil
}
