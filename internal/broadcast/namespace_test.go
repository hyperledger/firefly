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
	"strings"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastNamespaceBadName(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	_, err := bm.BroadcastNamespace(context.Background(), &fftypes.Namespace{
		Name: "!ns",
	}, false)
	assert.Regexp(t, "FF10131.*name", err)
}

func TestBroadcastNamespaceDescriptionTooLong(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)

	mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	buff := strings.Builder{}
	buff.Grow(4097)
	for i := 0; i < 4097; i++ {
		buff.WriteByte(byte('a' + i%26))
	}
	_, err := bm.BroadcastNamespace(context.Background(), &fftypes.Namespace{
		Name:        "ns1",
		Description: buff.String(),
	}, false)
	assert.Regexp(t, "FF10188.*description", err)
}

func TestBroadcastNamespaceBroadcastOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	mim.On("ResolveInputIdentity", mock.Anything, mock.Anything).Return(nil)
	mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{Name: "ns1"}, nil)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	mdi.On("UpsertMessage", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	buff := strings.Builder{}
	buff.Grow(4097)
	for i := 0; i < 4097; i++ {
		buff.WriteByte(byte('a' + i%26))
	}
	_, err := bm.BroadcastNamespace(context.Background(), &fftypes.Namespace{
		Name:        "ns1",
		Description: "my namespace",
	}, false)
	assert.NoError(t, err)
}
