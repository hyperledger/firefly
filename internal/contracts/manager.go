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

package contracts

import (
	"context"
	"fmt"
	"strings"

	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/publicstorage"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Manager interface {
	BroadcastFFI(ctx context.Context, ns string, ffi *fftypes.FFI, waitConfirm bool) (output *fftypes.FFI, err error)
	GetFFI(ctx context.Context, ns, name, version string) (*fftypes.FFI, error)
	GetFFIByID(ctx context.Context, id *fftypes.UUID) (*fftypes.FFI, error)
	GetFFIByIDWithChildren(ctx context.Context, id *fftypes.UUID) (*fftypes.FFI, error)
	GetFFIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.FFI, *database.FilterResult, error)

	InvokeContract(ctx context.Context, ns string, req *fftypes.ContractCallRequest) (interface{}, error)
	InvokeContractAPI(ctx context.Context, ns, apiName, methodPath string, req *fftypes.ContractCallRequest) (interface{}, error)
	GetContractAPI(ctx context.Context, httpServerURL, ns, apiName string) (*fftypes.ContractAPI, error)
	GetContractAPIs(ctx context.Context, httpServerURL, ns string, filter database.AndFilter) ([]*fftypes.ContractAPI, *database.FilterResult, error)
	BroadcastContractAPI(ctx context.Context, ns string, api *fftypes.ContractAPI, waitConfirm bool) (output *fftypes.ContractAPI, err error)
	SubscribeContract(ctx context.Context, ns, eventPath string, req *fftypes.ContractSubscribeRequest) (*fftypes.ContractSubscription, error)
	SubscribeContractAPI(ctx context.Context, ns, apiName, eventPath string, req *fftypes.ContractSubscribeRequest) (*fftypes.ContractSubscription, error)

	ValidateFFIAndSetPathnames(ctx context.Context, ffi *fftypes.FFI) error

	AddContractSubscription(ctx context.Context, ns string, sub *fftypes.ContractSubscriptionInput) (output *fftypes.ContractSubscription, err error)
	GetContractSubscriptionByNameOrID(ctx context.Context, ns, nameOrID string) (*fftypes.ContractSubscription, error)
	GetContractSubscriptions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.ContractSubscription, *database.FilterResult, error)
	DeleteContractSubscriptionByNameOrID(ctx context.Context, ns, nameOrID string) error
	GetContractEventByID(ctx context.Context, id *fftypes.UUID) (*fftypes.BlockchainEvent, error)
	GetContractEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.BlockchainEvent, *database.FilterResult, error)
}

type contractManager struct {
	database          database.Plugin
	publicStorage     publicstorage.Plugin
	broadcast         broadcast.Manager
	identity          identity.Manager
	blockchain        blockchain.Plugin
	ffiParamValidator fftypes.FFIParamValidator
}

func NewContractManager(ctx context.Context, database database.Plugin, publicStorage publicstorage.Plugin, broadcast broadcast.Manager, identity identity.Manager, blockchain blockchain.Plugin) (Manager, error) {
	if database == nil || publicStorage == nil || broadcast == nil || identity == nil || blockchain == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	v, err := blockchain.GetFFIParamValidator(ctx)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgPluginInitializationFailed)
	}
	return &contractManager{
		database:          database,
		publicStorage:     publicStorage,
		broadcast:         broadcast,
		identity:          identity,
		blockchain:        blockchain,
		ffiParamValidator: v,
	}, nil
}

func (cm *contractManager) newFFISchemaCompiler() *jsonschema.Compiler {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	f := FFIParamValidator{}
	c.RegisterExtension(f.GetExtensionName(), f.GetMetaSchema(), f)
	if cm.ffiParamValidator != nil {
		c.RegisterExtension(cm.ffiParamValidator.GetExtensionName(), cm.ffiParamValidator.GetMetaSchema(), cm.ffiParamValidator)
	}
	return c
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

	output = ffi
	msg, err := cm.broadcast.BroadcastDefinitionAsNode(ctx, ns, ffi, fftypes.SystemTagDefineFFI, waitConfirm)
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

func (cm *contractManager) GetFFIByIDWithChildren(ctx context.Context, id *fftypes.UUID) (ffi *fftypes.FFI, err error) {
	err = cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		ffi, err = cm.database.GetFFIByID(ctx, id)
		if err != nil || ffi == nil {
			return err
		}

		mfb := database.FFIMethodQueryFactory.NewFilter(ctx)
		ffi.Methods, _, err = cm.database.GetFFIMethods(ctx, mfb.Eq("interface", id))
		if err != nil {
			return err
		}

		efb := database.FFIEventQueryFactory.NewFilter(ctx)
		ffi.Events, _, err = cm.database.GetFFIEvents(ctx, efb.Eq("interface", id))
		if err != nil {
			return err
		}
		return nil
	})
	return ffi, err
}

