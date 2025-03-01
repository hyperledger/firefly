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
	"errors"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/multiparty"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/multipartymocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestCheckNodeIdentityStatusSuccess(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	ctx := context.TODO()

	nm.multiparty.(*multipartymocks.Manager).On("LocalNode").Return(multiparty.LocalNode{
		Name: "local-node",
	})

	matchingProfile := fftypes.JSONAnyPtr(`{"cert": "a-cert" }`).JSONObject()
	node := &core.Identity{
		IdentityProfile: core.IdentityProfile{
			Profile: matchingProfile,
		},
	}
	nm.identity.(*identitymanagermocks.Manager).On("GetLocalNode", ctx).Return(node, nil)

	nm.exchange.(*dataexchangemocks.Plugin).On("GetEndpointInfo", ctx, "local-node").Return(matchingProfile, nil)
	nm.exchange.(*dataexchangemocks.Plugin).On("CheckNodeIdentityStatus", ctx, matchingProfile, node).Return(nil)

	err := nm.CheckNodeIdentityStatus(context.TODO())
	assert.NoError(t, err)
}

func TestCheckNodeIdentityStatusEndpointInfoFails(t *testing.T) {

	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	ctx := context.TODO()

	nm.multiparty.(*multipartymocks.Manager).On("LocalNode").Return(multiparty.LocalNode{
		Name: "local-node",
	})

	nodeProfile := fftypes.JSONAnyPtr(`{"cert": "a-cert" }`).JSONObject()

	nm.identity.(*identitymanagermocks.Manager).On("GetLocalNode", ctx).Return(&core.Identity{
		IdentityProfile: core.IdentityProfile{
			Profile: nodeProfile,
		},
	}, nil)

	nm.exchange.(*dataexchangemocks.Plugin).On("GetEndpointInfo", ctx, "local-node").Return(nil, errors.New("endpoint info went pop"))

	err := nm.CheckNodeIdentityStatus(context.TODO())
	assert.Error(t, err)
}
