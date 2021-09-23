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

package broadcast

import (
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/identitymocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastDefinitionAsNodeBadId(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	config.Set(config.OrgIdentity, "wrong")
	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "wrong").Return(nil, fmt.Errorf("pop"))
	_, err := bm.broadcastDefinitionAsNode(bm.ctx, &fftypes.Namespace{}, fftypes.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)
}

func TestBroadcastDefinitionAsNodeBadSigningId(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	config.Set(config.OrgIdentity, "wrong")
	mii := bm.identity.(*identitymocks.Plugin)
	mbi := bm.blockchain.(*blockchainmocks.Plugin)
	badID := &fftypes.Identity{OnChain: "0x99999"}
	mii.On("Resolve", mock.Anything, "wrong").Return(badID, nil)
	mbi.On("VerifyIdentitySyntax", mock.Anything, badID).Return(fmt.Errorf("pop"))
	_, err := bm.broadcastDefinitionAsNode(bm.ctx, &fftypes.Namespace{}, fftypes.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)
}
