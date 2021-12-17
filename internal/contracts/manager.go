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

	"github.com/getkin/kin-openapi/openapi3"
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
	GetFFIByIDWithChildren(ctx context.Context, id string) (*fftypes.FFI, error)
	GetFFIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.FFI, *database.FilterResult, error)

	InvokeContract(ctx context.Context, ns string, req *fftypes.InvokeContractRequest) (interface{}, error)
	InvokeContractAPI(ctx context.Context, ns, apiName, methodName string, req *fftypes.InvokeContractRequest) (interface{}, error)
	GetContractAPIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractAPI, *database.FilterResult, error)
	BroadcastContractAPI(ctx context.Context, ns string, api *fftypes.ContractAPI, waitConfirm bool) (output *fftypes.ContractAPI, err error)

	ValidateFFI(ctx context.Context, ffi *fftypes.FFI) error
	ValidateInvokeContractRequest(ctx context.Context, req *fftypes.InvokeContractRequest) error

	AddContractSubscription(ctx context.Context, ns string, sub *fftypes.ContractSubscriptionInput) (output *fftypes.ContractSubscription, err error)
	GetContractSubscriptionByNameOrID(ctx context.Context, ns, nameOrID string) (*fftypes.ContractSubscription, error)
	GetContractSubscriptions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractSubscription, *database.FilterResult, error)
	DeleteContractSubscriptionByNameOrID(ctx context.Context, ns, nameOrID string) error
	GetContractEventByID(ctx context.Context, id *fftypes.UUID) (*fftypes.ContractEvent, error)
	GetContractEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractEvent, *database.FilterResult, error)
	GetContractAPISwagger(ctx context.Context, ns, apiName string) (*openapi3.T, error)
}

type contractManager struct {
	database      database.Plugin
	publicStorage publicstorage.Plugin
	broadcast     broadcast.Manager
	identity      identity.Manager
	blockchain    blockchain.Plugin
	swaggerGen    ContractAPISwaggerGen
}

func NewContractManager(ctx context.Context, database database.Plugin, publicStorage publicstorage.Plugin, broadcast broadcast.Manager, identity identity.Manager, blockchain blockchain.Plugin) (Manager, error) {
	if database == nil || publicStorage == nil || broadcast == nil || identity == nil || blockchain == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	return &contractManager{
		database:      database,
		publicStorage: publicStorage,
		broadcast:     broadcast,
		identity:      identity,
		blockchain:    blockchain,
		swaggerGen:    newContractAPISwaggerGen(),
	}, nil
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

	if err := cm.ValidateFFI(ctx, ffi); err != nil {
		return nil, err
	}

	localOrgDID, err := cm.identity.ResolveLocalOrgDID(ctx)
	if err != nil {
		return nil, err
	}
	identity := &fftypes.Identity{
		Author: localOrgDID,
		Key:    cm.identity.GetOrgKey(ctx),
	}

	output = ffi
	msg, err := cm.broadcast.BroadcastDefinition(ctx, ns, ffi, identity, fftypes.SystemTagDefineFFI, waitConfirm)
	if err != nil {
		return nil, err
	}
	output.Message = msg.Header.ID
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

func (cm *contractManager) GetFFIByIDWithChildren(ctx context.Context, id string) (*fftypes.FFI, error) {
	ffi, err := cm.database.GetFFIByID(ctx, id)
	if err != nil || ffi == nil {
		return nil, err
	}

	mfb := database.FFIMethodQueryFactory.NewFilter(ctx)
	ffi.Methods, _, err = cm.database.GetFFIMethods(ctx, mfb.Eq("interfaceid", id))
	if err != nil {
		return nil, err
	}

	efb := database.FFIEventQueryFactory.NewFilter(ctx)
	ffi.Events, _, err = cm.database.GetFFIEvents(ctx, efb.Eq("interfaceid", id))
	if err != nil {
		return nil, err
	}

	return ffi, nil
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
		return nil, i18n.NewError(ctx, i18n.MsgContractMethodNotSet)
	}
	method = req.Method
	// We have a method name but no method signature - look up the method in the DB
	if method.Name != "" && (method.Params == nil || method.Returns == nil) {
		if req.ContractID.String() == "" {
			return nil, i18n.NewError(ctx, i18n.MsgContractNoMethodSignature)
		}

		method, err = cm.database.GetFFIMethod(ctx, ns, req.ContractID, method.Name)
		if err != nil || method == nil {
			return nil, i18n.NewError(ctx, i18n.MsgContractResolveError)
		}
	}
	return method, nil
}

func (cm *contractManager) GetContractAPIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractAPI, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	return cm.database.GetContractAPIs(ctx, ns, filter)
}

