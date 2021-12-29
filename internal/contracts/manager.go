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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/publicstorage"
)

type Manager interface {
	BroadcastFFI(ctx context.Context, ns string, ffi *fftypes.FFI, waitConfirm bool) (output *fftypes.FFI, err error)
	GetFFI(ctx context.Context, ns, name, version string) (*fftypes.FFI, error)
	GetFFIByID(ctx context.Context, id *fftypes.UUID) (*fftypes.FFI, error)
	GetFFIByIDWithChildren(ctx context.Context, id *fftypes.UUID) (*fftypes.FFI, error)
	GetFFIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.FFI, *database.FilterResult, error)

	InvokeContract(ctx context.Context, ns string, req *fftypes.InvokeContractRequest) (interface{}, error)
	InvokeContractAPI(ctx context.Context, ns, apiName, methodPath string, req *fftypes.InvokeContractRequest) (interface{}, error)
	GetContractAPIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractAPI, *database.FilterResult, error)
	BroadcastContractAPI(ctx context.Context, ns string, api *fftypes.ContractAPI, waitConfirm bool) (output *fftypes.ContractAPI, err error)
	SubscribeContract(ctx context.Context, ns, eventPath string, req *fftypes.ContractSubscribeRequest) (*fftypes.ContractSubscription, error)
	SubscribeContractAPI(ctx context.Context, ns, apiName, eventPath string, req *fftypes.ContractSubscribeRequest) (*fftypes.ContractSubscription, error)

	ValidateFFIAndSetPathnames(ctx context.Context, ffi *fftypes.FFI) error

	AddContractSubscription(ctx context.Context, ns string, sub *fftypes.ContractSubscriptionInput) (output *fftypes.ContractSubscription, err error)
	GetContractSubscriptionByNameOrID(ctx context.Context, ns, nameOrID string) (*fftypes.ContractSubscription, error)
	GetContractSubscriptions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractSubscription, *database.FilterResult, error)
	DeleteContractSubscriptionByNameOrID(ctx context.Context, ns, nameOrID string) error
	GetContractEventByID(ctx context.Context, id *fftypes.UUID) (*fftypes.ContractEvent, error)
	GetContractEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractEvent, *database.FilterResult, error)

	GetContractAPISwagger(ctx context.Context, httpServerURL, ns, apiName string) (*openapi3.T, error)
}

