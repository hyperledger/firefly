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

package networkmap

import (
	"context"
	"time"

	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/log"
)

func (nm *networkMap) updateIdentityGauges(ctx context.Context) error {
	_, iRes, iErr := nm.GetIdentitiesGlobal(ctx, database.IdentityQueryFactory.NewFilter(ctx).And())
	if iErr != nil {
		nm.metrics.SetNetworkIdentities(-1, "any")
		return iErr
	}
	nm.metrics.SetNetworkIdentities(*iRes.TotalCount, "any")

	_, oRes, oErr := nm.GetOrganizations(ctx, database.IdentityQueryFactory.NewFilter(ctx).And())
	if oErr != nil {
		nm.metrics.SetNetworkIdentities(-1, "organization")
		return oErr
	}
	nm.metrics.SetNetworkIdentities(*oRes.TotalCount, "organization")

	_, nRes, nErr := nm.GetNodes(ctx, database.IdentityQueryFactory.NewFilter(ctx).And())
	if nErr != nil {
		nm.metrics.SetNetworkIdentities(-1, "node")
		return nErr
	}
	nm.metrics.SetNetworkIdentities(*nRes.TotalCount, "node")

	return nil
}

func (nm *networkMap) networkMetricsLoop() {
	ctx := log.WithLogField(nm.ctx, "role", "networkmap")
	ctx = log.WithLogField(ctx, "loop", "metrics")

	for {
		log.L(ctx).Debug("Sleeping for networkmap metrics loop")
		time.Sleep(30 * time.Second)
		if nm.metrics.IsMetricsEnabled() {
			if err := nm.updateIdentityGauges(ctx); err != nil {
				log.L(ctx).Error("Failed to observe network metrics", err)
			}
		}
	}
}
