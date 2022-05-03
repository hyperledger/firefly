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

package defsender

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastTokenPoolNSGetFail(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	mdm := ds.data.(*datamocks.Manager)

	pool := &fftypes.TokenPoolAnnouncement{
		Pool: &fftypes.TokenPool{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "mypool",
			Type:      fftypes.TokenTypeNonFungible,
			Locator:   "N1",
			Symbol:    "COIN",
		},
	}

	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(fmt.Errorf("pop"))

	_, err := ds.BroadcastTokenPool(context.Background(), "ns1", pool, false)
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
}

func TestBroadcastTokenPoolInvalid(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	mdm := ds.data.(*datamocks.Manager)

	pool := &fftypes.TokenPoolAnnouncement{
		Pool: &fftypes.TokenPool{
			ID:        fftypes.NewUUID(),
			Namespace: "",
			Name:      "",
			Type:      fftypes.TokenTypeNonFungible,
			Locator:   "N1",
			Symbol:    "COIN",
		},
	}

	_, err := ds.BroadcastTokenPool(context.Background(), "ns1", pool, false)
	assert.Regexp(t, "FF00140", err)

	mdm.AssertExpectations(t)
}

func TestBroadcastTokenPoolOk(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	mdm := ds.data.(*datamocks.Manager)
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}

	pool := &fftypes.TokenPoolAnnouncement{
		Pool: &fftypes.TokenPool{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Name:      "mypool",
			Type:      fftypes.TokenTypeNonFungible,
			Locator:   "N1",
			Symbol:    "COIN",
		},
	}

	mim.On("ResolveInputSigningIdentity", mock.Anything, "ns1", mock.Anything).Return(nil)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mbm.On("NewBroadcast", "ns1", mock.Anything).Return(mms)
	mms.On("Send", context.Background()).Return(nil)

	_, err := ds.BroadcastTokenPool(context.Background(), "ns1", pool, false)
	assert.NoError(t, err)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}
