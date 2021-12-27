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

package orchestrator

import (
	"context"

	"github.com/hyperledger/firefly/internal/events/system"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (or *orchestrator) CreateSubscription(ctx context.Context, ns string, subDef *fftypes.Subscription) (*fftypes.Subscription, error) {
	return or.createUpdateSubscription(ctx, ns, subDef, true)
}

func (or *orchestrator) CreateUpdateSubscription(ctx context.Context, ns string, subDef *fftypes.Subscription) (*fftypes.Subscription, error) {
	return or.createUpdateSubscription(ctx, ns, subDef, false)
}

func (or *orchestrator) createUpdateSubscription(ctx context.Context, ns string, subDef *fftypes.Subscription, mustNew bool) (*fftypes.Subscription, error) {
	subDef.ID = fftypes.NewUUID()
	subDef.Created = fftypes.Now()
	subDef.Namespace = ns
	subDef.Ephemeral = false
	if err := or.data.VerifyNamespaceExists(ctx, subDef.Namespace); err != nil {
		return nil, err
	}
	if err := fftypes.ValidateFFNameFieldNoUUID(ctx, subDef.Name, "name"); err != nil {
		return nil, err
	}
	if subDef.Transport == system.SystemEventsTransport {
		return nil, i18n.NewError(ctx, i18n.MsgSystemTransportInternal)
	}

	return subDef, or.events.CreateUpdateDurableSubscription(ctx, subDef, mustNew)
}

func (or *orchestrator) DeleteSubscription(ctx context.Context, ns, id string) error {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return err
	}
	sub, err := or.database.GetSubscriptionByID(ctx, u)
	if err != nil {
		return err
	}
	if sub == nil || sub.Namespace != ns {
		return i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	return or.events.DeleteDurableSubscription(ctx, sub)
}

func (or *orchestrator) GetSubscriptions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Subscription, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetSubscriptions(ctx, filter)
}

func (or *orchestrator) GetSubscriptionByID(ctx context.Context, ns, id string) (*fftypes.Subscription, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetSubscriptionByID(ctx, u)
}
