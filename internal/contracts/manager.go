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
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/publicstorage"
)

type ContractManager struct {
	database      database.Plugin
	publicStorage publicstorage.Plugin
	broadcast     broadcast.Manager
	identity      identity.Manager
	blockchain    blockchain.Plugin
}

func NewContractManager(database database.Plugin, publicStorage publicstorage.Plugin, broadcast broadcast.Manager, identity identity.Manager, blockchain blockchain.Plugin) *ContractManager {
	return &ContractManager{
		database,
		publicStorage,
		broadcast,
		identity,
		blockchain,
	}
}

func (cm *ContractManager) BroadcastContractInterface(ctx context.Context, ns string, ffi *fftypes.FFI, waitConfirm bool) (output *fftypes.FFI, err error) {
	ffi.ID = fftypes.NewUUID()
	ffi.Namespace = ns

	existing, err := cm.database.GetContractInterfaceByNameAndVersion(ctx, ffi.Namespace, ffi.Name, ffi.Version)

	if existing != nil && err == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractInterfaceExists, ffi.Namespace, ffi.Name, ffi.Version)
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
	_, err = cm.broadcast.BroadcastDefinition(ctx, ns, ffi, identity, fftypes.SystemTagDefineContractInterface, waitConfirm)
	if err != nil {
		return nil, err
	}
	return ffi, nil
}

// func (cm *ContractManager) AddContractInstance(ctx context.Context, ns string, ci *fftypes.ContractInstance, waitConfirm bool) (output *fftypes.ContractInstance, err error) {
// 	ci.ID = fftypes.NewUUID()
// 	ci.Namespace = ns

// 	err = ci.Validate(ctx, false)
// 	if err != nil {
// 		return nil, err
// 	}

// 	existing, err := cm.database.GetContractInstanceByName(ctx, ci.Namespace, ci.Name)
// 	if existing != nil && err == nil {
// 		return nil, i18n.NewError(ctx, i18n.MsgContractInstanceExists, ci.Namespace, ci.Name)
// 	}

// 	if ci.ContractInterface.ID != nil {
// 		ContractInterface, err := cm.database.GetContractInterfaceByID(ctx, ci.ContractInterface.ID.String())
// 		if err != nil {
// 			return nil, i18n.NewError(ctx, i18n.MsgContractInterfaceNotFound, fmt.Sprintf("id: '%s'", ci.ContractInterface.ID))
// 		}
// 		ci.ContractInterface = ContractInterface
// 	} else {
// 		ContractInterface, err := cm.database.GetContractInterfaceByNameAndVersion(ctx, ns, ci.ContractInterface.Name, ci.ContractInterface.Version)
// 		if err != nil {
// 			return nil, i18n.NewError(ctx, i18n.MsgContractInterfaceNotFound, fmt.Sprintf("namespace: '%s' name: '%s' version: '%s'", ns, ci.ContractInterface.Name, ci.ContractInterface.Version))
// 		}
// 		ci.ContractInterface = ContractInterface
// 	}

// 	localOrgDID, err := cm.identity.ResolveLocalOrgDID(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	identity := &fftypes.Identity{
// 		Author: localOrgDID,
// 		Key:    cm.identity.GetOrgKey(ctx),
// 	}

// 	// TODO: Do we do anything with this message here?
// 	_, err = cm.broadcast.BroadcastDefinition(ctx, ns, ci, identity, fftypes.SystemTagDefineContractInstance, waitConfirm)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return ci, nil
// }

func (cm *ContractManager) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (cm *ContractManager) GetContractInterfaceByNameAndVersion(ctx context.Context, ns, name, version string) (*fftypes.FFI, error) {
	return cm.database.GetContractInterfaceByNameAndVersion(ctx, ns, name, version)
}

func (cm *ContractManager) GetContractInterfaceByID(ctx context.Context, id string) (*fftypes.FFI, error) {
	return cm.database.GetContractInterfaceByID(ctx, id)
}

func (cm *ContractManager) GetContractInterfaces(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.FFI, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	return cm.database.GetContractInterfaces(ctx, ns, filter)
}

func (cm *ContractManager) GetContractMethod(ctx context.Context, ns, contractID, methodName string) (*fftypes.FFIMethod, error) {
	return nil, nil
}

// func (cm *ContractManager) GetContractInstanceByNameOrID(ctx context.Context, ns, nameOrID string) (*fftypes.ContractInstance, error) {
// 	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
// 		return nil, err
// 	}

// 	var ci *fftypes.ContractInstance

// 	instanceID, err := fftypes.ParseUUID(ctx, nameOrID)
// 	if err != nil {
// 		if err := fftypes.ValidateFFNameField(ctx, nameOrID, "name"); err != nil {
// 			return nil, err
// 		}
// 		if ci, err = cm.database.GetContractInstanceByName(ctx, ns, nameOrID); err != nil {
// 			return nil, err
// 		}
// 	} else if ci, err = cm.database.GetContractInstanceByID(ctx, instanceID.String()); err != nil {
// 		return nil, err
// 	}
// 	if ci == nil {
// 		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
// 	}
// 	if err = cm.joinContractInterface(ctx, ci); err != nil {
// 		return nil, err
// 	}
// 	return ci, nil
// }

// func (cm *ContractManager) GetContractInstances(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractInstance, *database.FilterResult, error) {
// 	filter = cm.scopeNS(ns, filter)
// 	instances, res, err := cm.database.GetContractInstances(ctx, ns, filter)
// 	if err != nil {
// 		return instances, res, err
// 	}
// 	for _, ci := range instances {
// 		if err = cm.joinContractInterface(ctx, ci); err != nil {
// 			return instances, res, err
// 		}
// 	}
// 	return instances, res, nil
// }

// func (cm *ContractManager) joinContractInterface(ctx context.Context, ci *fftypes.ContractInstance) error {
// 	cd, err := cm.database.GetContractInterfaceByID(ctx, ci.ContractInterface.ID.String())
// 	if err != nil {
// 		return err
// 	}
// 	ci.ContractInterface = cd
// 	return nil
// }

func (cm *ContractManager) InvokeContract(ctx context.Context, ns, contractInstanceNameOrID, methodName string, req *fftypes.ContractInvocationRequest) (interface{}, error) {
	// if contractInstanceNameOrID != "" {
	// 	ci, err := cm.GetContractInstanceByNameOrID(ctx, ns, contractInstanceNameOrID)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	req.OnChainLocation = ci.OnChainLocation
	// }
	if methodName != "" {
		req.Method = methodName
	}

	signingKey := cm.identity.GetOrgKey(ctx)
	operationID := fftypes.NewUUID()

	method, err := cm.database.GetContractMethodByName(ctx, ns, contractInstanceNameOrID, methodName)
	if err != nil {
		return nil, err
	}

	return cm.blockchain.InvokeContract(ctx, operationID, signingKey, req.OnChainLocation, method, req.Params)
}

func (cm *ContractManager) GetMethod(ctx context.Context, ns, contractInstanceNameOrID, methodName string) (*fftypes.FFIMethod, error) {
	return cm.database.GetContractMethodByName(ctx, ns, contractInstanceNameOrID, methodName)
}
