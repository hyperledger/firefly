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

package events

import (
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
)

func (em *eventManager) TokenPoolCreated(tk tokens.Plugin, pool *fftypes.TokenPool, signingIdentity string, additionalInfo fftypes.JSONObject) error {

	log.L(em.ctx).Infof("-> TokenPoolCreated id=%s author=%s", pool.ID, signingIdentity)
	defer func() {
		log.L(em.ctx).Infof("<- TokenPoolCreated id=%s author=%s", pool.ID, signingIdentity)
	}()
	log.L(em.ctx).Tracef("TokenPoolCreated info: %+v", additionalInfo)

	if err := fftypes.ValidateFFNameField(em.ctx, pool.Namespace, "namespace"); err != nil {
		log.L(em.ctx).Errorf("Invalid pool '%s' - invalid namespace '%s': %a", pool.ID, pool.Namespace, err)
		return err
	}

	if err := em.database.UpsertTokenPool(em.ctx, pool, false); err != nil {
		return err
	}

	event := fftypes.NewEvent(fftypes.EventTypePoolConfirmed, pool.Namespace, pool.ID)
	return em.database.InsertEvent(em.ctx, event)
}
