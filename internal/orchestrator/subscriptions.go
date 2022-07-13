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
	"context"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

func (or *orchestrator) CreateSubscription(ctx context.Context, subDef *core.Subscription) (*core.Subscription, error) {
	return or.createUpdateSubscription(ctx, subDef, true)
}

func (or *orchestrator) CreateUpdateSubscription(ctx context.Context, subDef *core.Subscription) (*core.Subscription, error) {
	return or.createUpdateSubscription(ctx, subDef, false)
}

func (or *orchestrator) createUpdateSubscription(ctx context.Context, subDef *core.Subscription, mustNew bool) (*core.Subscription, error) {
	subDef.ID = fftypes.NewUUID()
	subDef.Created = fftypes.Now()
	subDef.Namespace = or.namespace.LocalName
	subDef.Ephemeral = false
	if err := fftypes.ValidateFFNameFieldNoUUID(ctx, subDef.Name, "name"); err != nil {
		return nil, err
	}
	if subDef.Transport == system.SystemEventsTransport {
		return nil, i18n.NewError(ctx, coremsgs.MsgSystemTransportInternal)
	}

	return subDef, or.events.CreateUpdateDurableSubscription(ctx, subDef, mustNew)
}

func (or *orchestrator) DeleteSubscription(ctx context.Context, id string) error {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return err
	}
	sub, err := or.database().GetSubscriptionByID(ctx, or.namespace.LocalName, u)
	if err != nil {
		return err
	}
	if sub == nil {
		return i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	return or.events.DeleteDurableSubscription(ctx, sub)
}

func (or *orchestrator) GetSubscriptions(ctx context.Context, filter database.AndFilter) ([]*core.Subscription, *database.FilterResult, error) {
	return or.database().GetSubscriptions(ctx, or.namespace.LocalName, filter)
}

func (or *orchestrator) GetSubscriptionByID(ctx context.Context, id string) (*core.Subscription, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	return or.database().GetSubscriptionByID(ctx, or.namespace.LocalName, u)
}
