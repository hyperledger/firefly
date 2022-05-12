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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly/internal/coreconfig"
)

type Manager interface {
	// GetConfigWithFallback retrieve a config string from the namespace or the root config
	GetConfigWithFallback(ns string, key config.RootKey) string
}

type namespaceManager struct {
	ctx      context.Context
	nsConfig map[string]config.Section
}

func NewNamespaceManager(ctx context.Context, predefinedNS config.ArraySection) Manager {
	ns := &namespaceManager{
		ctx: ctx,
	}
	if predefinedNS != nil {
		ns.nsConfig = buildNamespaceMap(predefinedNS)
	}
	return ns
}

func buildNamespaceMap(conf config.ArraySection) map[string]config.Section {
	result := make(map[string]config.Section, conf.ArraySize())
	for i := 0; i < conf.ArraySize(); i++ {
		nsConfig := conf.ArrayEntry(i)
		name := nsConfig.GetString(coreconfig.NamespaceName)
		if name != "" {
			result[name] = nsConfig
		}
	}
	return result
}

func (nm *namespaceManager) GetConfigWithFallback(ns string, key config.RootKey) string {
	if nsConfig, ok := nm.nsConfig[ns]; ok {
		val := nsConfig.GetString(string(key))
		if val != "" {
			return val
		}
	}
	return config.GetString(key)
}
