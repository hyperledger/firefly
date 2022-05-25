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
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastTokenPoolNSGetFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)

	pool := &core.TokenPoolAnnouncement{
		Pool: &core.TokenPool{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "mypool",
			Type:      core.TokenTypeNonFungible,
			Locator:   "N1",
			Symbol:    "COIN",
		},
	}

	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(fmt.Errorf("pop"))

	_, err := bm.BroadcastTokenPool(context.Background(), "ns1", pool, false)
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
}

func TestBroadcastTokenPoolInvalid(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)

	pool := &core.TokenPoolAnnouncement{
		Pool: &core.TokenPool{
			ID:        fftypes.NewUUID(),
			Namespace: "",
			Name:      "",
			Type:      core.TokenTypeNonFungible,
			Locator:   "N1",
			Symbol:    "COIN",
		},
	}

	_, err := bm.BroadcastTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "FF00140", err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastTokenPoolBroadcastFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	pool := &core.TokenPoolAnnouncement{
		Pool: &core.TokenPool{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "mypool",
			Type:      core.TokenTypeNonFungible,
			Locator:   "N1",
			Symbol:    "COIN",
		},
	}

	mim.On("ResolveInputSigningIdentity", mock.Anything, "ns1", mock.Anything).Return(nil)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("WriteNewMessage", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := bm.BroadcastTokenPool(context.Background(), "ns1", pool, false)
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBroadcastTokenPoolOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	pool := &core.TokenPoolAnnouncement{
		Pool: &core.TokenPool{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "mypool",
			Type:      core.TokenTypeNonFungible,
			Locator:   "N1",
			Symbol:    "COIN",
		},
	}

	mim.On("ResolveInputSigningIdentity", mock.Anything, "ns1", mock.Anything).Return(nil)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("WriteNewMessage", mock.Anything, mock.Anything).Return(nil)

	_, err := bm.BroadcastTokenPool(context.Background(), "ns1", pool, false)
	assert.NoError(t, err)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}
