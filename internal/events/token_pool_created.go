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
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
)

func (em *eventManager) validateTokenPool(pool *fftypes.TokenPool) error {
	l := log.L(em.ctx)

	if err := fftypes.ValidateFFNameField(em.ctx, pool.Namespace, "namespace"); err != nil {
		l.Errorf("Invalid pool '%s' - invalid namespace '%s': %a", pool.ID, pool.Namespace, err)
		return err
	}
	if err := fftypes.ValidateFFNameField(em.ctx, pool.Name, "name"); err != nil {
		l.Errorf("Invalid pool '%s' - invalid name '%s': %a", pool.ID, pool.Name, err)
		return err
	}

	return nil
}

func (em *eventManager) TokenPoolCreated(tk tokens.Plugin, pool *fftypes.TokenPool, signingIdentity string, additionalInfo fftypes.JSONObject) error {
	valid := true
	err := em.validateTokenPool(pool)
	if err == nil {
		err = em.database.UpsertTokenPool(em.ctx, pool, false)
		if err == database.IDMismatch {
			valid = false
		}
	} else {
		valid = false
	}

	switch {
	case !valid:
		log.L(em.ctx).Warnf("Token pool rejected id=%s author=%s", pool.ID, signingIdentity)
		event := fftypes.NewEvent(fftypes.EventTypePoolRejected, pool.Namespace, pool.ID)
		return em.database.InsertEvent(em.ctx, event)
	case err == nil:
		log.L(em.ctx).Infof("Token pool created id=%s author=%s", pool.ID, signingIdentity)
		event := fftypes.NewEvent(fftypes.EventTypePoolConfirmed, pool.Namespace, pool.ID)
		return em.database.InsertEvent(em.ctx, event)
	default:
		return err
	}
}
