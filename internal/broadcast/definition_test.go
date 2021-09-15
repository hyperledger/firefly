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

	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastDefinitionAsNodeUpsertFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))
	_, err := bm.BroadcastDefinitionAsNode(bm.ctx, &fftypes.Namespace{}, fftypes.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)
}

func TestBroadcastDefinitionBadIdentity(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	mim := bm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := bm.BroadcastDefinition(bm.ctx, &fftypes.Namespace{}, &fftypes.Identity{
		Author: "wrong",
		Key:    "wrong",
	}, fftypes.SystemTagDefineNamespace, false)
	assert.Regexp(t, "pop", err)
}
