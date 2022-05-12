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
	"testing"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/stretchr/testify/assert"
)

func newTestNamespaceManager(t *testing.T, predefinedNS config.ArraySection) *namespaceManager {
	nm := NewNamespaceManager(context.Background(), predefinedNS)
	return nm.(*namespaceManager)
}

func TestGetConfig(t *testing.T) {
	coreconfig.Reset()
	namespaceConfig := config.RootSection("namespaces")
	predefinedNS := namespaceConfig.SubArray("predefined")
	namespaceConfig.AddKnownKey("predefined.0."+coreconfig.NamespaceName, "ns1")
	namespaceConfig.AddKnownKey("predefined.0."+coreconfig.NamespaceOrgKey, "0x12345")

	nm := newTestNamespaceManager(t, predefinedNS)

	key := nm.GetConfigWithFallback("ns1", coreconfig.OrgKey)
	assert.Equal(t, "0x12345", key)
}

func TestGetConfigFallback(t *testing.T) {
	coreconfig.Reset()
	namespaceConfig := config.RootSection("namespaces")
	predefinedNS := namespaceConfig.SubArray("predefined")
	namespaceConfig.AddKnownKey("predefined.0."+coreconfig.NamespaceName, "ns1")
	config.Set(coreconfig.OrgKey, "0x123")

	nm := newTestNamespaceManager(t, predefinedNS)

	key := nm.GetConfigWithFallback("ns1", coreconfig.OrgKey)
	assert.Equal(t, "0x123", key)
}
