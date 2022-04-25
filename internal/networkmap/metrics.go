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
	"github.com/hyperledger/firefly/pkg/database"
)

func (nm *networkMap) UpdateIdentityGauges(ctx context.Context) {
	if nm.metrics.IsMetricsEnabled() {
		_, iRes, iErr := nm.GetIdentitiesGlobal(ctx, database.IdentityQueryFactory.NewFilter(ctx).And())
		if iErr != nil {
			nm.metrics.SetNetworkIdentities(-1, "any")
		}
		nm.metrics.SetNetworkIdentities(*iRes.TotalCount, "any")

		_, oRes, oErr := nm.GetOrganizations(ctx, database.IdentityQueryFactory.NewFilter(ctx).And())
		if oErr != nil {
			nm.metrics.SetNetworkIdentities(-1, "organization")
		}
		nm.metrics.SetNetworkIdentities(*oRes.TotalCount, "organization")

		_, nRes, nErr := nm.GetNodes(ctx, database.IdentityQueryFactory.NewFilter(ctx).And())
		if nErr != nil {
			nm.metrics.SetNetworkIdentities(-1, "node")
		}
		nm.metrics.SetNetworkIdentities(*nRes.TotalCount, "node")
	}
}