func (cm *contractManager) GetContractAPISwagger(ctx context.Context, ns, apiName string) (*openapi3.T, error) {

	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	if err != nil {
		return nil, err
	}
	if api == nil || api.Contract == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NoResult)
	}

	ffi, err := cm.GetFFIByIDWithChildren(ctx, api.Contract.ID.String())
	if err != nil {
		return nil, err
	}

	return cm.swaggerGen.Generate(ctx, ffi)

}

func (cm *contractManager) BroadcastContractAPI(ctx context.Context, ns string, api *fftypes.ContractAPI, waitConfirm bool) (output *fftypes.ContractAPI, err error) {
	api.ID = fftypes.NewUUID()
	api.Namespace = ns

	existing, err := cm.database.GetContractAPIByName(ctx, api.Namespace, api.Name)

	if existing != nil && err == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractAPIExists, api.Namespace, api.Name)
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

func (cm *contractManager) ValidateFFI(ctx context.Context, ffi *fftypes.FFI) error {
	for _, method := range ffi.Methods {
		if err := cm.validateFFIMethod(ctx, method); err != nil {
			return err
		}
	}
	for _, event := range ffi.Events {
		if err := cm.validateFFIEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (cm *contractManager) validateFFIMethod(ctx context.Context, method *fftypes.FFIMethod) error {
	for _, param := range method.Params {
		if err := cm.blockchain.ValidateFFIParam(ctx, param); err != nil {
			return err
		}
	}
	for _, param := range method.Returns {
		if err := cm.blockchain.ValidateFFIParam(ctx, param); err != nil {
			return err
		}
	}
	return nil
}

func (cm *contractManager) validateFFIEvent(ctx context.Context, event *fftypes.FFIEvent) error {
	for _, param := range event.Params {
		if err := cm.blockchain.ValidateFFIParam(ctx, param); err != nil {
			return err
		}
	}
	return nil
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

	return nil
}

func (cm *contractManager) AddContractSubscription(ctx context.Context, ns string, sub *fftypes.ContractSubscriptionInput) (output *fftypes.ContractSubscription, err error) {
	sub.ID = fftypes.NewUUID()
	sub.Namespace = ns
	sub.Event.ID = fftypes.NewUUID()
	sub.ContractSubscription.Event = sub.Event.ID

	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}

	if sub.Name != "" {
		if err := fftypes.ValidateFFNameField(ctx, sub.Name, "name"); err != nil {
			return nil, err
		}
		if existing, err := cm.database.GetContractSubscription(ctx, ns, sub.Name); err != nil {
			return nil, err
		} else if existing != nil {
			return nil, i18n.NewError(ctx, i18n.MsgContractSubscriptionExists, ns, sub.Name)
		}
	}

	if err := cm.validateFFIEvent(ctx, &sub.Event); err != nil {
		return nil, err
	}
	if err = cm.blockchain.AddSubscription(ctx, sub); err != nil {
		return nil, err
	}

	err = cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		if err = cm.database.UpsertFFIEvent(ctx, ns, nil, &sub.Event); err != nil {
			return err
		}
		if err = cm.database.UpsertContractSubscription(ctx, &sub.ContractSubscription); err != nil {
			return err
		}
		return nil
	})
	return &sub.ContractSubscription, err
}

func (cm *contractManager) GetContractSubscriptionByNameOrID(ctx context.Context, ns, nameOrID string) (sub *fftypes.ContractSubscription, err error) {
	id, err := fftypes.ParseUUID(ctx, nameOrID)
	if err != nil {
		if err := fftypes.ValidateFFNameField(ctx, nameOrID, "name"); err != nil {
			return nil, err
		}
		if sub, err = cm.database.GetContractSubscription(ctx, ns, nameOrID); err != nil {
			return nil, err
		}
	} else if sub, err = cm.database.GetContractSubscriptionByID(ctx, id); err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	return sub, nil
}

func (cm *contractManager) GetContractSubscriptions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractSubscription, *database.FilterResult, error) {
	return cm.database.GetContractSubscriptions(ctx, cm.scopeNS(ns, filter))
}

func (cm *contractManager) DeleteContractSubscriptionByNameOrID(ctx context.Context, ns, nameOrID string) error {
	sub, err := cm.GetContractSubscriptionByNameOrID(ctx, ns, nameOrID)
	if err != nil {
		return err
	}
	if err = cm.blockchain.DeleteSubscription(ctx, sub); err != nil {
		return err
	}
	return cm.database.DeleteContractSubscriptionByID(ctx, sub.ID)
}

func (cm *contractManager) GetContractEventByID(ctx context.Context, id *fftypes.UUID) (*fftypes.ContractEvent, error) {
	return cm.database.GetContractEventByID(ctx, id)
}

func (cm *contractManager) GetContractEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractEvent, *database.FilterResult, error) {
	return cm.database.GetContractEvents(ctx, cm.scopeNS(ns, filter))
}
