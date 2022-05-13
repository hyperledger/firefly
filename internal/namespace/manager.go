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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

type Manager interface {
	// Init initializes the manager
	Init(ctx context.Context) error
	// GetConfigWithFallback retrieve a config string from the namespace or the root config
	GetConfigWithFallback(ns string, key config.RootKey) string
}

type namespaceManager struct {
	ctx      context.Context
	database database.Plugin
	nsConfig map[string]config.Section
}

func NewNamespaceManager(ctx context.Context, di database.Plugin) (Manager, error) {
	if di == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError)
	}
	nm := &namespaceManager{
		ctx:      ctx,
		database: di,
		nsConfig: buildNamespaceMap(ctx),
	}
	return nm, nil
}

func buildNamespaceMap(ctx context.Context) map[string]config.Section {
	conf := namespacePredefined
	namespaces := make(map[string]config.Section, conf.ArraySize())
	for i := 0; i < conf.ArraySize(); i++ {
		nsConfig := conf.ArrayEntry(i)
		name := nsConfig.GetString(coreconfig.NamespaceName)
		if name != "" {
			if _, ok := namespaces[name]; ok {
				log.L(ctx).Warnf("Duplicate predefined namespace (ignored): %s", name)
			}
			namespaces[name] = nsConfig
		}
	}
	return namespaces
}

func (nm *namespaceManager) Init(ctx context.Context) error {
	return nm.initNamespaces(ctx)
}

func (nm *namespaceManager) getPredefinedNamespaces(ctx context.Context) ([]*core.Namespace, error) {
	defaultNS := config.GetString(coreconfig.NamespacesDefault)
	namespaces := []*core.Namespace{
		{
			Name:        core.LegacySystemNamespace,
			Type:        core.NamespaceTypeSystem,
			Description: i18n.Expand(ctx, coremsgs.CoreSystemNSDescription),
		},
	}
	i := 0
	foundDefault := false
	for name, nsObject := range nm.nsConfig {
		if err := core.ValidateFFNameField(ctx, name, fmt.Sprintf("namespaces.predefined[%d].name", i)); err != nil {
			return nil, err
		}
		i++
		foundDefault = foundDefault || name == defaultNS
		namespaces = append(namespaces, &core.Namespace{
			Type:        core.NamespaceTypeLocal,
			Name:        name,
			Description: nsObject.GetString("description"),
		})
	}
	if !foundDefault {
		return nil, i18n.NewError(ctx, coremsgs.MsgDefaultNamespaceNotFound, defaultNS)
	}
	return namespaces, nil
}

func (nm *namespaceManager) initNamespaces(ctx context.Context) error {
	predefined, err := nm.getPredefinedNamespaces(ctx)
	if err != nil {
		return err
	}
	for _, newNS := range predefined {
		ns, err := nm.database.GetNamespace(ctx, newNS.Name)
		if err != nil {
			return err
		}
		var updated bool
		if ns == nil {
			updated = true
			newNS.ID = fftypes.NewUUID()
			newNS.Created = fftypes.Now()
		} else {
			// Only update if the description has changed, and the one in our DB is locally defined
			updated = ns.Description != newNS.Description && ns.Type == core.NamespaceTypeLocal
		}
		if updated {
			if err := nm.database.UpsertNamespace(ctx, newNS, true); err != nil {
				return err
			}
		}
	}
	return nil
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