func (cm *contractManager) GetFFIs(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.FFI, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	return cm.database.GetFFIs(ctx, ns, filter)
}

func (cm *contractManager) InvokeContract(ctx context.Context, ns string, req *fftypes.ContractCallRequest) (res interface{}, err error) {
	req.Key, err = cm.identity.ResolveSigningKey(ctx, req.Key)
	if err != nil {
		return nil, err
	}

	if req.Method, err = cm.resolveInvokeContractRequest(ctx, ns, req); err != nil {
		return nil, err
	}
	if err := cm.validateInvokeContractRequest(ctx, req); err != nil {
		return nil, err
	}

	switch req.Type {
	case fftypes.CallTypeInvoke:
		operationID := fftypes.NewUUID()
		return cm.blockchain.InvokeContract(ctx, operationID, req.Key, req.Location, req.Method, req.Input)
	case fftypes.CallTypeQuery:
		return cm.blockchain.QueryContract(ctx, req.Location, req.Method, req.Input)
	default:
		panic(fmt.Sprintf("unknown call type: %s", req.Type))
	}
}

func (cm *contractManager) InvokeContractAPI(ctx context.Context, ns, apiName, methodPath string, req *fftypes.ContractCallRequest) (interface{}, error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	if err != nil {
		return nil, err
	} else if api == nil || api.Interface == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	req.Interface = api.Interface.ID
	req.Method = &fftypes.FFIMethod{
		Pathname: methodPath,
	}
	if api.Location != nil {
		req.Location = api.Location
	}
	return cm.InvokeContract(ctx, ns, req)
}

func (cm *contractManager) resolveInvokeContractRequest(ctx context.Context, ns string, req *fftypes.ContractCallRequest) (method *fftypes.FFIMethod, err error) {
	if req.Method == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractMethodNotSet)
	}
	method = req.Method

	// We have a method name but no method signature - look up the method in the DB
	if method.Pathname == "" {
		method.Pathname = method.Name
	}
	if method.Pathname != "" && (method.Params == nil || method.Returns == nil) {
		if req.Interface == nil {
			return nil, i18n.NewError(ctx, i18n.MsgContractNoMethodSignature)
		}

		method, err = cm.database.GetFFIMethod(ctx, ns, req.Interface, method.Pathname)
		if err != nil || method == nil {
			return nil, i18n.NewError(ctx, i18n.MsgContractMethodResolveError, err)
		}
	}
	return method, nil
}

func (cm *contractManager) addContractURLs(httpServerURL string, api *fftypes.ContractAPI) {
	if api != nil {
		// These URLs must match the actual routes in apiserver.createMuxRouter()!
		baseURL := fmt.Sprintf("%s/namespaces/%s/apis/%s", httpServerURL, api.Namespace, api.Name)
		api.URLs.OpenAPI = baseURL + "/api/swagger.json"
		api.URLs.UI = baseURL + "/api"
	}
}

func (cm *contractManager) GetContractAPI(ctx context.Context, httpServerURL, ns, apiName string) (*fftypes.ContractAPI, error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	cm.addContractURLs(httpServerURL, api)
	return api, err
}

func (cm *contractManager) GetContractAPIs(ctx context.Context, httpServerURL, ns string, filter database.AndFilter) ([]*fftypes.ContractAPI, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	apis, fr, err := cm.database.GetContractAPIs(ctx, ns, filter)
	for _, api := range apis {
		cm.addContractURLs(httpServerURL, api)
	}
	return apis, fr, err
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

	err = cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		existing, err := cm.database.GetContractAPIByName(ctx, api.Namespace, api.Name)
		if existing != nil && err == nil {
			if !api.LocationAndLedgerEquals(existing) {
				return i18n.NewError(ctx, i18n.MsgContractLocationExists)
			}
		}

		if err := cm.resolveFFIReference(ctx, ns, api.Interface); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	msg, err := cm.broadcast.BroadcastDefinitionAsNode(ctx, ns, api, fftypes.SystemTagDefineContractAPI, waitConfirm)
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
	if err := ffi.Validate(ctx, false); err != nil {
		return err
	}

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
		if err := cm.validateFFIParam(ctx, param); err != nil {
			return err
		}
	}
	for _, param := range method.Returns {
		if err := cm.validateFFIParam(ctx, param); err != nil {
			return err
		}
	}
	return nil
}

