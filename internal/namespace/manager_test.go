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

package namespace

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testNamespaceManager struct {
	namespaceManager
	mdi *databasemocks.Plugin
}

func (nm *testNamespaceManager) cleanup(t *testing.T) {
	nm.mdi.AssertExpectations(t)
}

func newTestNamespaceManager(resetConfig bool) *testNamespaceManager {
	if resetConfig {
		coreconfig.Reset()
		InitConfig(true)
	}
	nm := &testNamespaceManager{
		mdi: &databasemocks.Plugin{},
		namespaceManager: namespaceManager{
			nsConfig: buildNamespaceMap(context.Background()),
		},
	}
	return nm
}

func TestNewNamespaceManager(t *testing.T) {
	nm := NewNamespaceManager(context.Background())
	assert.NotNil(t, nm)
}

func TestInit(t *testing.T) {
	coreconfig.Reset()
	nm := newTestNamespaceManager(false)
	defer nm.cleanup(t)

	nm.Init(context.Background(), nm.mdi)
}

func TestInitNamespacesBadName(t *testing.T) {
	coreconfig.Reset()
	namespaceConfig.AddKnownKey("predefined.0."+coreconfig.NamespaceName, "!Badness")

	nm := newTestNamespaceManager(false)
	defer nm.cleanup(t)

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF00140", err)
}

func TestInitNamespacesGetFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "pop", err)
}

func TestInitNamespacesUpsertFail(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	nm.mdi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))
	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "pop", err)
}

func TestInitNamespacesUpsertNotNeeded(t *testing.T) {
	nm := newTestNamespaceManager(true)
	defer nm.cleanup(t)

	nm.mdi.On("GetNamespace", mock.Anything, mock.Anything).Return(&core.Namespace{
		Type: core.NamespaceTypeBroadcast, // any broadcasted NS will not be updated
	}, nil)
	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.NoError(t, err)
}

func TestInitNamespacesDefaultMissing(t *testing.T) {
	coreconfig.Reset()
	config.Set(coreconfig.NamespacesPredefined, fftypes.JSONObjectArray{})

	nm := newTestNamespaceManager(false)
	defer nm.cleanup(t)

	err := nm.initNamespaces(context.Background(), nm.mdi)
	assert.Regexp(t, "FF10166", err)
}

func TestInitNamespacesDupName(t *testing.T) {
	coreconfig.Reset()
	InitConfig(true)

	namespaceConfig.AddKnownKey("predefined.0.name", "ns1")
	namespaceConfig.AddKnownKey("predefined.1.name", "ns2")
	namespaceConfig.AddKnownKey("predefined.2.name", "ns2")
	config.Set(coreconfig.NamespacesDefault, "ns1")

	nm := newTestNamespaceManager(false)
	defer nm.cleanup(t)

	nsList, err := nm.getPredefinedNamespaces(context.Background())
	assert.NoError(t, err)
	assert.Len(t, nsList, 3)
	names := make([]string, len(nsList))
	for i, ns := range nsList {
		names[i] = ns.Name
	}
	assert.Contains(t, names, core.SystemNamespace)
	assert.Contains(t, names, "ns1")
	assert.Contains(t, names, "ns2")
}
