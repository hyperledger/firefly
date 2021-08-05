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

package networkmap

import (
	"context"

	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (nm *networkMap) GetOrganizationByID(ctx context.Context, id string) (*fftypes.Organization, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	return nm.database.GetOrganizationByID(ctx, u)
}

func (nm *networkMap) GetOrganizations(ctx context.Context, filter database.AndFilter) ([]*fftypes.Organization, *database.FilterResult, error) {
	return nm.database.GetOrganizations(ctx, filter)
}

func (nm *networkMap) GetNodeByID(ctx context.Context, id string) (*fftypes.Node, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	return nm.database.GetNodeByID(ctx, u)
}

func (nm *networkMap) GetNodes(ctx context.Context, filter database.AndFilter) ([]*fftypes.Node, *database.FilterResult, error) {
	return nm.database.GetNodes(ctx, filter)
}
