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
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/publicstorage"
)

type Manager interface {
	BroadcastFFI(ctx context.Context, ns string, ffi *fftypes.FFI, waitConfirm bool) (output *fftypes.FFI, err error)
	GetFFI(ctx context.Context, ns, name, version string) (*fftypes.FFI, error)
	GetFFIByID(ctx context.Context, id string) (*fftypes.FFI, error)
	GetFFIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.FFI, *database.FilterResult, error)

	InvokeContract(ctx context.Context, ns string, req *fftypes.InvokeContractRequest) (interface{}, error)
	InvokeContractAPI(ctx context.Context, ns, apiName, methodName string, req *fftypes.InvokeContractRequest) (interface{}, error)
	GetContractAPIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractAPI, *database.FilterResult, error)
	BroadcastContractAPI(ctx context.Context, ns string, api *fftypes.ContractAPI, waitConfirm bool) (output *fftypes.ContractAPI, err error)

	ValidateFFI(ctx context.Context, ns string, ffi *fftypes.FFI) error
	ValidateInvokeContractRequest(ctx context.Context, req *fftypes.InvokeContractRequest) error
}

type contractManager struct {
	database      database.Plugin
	publicStorage publicstorage.Plugin
	broadcast     broadcast.Manager
	identity      identity.Manager
	blockchain    blockchain.Plugin
}

func NewContractManager(database database.Plugin, publicStorage publicstorage.Plugin, broadcast broadcast.Manager, identity identity.Manager, blockchain blockchain.Plugin) Manager {
	return &contractManager{
		database,
		publicStorage,
		broadcast,
		identity,
		blockchain,
	}
}

func (cm *contractManager) BroadcastFFI(ctx context.Context, ns string, ffi *fftypes.FFI, waitConfirm bool) (output *fftypes.FFI, err error) {
	ffi.ID = fftypes.NewUUID()
	ffi.Namespace = ns

	existing, err := cm.database.GetFFI(ctx, ffi.Namespace, ffi.Name, ffi.Version)

	if existing != nil && err == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractInterfaceExists, ffi.Namespace, ffi.Name, ffi.Version)
	}

	for _, method := range ffi.Methods {
		method.ID = fftypes.NewUUID()
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
	_, err = cm.broadcast.BroadcastDefinition(ctx, ns, ffi, identity, fftypes.SystemTagDefineFFI, waitConfirm)
	if err != nil {
		return nil, err
	}
	return ffi, nil
}

func (cm *contractManager) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (cm *contractManager) GetFFI(ctx context.Context, ns, name, version string) (*fftypes.FFI, error) {
	return cm.database.GetFFI(ctx, ns, name, version)
}

func (cm *contractManager) GetFFIByID(ctx context.Context, id string) (*fftypes.FFI, error) {
	return cm.database.GetFFIByID(ctx, id)
}

func (cm *contractManager) GetFFIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.FFI, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	return cm.database.GetFFIs(ctx, ns, filter)
}

func (cm *contractManager) InvokeContract(ctx context.Context, ns string, req *fftypes.InvokeContractRequest) (interface{}, error) {
	signingKey := cm.identity.GetOrgKey(ctx)
	operationID := fftypes.NewUUID()
	method, err := cm.resolveInvokeContractRequest(ctx, ns, req)
	if err != nil {
		return nil, err
	}
	if err := cm.ValidateInvokeContractRequest(ctx, req); err != nil {
		return nil, err
	}
	return cm.blockchain.InvokeContract(ctx, operationID, signingKey, req.Location, method, req.Params)
}

func (cm *contractManager) InvokeContractAPI(ctx context.Context, ns, apiName, methodName string, req *fftypes.InvokeContractRequest) (interface{}, error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	if err != nil {
		return nil, err
	}
	req.ContractID = api.Contract.ID
	req.Method = &fftypes.FFIMethod{
		Name: methodName,
	}
	if api.Location != nil {
		req.Location = api.Location
	}
	return cm.InvokeContract(ctx, ns, req)
}

func (cm *contractManager) resolveInvokeContractRequest(ctx context.Context, ns string, req *fftypes.InvokeContractRequest) (method *fftypes.FFIMethod, err error) {
	if req.Method == nil {
		// TODO: more helpful error message here
		return nil, fmt.Errorf("method nil")
	}
	method = req.Method
	// We have a method name but no method signature - look up the method in the DB
	if method.Name != "" && (method.Params == nil || method.Returns == nil) {
		if req.ContractID.String() == "" {
			return nil, fmt.Errorf("error resolving contract method - method signature is required if contract ID is absent")
		}

		method, err = cm.database.GetFFIMethod(ctx, ns, req.ContractID, method.Name)
		if err != nil {
			return nil, fmt.Errorf("error resolving contract method")
		}
	}
	return method, nil
}

func (cm *contractManager) GetContractAPIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractAPI, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	return cm.database.GetContractAPIs(ctx, ns, filter)
}

func (cm *contractManager) BroadcastContractAPI(ctx context.Context, ns string, api *fftypes.ContractAPI, waitConfirm bool) (output *fftypes.ContractAPI, err error) {
	api.ID = fftypes.NewUUID()
	api.Namespace = ns

	existing, err := cm.database.GetContractAPIByID(ctx, api.ID.String())

	if existing != nil && err == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractInterfaceExists, api.Namespace, api.Contract.Name, api.Contract.Version)
	}

	localOrgDID, err := cm.identity.ResolveLocalOrgDID(ctx)
	if err != nil {
		return nil, err
	}
	identity := &fftypes.Identity{
		Author: localOrgDID,
		Key:    cm.identity.GetOrgKey(ctx),
	}

	msg, err := cm.broadcast.BroadcastDefinition(ctx, ns, api, identity, fftypes.SystemTagDefineContractAPI, waitConfirm)
	if err != nil {
		return nil, err
	}
	api.Message = msg.Header.ID
	return api, nil
}

func (cm *contractManager) ValidateFFI(ctx context.Context, ns string, ffi *fftypes.FFI) error {
	for _, method := range ffi.Methods {
		if err := cm.validateFFIMethod(ctx, method); err != nil {
			return err
		}
	}
	for _, method := range ffi.Events {
		if err := cm.validateFFIEvent(ctx, method); err != nil {
			return err
		}
	}
	return nil
}

func (cm *contractManager) validateFFIMethod(ctx context.Context, method *fftypes.FFIMethod) error {
	return cm.blockchain.ValidateFFIMethod(ctx, method)
}

func (cm *contractManager) validateFFIEvent(ctx context.Context, event *fftypes.FFIEvent) error {
	return cm.blockchain.ValidateFFIEvent(ctx, event)
}

func (cm *contractManager) ValidateInvokeContractRequest(ctx context.Context, req *fftypes.InvokeContractRequest) error {
	if err := cm.validateFFIMethod(ctx, req.Method); err != nil {
		return err
	}

	for _, param := range req.Method.Params {
		value, ok := req.Params[param.Name]
		if !ok {
			return i18n.NewError(ctx, i18n.MsgContractMissingInputArgument, param.Name)
		}
		if err := checkParam(ctx, value, param); err != nil {
			return err
		}
	}

	return cm.blockchain.ValidateInvokeContractRequest(ctx, req)
}