func (cm *contractManager) validateFFIParam(ctx context.Context, param *fftypes.FFIParam) error {
	c := cm.newFFISchemaCompiler()
	if err := c.AddResource(param.Name, strings.NewReader(param.Schema.String())); err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgFFISchemaParseFail, param.Name)
	}
	if _, err := c.Compile(param.Name); err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgFFISchemaCompileFail, param.Name)
	}
	return nil
}

func (cm *contractManager) validateFFIEvent(ctx context.Context, event *fftypes.FFIEventDefinition) error {
	if event.Name == "" {
		return i18n.NewError(ctx, i18n.MsgEventNameMustBeSet)
	}
	for _, param := range event.Params {
		if err := cm.validateFFIParam(ctx, param); err != nil {
			return err
		}
	}
	return nil
}

func (cm *contractManager) validateInvokeContractRequest(ctx context.Context, req *fftypes.ContractCallRequest) error {
	if err := cm.validateFFIMethod(ctx, req.Method); err != nil {
		return err
	}

	for _, param := range req.Method.Params {
		value, ok := req.Input[param.Name]
		if !ok {
			return i18n.NewError(ctx, i18n.MsgContractMissingInputArgument, param.Name)
		}
		if err := cm.checkParamSchema(ctx, value, param); err != nil {
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

	err = cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		if sub.Name != "" {
			if err := fftypes.ValidateFFNameField(ctx, sub.Name, "name"); err != nil {
				return err
			}
			if existing, err := cm.database.GetContractSubscription(ctx, ns, sub.Name); err != nil {
				return err
			} else if existing != nil {
				return i18n.NewError(ctx, i18n.MsgContractSubscriptionExists, ns, sub.Name)
			}
		}

		if sub.Interface != nil {
			if err := cm.resolveFFIReference(ctx, ns, sub.Interface); err != nil {
				return err
			}
		}

		if sub.Event == nil {
			if sub.EventID == nil {
				return i18n.NewError(ctx, i18n.MsgSubscriptionNoEvent)
			}

			event, err := cm.database.GetFFIEventByID(ctx, sub.EventID)
			if err != nil {
				return err
			}
			if event == nil || event.Namespace != sub.Namespace {
				return i18n.NewError(ctx, i18n.MsgSubscriptionEventNotFound, sub.Namespace, sub.EventID)
			}
			// Copy the event definition into the subscription
			sub.Event = &fftypes.FFISerializedEvent{
				FFIEventDefinition: event.FFIEventDefinition,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
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
	return cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		sub, err := cm.GetContractSubscriptionByNameOrID(ctx, ns, nameOrID)
		if err != nil {
			return err
		}
		if err = cm.blockchain.DeleteSubscription(ctx, sub); err != nil {
			return err
		}
		return cm.database.DeleteContractSubscriptionByID(ctx, sub.ID)
	})
}

func (cm *contractManager) SubscribeContract(ctx context.Context, ns, eventPath string, req *fftypes.ContractSubscribeRequest) (*fftypes.ContractSubscription, error) {
	event, err := cm.database.GetFFIEvent(ctx, ns, req.Interface, eventPath)
	if err != nil || event == nil {
		return nil, i18n.NewError(ctx, i18n.MsgContractEventResolveError)
	}

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Interface: &fftypes.FFIReference{
				ID: req.Interface,
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

	req.Interface = api.Interface.ID
	if api.Location != nil {
		req.Location = api.Location
	}

	return cm.SubscribeContract(ctx, ns, eventPath, req)
}

func (cm *contractManager) GetContractEventByID(ctx context.Context, id *fftypes.UUID) (*fftypes.BlockchainEvent, error) {
	return cm.database.GetBlockchainEventByID(ctx, id)
}

func (cm *contractManager) GetContractEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.BlockchainEvent, *database.FilterResult, error) {
	return cm.database.GetBlockchainEvents(ctx, cm.scopeNS(ns, filter))
}

func (cm *contractManager) checkParamSchema(ctx context.Context, input interface{}, param *fftypes.FFIParam) error {
	// TODO: Cache the compiled schema?
	c := jsonschema.NewCompiler()
	err := c.AddResource(param.Name, strings.NewReader(param.Schema.String()))
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgFFISchemaParseFail, param.Name)
	}
	schema, err := c.Compile(param.Name)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgFFIValidationFail, param.Name, param.Schema)
	}
	if err := schema.Validate(input); err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgFFIValidationFail, param.Name)
	}
	return nil
}
