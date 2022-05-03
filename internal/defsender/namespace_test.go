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
	"strings"
	"testing"

	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastNamespaceBadName(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	_, err := ds.BroadcastNamespace(context.Background(), &fftypes.Namespace{
		Name: "!ns",
	}, false)
	assert.Regexp(t, "FF00140.*name", err)
}

func TestBroadcastNamespaceDescriptionTooLong(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()

	buff := strings.Builder{}
	buff.Grow(4097)
	for i := 0; i < 4097; i++ {
		buff.WriteByte(byte('a' + i%26))
	}
	_, err := ds.BroadcastNamespace(context.Background(), &fftypes.Namespace{
		Name:        "ns1",
		Description: buff.String(),
	}, false)
	assert.Regexp(t, "FF00135.*description", err)
}

func TestBroadcastNamespaceBroadcastOk(t *testing.T) {
	ds, cancel := newTestDefinitionSender(t)
	defer cancel()
	mim := ds.identity.(*identitymanagermocks.Manager)
	mbm := ds.broadcast.(*broadcastmocks.Manager)
	mms := &sysmessagingmocks.MessageSender{}

	mim.On("ResolveInputSigningIdentity", mock.Anything, fftypes.SystemNamespace, mock.Anything).Return(nil)
	mbm.On("NewBroadcast", fftypes.SystemNamespace, mock.Anything).Return(mms)
	mms.On("Send", context.Background()).Return(nil)

	buff := strings.Builder{}
	buff.Grow(4097)
	for i := 0; i < 4097; i++ {
		buff.WriteByte(byte('a' + i%26))
	}
	_, err := ds.BroadcastNamespace(context.Background(), &fftypes.Namespace{
		Name:        "ns1",
		Description: "my namespace",
	}, false)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}
