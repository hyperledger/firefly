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
	"fmt"

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

func (cm *ContractManager) AddContractInstance(ctx context.Context, ns string, ci *fftypes.ContractInstance, waitConfirm bool) (output *fftypes.ContractInstance, err error) {
	ci.ID = fftypes.NewUUID()
	ci.Namespace = ns

	err = ci.Validate(ctx, false)
	if err != nil {
		return nil, err
	}

	existing, err := cm.database.GetContractInstanceByName(ctx, ci.Namespace, ci.Name)
	if existing != nil && err == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractInstanceExists, ci.Namespace, ci.Name)
	}

	if ci.ContractDefinition.ID != nil {
		contractDefinition, err := cm.database.GetContractDefinitionByID(ctx, ci.ContractDefinition.ID.String())
		if err != nil {
			return nil, i18n.NewError(ctx, i18n.MsgContractDefinitionNotFound, fmt.Sprintf("id: '%s'", ci.ContractDefinition.ID))
		}
		ci.ContractDefinition = contractDefinition
	} else {
		contractDefinition, err := cm.database.GetContractDefinitionByNameAndVersion(ctx, ns, ci.ContractDefinition.Name, ci.ContractDefinition.Version)
		if err != nil {
			return nil, i18n.NewError(ctx, i18n.MsgContractDefinitionNotFound, fmt.Sprintf("namespace: '%s' name: '%s' version: '%s'", ns, ci.ContractDefinition.Name, ci.ContractDefinition.Version))
		}
		ci.ContractDefinition = contractDefinition
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
	_, err = cm.broadcast.BroadcastDefinition(ctx, ns, ci, identity, fftypes.SystemTagDefineContractInstance, waitConfirm)
	if err != nil {
		return nil, err
	}
	return ci, nil
}

func (cm *ContractManager) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (cm *ContractManager) GetContractDefinitionByNameAndVersion(ctx context.Context, ns, name, version string) (*fftypes.ContractDefinition, error) {
	return cm.database.GetContractDefinitionByNameAndVersion(ctx, ns, name, version)
}

func (cm *ContractManager) GetContractDefinitionByID(ctx context.Context, id string) (*fftypes.ContractDefinition, error) {
	return cm.database.GetContractDefinitionByID(ctx, id)
}

func (cm *ContractManager) GetContractDefinitions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractDefinition, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	return cm.database.GetContractDefinitions(ctx, ns, filter)
}

func (cm *ContractManager) GetContractInstanceByNameOrID(ctx context.Context, ns, nameOrID string) (*fftypes.ContractInstance, error) {
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}

	var ci *fftypes.ContractInstance

	instanceID, err := fftypes.ParseUUID(ctx, nameOrID)
	if err != nil {
		if err := fftypes.ValidateFFNameField(ctx, nameOrID, "name"); err != nil {
			return nil, err
		}
		if ci, err = cm.database.GetContractInstanceByName(ctx, ns, nameOrID); err != nil {
			return nil, err
		}
	} else if ci, err = cm.database.GetContractInstanceByID(ctx, instanceID.String()); err != nil {
		return nil, err
	}
	if ci == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	if err = cm.joinContractDefinition(ctx, ci); err != nil {
		return nil, err
	}
	return ci, nil
}

func (cm *ContractManager) GetContractInstances(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractInstance, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	instances, res, err := cm.database.GetContractInstances(ctx, ns, filter)
	if err != nil {
		return instances, res, err
	}
	for _, ci := range instances {
		if err = cm.joinContractDefinition(ctx, ci); err != nil {
			return instances, res, err
		}
	}
	return instances, res, nil
}

func (cm *ContractManager) joinContractDefinition(ctx context.Context, ci *fftypes.ContractInstance) error {
	cd, err := cm.database.GetContractDefinitionByID(ctx, ci.ContractDefinition.ID.String())
	if err != nil {
		return err
	}
	ci.ContractDefinition = cd
	return nil
}