type contractManager struct {
	database      database.Plugin
	publicStorage publicstorage.Plugin
	broadcast     broadcast.Manager
	identity      identity.Manager
	blockchain    blockchain.Plugin
	swaggerGen    oapispec.FFISwaggerGen
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
		swaggerGen:    oapispec.NewFFISwaggerGen(),
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
	for _, event := range ffi.Events {
		event.ID = fftypes.NewUUID()
	}
	if err := cm.ValidateFFIAndSetPathnames(ctx, ffi); err != nil {
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

func (cm *contractManager) GetFFIByID(ctx context.Context, id *fftypes.UUID) (*fftypes.FFI, error) {
	return cm.database.GetFFIByID(ctx, id)
}

func (cm *contractManager) GetFFIByIDWithChildren(ctx context.Context, id *fftypes.UUID) (*fftypes.FFI, error) {
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

func (cm *contractManager) InvokeContract(ctx context.Context, ns string, req *fftypes.InvokeContractRequest) (res interface{}, err error) {
	signingKey := cm.identity.GetOrgKey(ctx)
	operationID := fftypes.NewUUID()
	if req.Method, err = cm.resolveInvokeContractRequest(ctx, ns, req); err != nil {
		return nil, err
	}
	if err := cm.validateInvokeContractRequest(ctx, req); err != nil {
		return nil, err
	}
	return cm.blockchain.InvokeContract(ctx, operationID, signingKey, req.Location, req.Method, req.Params)
}

func (cm *contractManager) InvokeContractAPI(ctx context.Context, ns, apiName, methodPath string, req *fftypes.InvokeContractRequest) (interface{}, error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	if err != nil {
		return nil, err
	} else if api == nil || api.Interface == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	req.ContractID = api.Interface.ID
	req.Method = &fftypes.FFIMethod{
		Pathname: methodPath,
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
	if method.Pathname == "" {
		method.Pathname = method.Name
	}
	if method.Pathname != "" && (method.Params == nil || method.Returns == nil) {
		if req.ContractID == nil {
			return nil, i18n.NewError(ctx, i18n.MsgContractNoMethodSignature)
		}

		method, err = cm.database.GetFFIMethod(ctx, ns, req.ContractID, method.Pathname)
		if err != nil || method == nil {
			return nil, i18n.NewError(ctx, i18n.MsgContractMethodResolveError)
		}
	}
	return method, nil
}

func (cm *contractManager) GetContractAPIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractAPI, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	return cm.database.GetContractAPIs(ctx, ns, filter)
}

func (cm *contractManager) GetContractAPISwagger(ctx context.Context, httpServerURL, ns, apiName string) (*openapi3.T, error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	if err != nil {
		return nil, err
	} else if api == nil || api.Interface == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NoResult)
	}

	ffi, err := cm.GetFFIByIDWithChildren(ctx, api.Interface.ID)
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("%s/namespaces/%s/apis/%s", httpServerURL, ns, apiName)
	return cm.swaggerGen.Generate(ctx, baseURL, ffi)
}

func (cm *contractManager) resolveFFIReference(ctx context.Context, ns string, ref *fftypes.FFIReference) error {
	switch {
	case ref == nil:
		return i18n.NewError(ctx, i18n.MsgContractInterfaceNotFound, "")

	case ref.ID != nil:
		ffi, err := cm.database.GetFFIByID(ctx, ref.ID)
		if err != nil {
			return err
		} else if ffi == nil {
			return i18n.NewError(ctx, i18n.MsgContractInterfaceNotFound, ref.ID)
		}
		return nil

	case ref.Name != "" && ref.Version != "":
		ffi, err := cm.database.GetFFI(ctx, ns, ref.Name, ref.Version)
		if err != nil {
			return err
		} else if ffi == nil {
			return i18n.NewError(ctx, i18n.MsgContractInterfaceNotFound, ref.Name)
		}
		ref.ID = ffi.ID
		return nil

	default:
		return i18n.NewError(ctx, i18n.MsgContractInterfaceNotFound, ref.Name)
	}
}

func (cm *contractManager) BroadcastContractAPI(ctx context.Context, ns string, api *fftypes.ContractAPI, waitConfirm bool) (output *fftypes.ContractAPI, err error) {
	api.ID = fftypes.NewUUID()
	api.Namespace = ns

	existing, err := cm.database.GetContractAPIByName(ctx, api.Namespace, api.Name)
	if existing != nil && err == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractAPIExists, api.Namespace, api.Name)
	}

	if err := cm.resolveFFIReference(ctx, ns, api.Interface); err != nil {
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

	msg, err := cm.broadcast.BroadcastDefinition(ctx, ns, api, identity, fftypes.SystemTagDefineContractAPI, waitConfirm)
	if err != nil {
		return nil, err
	}
	api.Message = msg.Header.ID
	return api, nil
}

func (cm *contractManager) uniquePathName(name string, usedNames map[string]bool) string {
	pathName := name
	for counter := 1; ; counter++ {
		if _, existing := usedNames[pathName]; existing {
			pathName = fmt.Sprintf("%s_%d", name, counter)
		} else {
			usedNames[pathName] = true
			return pathName
		}
	}
}

func (cm *contractManager) ValidateFFIAndSetPathnames(ctx context.Context, ffi *fftypes.FFI) error {

	methodPathNames := map[string]bool{}
	for _, method := range ffi.Methods {
		method.Contract = ffi.ID
		method.Namespace = ffi.Namespace
		method.Pathname = cm.uniquePathName(method.Name, methodPathNames)
		if err := cm.validateFFIMethod(ctx, method); err != nil {
			return err
		}
	}

	eventPathNames := map[string]bool{}
	for _, event := range ffi.Events {
		event.Contract = ffi.ID
		event.Namespace = ffi.Namespace
		event.Pathname = cm.uniquePathName(event.Name, eventPathNames)
		if err := cm.validateFFIEvent(ctx, &event.FFIEventDefinition); err != nil {
			return err
		}
	}
	return nil
}

