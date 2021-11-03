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

package contracts

import (
	"context"

	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/publicstorage"
)

type ContractManager struct {
	database      database.Plugin
	publicStorage publicstorage.Plugin
	broadcast     broadcast.Manager
	identity      identity.Manager
}

func NewContractManager(database database.Plugin, publicStorage publicstorage.Plugin, broadcast broadcast.Manager, identity identity.Manager) *ContractManager {
	return &ContractManager{
		database,
		publicStorage,
		broadcast,
		identity,
	}
}

func (cm *ContractManager) AddContractDefinition(ctx context.Context, ns string, cd *fftypes.ContractDefinition, waitConfirm bool) (output *fftypes.ContractDefinition, err error) {
	cd.ID = fftypes.NewUUID()
	cd.Namespace = ns

	err = cd.Validate(ctx, false)
	if err != nil {
		return nil, err
	}

	existing, err := cm.database.GetContractDefinitionByNameAndVersion(ctx, cd.Namespace, cd.Name, cd.Version)

	if existing != nil && err == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractDefinitionExists, cd.Namespace, cd.Name, cd.Version)
	}

	localOrgDID, err := cm.identity.ResolveLocalOrgDID(ctx)
	if err != nil {
		return nil, err
	}
	identity := &fftypes.Identity{
		Author: localOrgDID,
		Key:    cm.identity.GetOrgKey(ctx),
	}

	// TODO: Do we do anything with this message here?
	_, err = cm.broadcast.BroadcastDefinition(ctx, ns, cd, identity, fftypes.SystemTagDefineContract, waitConfirm)
	if err != nil {
		return nil, err
	}
	return cd, nil
}

func (cm *ContractManager) GetContractDefinitionByNameAndVersion(ctx context.Context, ns, name, version string) (*fftypes.ContractDefinition, error) {
	return cm.database.GetContractDefinitionByNameAndVersion(ctx, ns, name, version)
}

func (cm *ContractManager) GetContractDefinitionByID(ctx context.Context, id string) (*fftypes.ContractDefinition, error) {
	return cm.database.GetContractDefinitionByID(ctx, id)
}
