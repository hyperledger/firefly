// Copyright © 2025 Kaleido, Inc.
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

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
)

func (nm *networkMap) CheckNodeIdentityStatus(ctx context.Context) error {
	localNodeName := nm.multiparty.LocalNode().Name
	if localNodeName == "" {
		return i18n.NewError(ctx, coremsgs.MsgNodeAndOrgIDMustBeSet)
	}

	node, err := nm.identity.GetLocalNode(ctx)
	if err != nil {
		return err
	}

	if node.Profile == nil {
		return i18n.NewError(ctx, coremsgs.MsgNoRegistrationMessageData) // TODO
	}

	dxPeer, err := nm.exchange.GetEndpointInfo(ctx, localNodeName)
	if err != nil {
		return err
	}

	return nm.exchange.CheckNodeIdentityStatus(ctx, dxPeer, node)
}