func (cm *contractManager) validateFFIMethod(ctx context.Context, method *fftypes.FFIMethod) error {
	if method.Name == "" {
		return i18n.NewError(ctx, i18n.MsgMethodNameMustBeSet)
	}
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

func (cm *contractManager) validateFFIEvent(ctx context.Context, event *fftypes.FFIEventDefinition) error {
	if event.Name == "" {
		return i18n.NewError(ctx, i18n.MsgEventNameMustBeSet)
	}
	for _, param := range event.Params {
		if err := cm.blockchain.ValidateFFIParam(ctx, param); err != nil {
			return err
		}
	}
	return nil
}

func (cm *contractManager) validateInvokeContractRequest(ctx context.Context, req *fftypes.InvokeContractRequest) error {
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

	if sub.Interface != nil {
		if err := cm.resolveFFIReference(ctx, ns, sub.Interface); err != nil {
			return nil, err
		}
	}

	if sub.Event == nil {
		if sub.EventID == nil {
			return nil, i18n.NewError(ctx, i18n.MsgSubscriptionNoEvent)
		}

		event, err := cm.database.GetFFIEventByID(ctx, sub.EventID)
		if err != nil {
			return nil, err
		}
		if event == nil || event.Namespace != sub.Namespace {
			return nil, i18n.NewError(ctx, i18n.MsgSubscriptionEventNotFound, sub.Namespace, sub.EventID)
		}
		// Copy the event definition into the subscription
		sub.Event = &fftypes.FFISerializedEvent{
			FFIEventDefinition: event.FFIEventDefinition,
		}
	}

	if err := cm.validateFFIEvent(ctx, &sub.Event.FFIEventDefinition); err != nil {
		return nil, err
	}
	if err = cm.blockchain.AddSubscription(ctx, sub); err != nil {
		return nil, err
	}
	if sub.Name == "" {
		sub.Name = sub.ProtocolID
	}

	if sub.Name == "" {
		sub.Name = sub.ProtocolID
	}

	if err = cm.database.UpsertContractSubscription(ctx, &sub.ContractSubscription); err != nil {
		return nil, err
	}

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

func (cm *contractManager) SubscribeContract(ctx context.Context, ns, eventPath string, req *fftypes.ContractSubscribeRequest) (*fftypes.ContractSubscription, error) {
	event, err := cm.database.GetFFIEvent(ctx, ns, req.ContractID, eventPath)
	if err != nil || event == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractEventResolveError)
	}

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Interface: &fftypes.FFIReference{
				ID: req.ContractID,
			},
			Location: req.Location,
			Event: &fftypes.FFISerializedEvent{
				FFIEventDefinition: event.FFIEventDefinition,
			},
		},
	}
	return cm.AddContractSubscription(ctx, ns, sub)
}

func (cm *contractManager) SubscribeContractAPI(ctx context.Context, ns, apiName, eventPath string, req *fftypes.ContractSubscribeRequest) (*fftypes.ContractSubscription, error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	if err != nil {
		return nil, err
	} else if api == nil || api.Interface == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}

	req.ContractID = api.Interface.ID
	if api.Location != nil {
		req.Location = api.Location
	}

	return cm.SubscribeContract(ctx, ns, eventPath, req)
}

func (cm *contractManager) GetContractEventByID(ctx context.Context, id *fftypes.UUID) (*fftypes.ContractEvent, error) {
	return cm.database.GetContractEventByID(ctx, id)
}

func (cm *contractManager) GetContractEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractEvent, *database.FilterResult, error) {
	return cm.database.GetContractEvents(ctx, cm.scopeNS(ns, filter))
}
