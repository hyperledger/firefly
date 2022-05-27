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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Manager interface {
	core.Named

	BroadcastFFI(ctx context.Context, ns string, ffi *core.FFI, waitConfirm bool) (output *core.FFI, err error)
	GetFFI(ctx context.Context, ns, name, version string) (*core.FFI, error)
	GetFFIWithChildren(ctx context.Context, ns, name, version string) (*core.FFI, error)
	GetFFIByID(ctx context.Context, id *fftypes.UUID) (*core.FFI, error)
	GetFFIByIDWithChildren(ctx context.Context, id *fftypes.UUID) (*core.FFI, error)
	GetFFIs(ctx context.Context, ns string, filter database.AndFilter) ([]*core.FFI, *database.FilterResult, error)

	InvokeContract(ctx context.Context, ns string, req *core.ContractCallRequest, waitConfirm bool) (interface{}, error)
	InvokeContractAPI(ctx context.Context, ns, apiName, methodPath string, req *core.ContractCallRequest, waitConfirm bool) (interface{}, error)
	GetContractAPI(ctx context.Context, httpServerURL, ns, apiName string) (*core.ContractAPI, error)
	GetContractAPIInterface(ctx context.Context, ns, apiName string) (*core.FFI, error)
	GetContractAPIs(ctx context.Context, httpServerURL, ns string, filter database.AndFilter) ([]*core.ContractAPI, *database.FilterResult, error)
	BroadcastContractAPI(ctx context.Context, httpServerURL, ns string, api *core.ContractAPI, waitConfirm bool) (output *core.ContractAPI, err error)

	ValidateFFIAndSetPathnames(ctx context.Context, ffi *core.FFI) error

	AddContractListener(ctx context.Context, ns string, listener *core.ContractListenerInput) (output *core.ContractListener, err error)
	AddContractAPIListener(ctx context.Context, ns, apiName, eventPath string, listener *core.ContractListener) (output *core.ContractListener, err error)
	GetContractListenerByNameOrID(ctx context.Context, ns, nameOrID string) (*core.ContractListener, error)
	GetContractListeners(ctx context.Context, ns string, filter database.AndFilter) ([]*core.ContractListener, *database.FilterResult, error)
	GetContractAPIListeners(ctx context.Context, ns string, apiName, eventPath string, filter database.AndFilter) ([]*core.ContractListener, *database.FilterResult, error)
	DeleteContractListenerByNameOrID(ctx context.Context, ns, nameOrID string) error
	GenerateFFI(ctx context.Context, ns string, generationRequest *core.FFIGenerationRequest) (*core.FFI, error)

	// From operations.OperationHandler
	PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error)
	RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error)
}

type contractManager struct {
	database          database.Plugin
	txHelper          txcommon.Helper
	broadcast         broadcast.Manager
	identity          identity.Manager
	blockchain        blockchain.Plugin
	ffiParamValidator core.FFIParamValidator
	operations        operations.Manager
	syncasync         syncasync.Bridge
}

func NewContractManager(ctx context.Context, di database.Plugin, bm broadcast.Manager, im identity.Manager, bi blockchain.Plugin, om operations.Manager, txHelper txcommon.Helper, sa syncasync.Bridge) (Manager, error) {
	if di == nil || bm == nil || im == nil || bi == nil || om == nil || txHelper == nil || sa == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "ContractManager")
	}
	v, err := bi.GetFFIParamValidator(ctx)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgPluginInitializationFailed)
	}

	cm := &contractManager{
		database:          di,
		txHelper:          txHelper,
		broadcast:         bm,
		identity:          im,
		blockchain:        bi,
		ffiParamValidator: v,
		operations:        om,
		syncasync:         sa,
	}

	om.RegisterHandler(ctx, cm, []core.OpType{
		core.OpTypeBlockchainInvoke,
	})

	return cm, nil
}

func (cm *contractManager) Name() string {
	return "ContractManager"
}

func (cm *contractManager) newFFISchemaCompiler() *jsonschema.Compiler {
	c := core.NewFFISchemaCompiler()
	if cm.ffiParamValidator != nil {
		c.RegisterExtension(cm.ffiParamValidator.GetExtensionName(), cm.ffiParamValidator.GetMetaSchema(), cm.ffiParamValidator)
	}
	return c
}

func (cm *contractManager) BroadcastFFI(ctx context.Context, ns string, ffi *core.FFI, waitConfirm bool) (output *core.FFI, err error) {
	ffi.ID = fftypes.NewUUID()
	ffi.Namespace = ns

	existing, err := cm.database.GetFFI(ctx, ffi.Namespace, ffi.Name, ffi.Version)
	if existing != nil && err == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgContractInterfaceExists, ffi.Namespace, ffi.Name, ffi.Version)
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
	msg, err := cm.broadcast.BroadcastDefinitionAsNode(ctx, ns, ffi, core.SystemTagDefineFFI, waitConfirm)
	if err != nil {
		return nil, err
	}
	output.Message = msg.Header.ID
	return ffi, nil
}

func (cm *contractManager) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (cm *contractManager) GetFFI(ctx context.Context, ns, name, version string) (*core.FFI, error) {
	return cm.database.GetFFI(ctx, ns, name, version)
}

func (cm *contractManager) GetFFIWithChildren(ctx context.Context, ns, name, version string) (*core.FFI, error) {
	ffi, err := cm.GetFFI(ctx, ns, name, version)
	if err == nil {
		err = cm.getFFIChildren(ctx, ffi)
	}
	return ffi, err
}

func (cm *contractManager) GetFFIByID(ctx context.Context, id *fftypes.UUID) (*core.FFI, error) {
	return cm.database.GetFFIByID(ctx, id)
}

func (cm *contractManager) getFFIChildren(ctx context.Context, ffi *core.FFI) (err error) {
	mfb := database.FFIMethodQueryFactory.NewFilter(ctx)
	ffi.Methods, _, err = cm.database.GetFFIMethods(ctx, mfb.Eq("interface", ffi.ID))
	if err != nil {
		return err
	}

	efb := database.FFIEventQueryFactory.NewFilter(ctx)
	ffi.Events, _, err = cm.database.GetFFIEvents(ctx, efb.Eq("interface", ffi.ID))
	if err != nil {
		return err
	}

	for _, event := range ffi.Events {
		event.Signature = cm.blockchain.GenerateEventSignature(ctx, &event.FFIEventDefinition)
	}
	return nil
}

func (cm *contractManager) GetFFIByIDWithChildren(ctx context.Context, id *fftypes.UUID) (ffi *core.FFI, err error) {
	err = cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		ffi, err = cm.database.GetFFIByID(ctx, id)
		if err != nil || ffi == nil {
			return err
		}
		return cm.getFFIChildren(ctx, ffi)
	})
	return ffi, err
}

func (cm *contractManager) GetFFIs(ctx context.Context, ns string, filter database.AndFilter) ([]*core.FFI, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	return cm.database.GetFFIs(ctx, ns, filter)
}

func (cm *contractManager) writeInvokeTransaction(ctx context.Context, ns string, req *core.ContractCallRequest) (*core.Operation, error) {
	txid, err := cm.txHelper.SubmitNewTransaction(ctx, ns, core.TransactionTypeContractInvoke)
	if err != nil {
		return nil, err
	}

	op := core.NewOperation(
		cm.blockchain,
		ns,
		txid,
		core.OpTypeBlockchainInvoke)
	if err = addBlockchainInvokeInputs(op, req); err == nil {
		err = cm.database.InsertOperation(ctx, op)
	}
	return op, err
}

func (cm *contractManager) InvokeContract(ctx context.Context, ns string, req *core.ContractCallRequest, waitConfirm bool) (res interface{}, err error) {
	req.Key, err = cm.identity.NormalizeSigningKey(ctx, ns, req.Key, identity.KeyNormalizationBlockchainPlugin)
	if err != nil {
		return nil, err
	}

	var op *core.Operation
	err = cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		if err = cm.resolveInvokeContractRequest(ctx, ns, req); err != nil {
			return err
		}
		if err := cm.validateInvokeContractRequest(ctx, req); err != nil {
			return err
		}
		if req.Type == core.CallTypeInvoke {
			op, err = cm.writeInvokeTransaction(ctx, ns, req)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	switch req.Type {
	case core.CallTypeInvoke:
		send := func(ctx context.Context) error {
			_, err := cm.operations.RunOperation(ctx, opBlockchainInvoke(op, req))
			return err
		}
		if waitConfirm {
			return cm.syncasync.WaitForInvokeOperation(ctx, ns, op.ID, send)
		}
		err = send(ctx)
		return op, err
	case core.CallTypeQuery:
		return cm.blockchain.QueryContract(ctx, req.Location, req.Method, req.Input)
	default:
		panic(fmt.Sprintf("unknown call type: %s", req.Type))
	}
}

func (cm *contractManager) InvokeContractAPI(ctx context.Context, ns, apiName, methodPath string, req *core.ContractCallRequest, waitConfirm bool) (interface{}, error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	if err != nil {
		return nil, err
	} else if api == nil || api.Interface == nil {
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	req.Interface = api.Interface.ID
	req.MethodPath = methodPath
	if api.Location != nil {
		req.Location = api.Location
	}
	return cm.InvokeContract(ctx, ns, req, waitConfirm)
}

func (cm *contractManager) resolveInvokeContractRequest(ctx context.Context, ns string, req *core.ContractCallRequest) (err error) {
	if req.Method == nil {
		if req.MethodPath == "" || req.Interface == nil {
			return i18n.NewError(ctx, coremsgs.MsgContractMethodNotSet)
		}
		req.Method, err = cm.database.GetFFIMethod(ctx, ns, req.Interface, req.MethodPath)
		if err != nil || req.Method == nil {
			return i18n.NewError(ctx, coremsgs.MsgContractMethodResolveError, err)
		}
	}
	return nil
}

func (cm *contractManager) addContractURLs(httpServerURL string, api *core.ContractAPI) {
	if api != nil {
		// These URLs must match the actual routes in apiserver.createMuxRouter()!
		baseURL := fmt.Sprintf("%s/namespaces/%s/apis/%s", httpServerURL, api.Namespace, api.Name)
		api.URLs.OpenAPI = baseURL + "/api/swagger.json"
		api.URLs.UI = baseURL + "/api"
	}
}

func (cm *contractManager) GetContractAPI(ctx context.Context, httpServerURL, ns, apiName string) (*core.ContractAPI, error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	cm.addContractURLs(httpServerURL, api)
	return api, err
}

func (cm *contractManager) GetContractAPIInterface(ctx context.Context, ns, apiName string) (*core.FFI, error) {
	api, err := cm.GetContractAPI(ctx, "", ns, apiName)
	if err != nil || api == nil {
		return nil, err
	}
	return cm.GetFFIByIDWithChildren(ctx, api.Interface.ID)
}

func (cm *contractManager) GetContractAPIs(ctx context.Context, httpServerURL, ns string, filter database.AndFilter) ([]*core.ContractAPI, *database.FilterResult, error) {
	filter = cm.scopeNS(ns, filter)
	apis, fr, err := cm.database.GetContractAPIs(ctx, ns, filter)
	for _, api := range apis {
		cm.addContractURLs(httpServerURL, api)
	}
	return apis, fr, err
}

func (cm *contractManager) resolveFFIReference(ctx context.Context, ns string, ref *core.FFIReference) error {
	switch {
	case ref == nil:
		return i18n.NewError(ctx, coremsgs.MsgContractInterfaceNotFound, "")

	case ref.ID != nil:
		ffi, err := cm.database.GetFFIByID(ctx, ref.ID)
		if err != nil {
			return err
		} else if ffi == nil {
			return i18n.NewError(ctx, coremsgs.MsgContractInterfaceNotFound, ref.ID)
		}
		return nil

	case ref.Name != "" && ref.Version != "":
		ffi, err := cm.database.GetFFI(ctx, ns, ref.Name, ref.Version)
		if err != nil {
			return err
		} else if ffi == nil {
			return i18n.NewError(ctx, coremsgs.MsgContractInterfaceNotFound, ref.Name)
		}
		ref.ID = ffi.ID
		return nil

	default:
		return i18n.NewError(ctx, coremsgs.MsgContractInterfaceNotFound, ref.Name)
	}
}

func (cm *contractManager) BroadcastContractAPI(ctx context.Context, httpServerURL, ns string, api *core.ContractAPI, waitConfirm bool) (output *core.ContractAPI, err error) {
	api.ID = fftypes.NewUUID()
	api.Namespace = ns

	if api.Location != nil {
		if api.Location, err = cm.blockchain.NormalizeContractLocation(ctx, api.Location); err != nil {
			return nil, err
		}
	}

	err = cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		existing, err := cm.database.GetContractAPIByName(ctx, api.Namespace, api.Name)
		if existing != nil && err == nil {
			if !api.LocationAndLedgerEquals(existing) {
				return i18n.NewError(ctx, coremsgs.MsgContractLocationExists)
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

	msg, err := cm.broadcast.BroadcastDefinitionAsNode(ctx, ns, api, core.SystemTagDefineContractAPI, waitConfirm)
	if err != nil {
		return nil, err
	}
	api.Message = msg.Header.ID
	cm.addContractURLs(httpServerURL, api)
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

func (cm *contractManager) ValidateFFIAndSetPathnames(ctx context.Context, ffi *core.FFI) error {
	if err := ffi.Validate(ctx, false); err != nil {
		return err
	}

	methodPathNames := map[string]bool{}
	for _, method := range ffi.Methods {
		method.Interface = ffi.ID
		method.Namespace = ffi.Namespace
		method.Pathname = cm.uniquePathName(method.Name, methodPathNames)
		if err := cm.validateFFIMethod(ctx, method); err != nil {
			return err
		}
	}

	eventPathNames := map[string]bool{}
	for _, event := range ffi.Events {
		event.Interface = ffi.ID
		event.Namespace = ffi.Namespace
		event.Pathname = cm.uniquePathName(event.Name, eventPathNames)
		if err := cm.validateFFIEvent(ctx, &event.FFIEventDefinition); err != nil {
			return err
		}
	}
	return nil
}

func (cm *contractManager) validateFFIMethod(ctx context.Context, method *core.FFIMethod) error {
	if method.Name == "" {
		return i18n.NewError(ctx, coremsgs.MsgMethodNameMustBeSet)
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

func (cm *contractManager) validateFFIParam(ctx context.Context, param *core.FFIParam) error {
	c := cm.newFFISchemaCompiler()
	if err := c.AddResource(param.Name, strings.NewReader(param.Schema.String())); err != nil {
		return i18n.WrapError(ctx, err, coremsgs.MsgFFISchemaParseFail, param.Name)
	}
	if _, err := c.Compile(param.Name); err != nil {
		return i18n.WrapError(ctx, err, coremsgs.MsgFFISchemaCompileFail, param.Name)
	}
	return nil
}

func (cm *contractManager) validateFFIEvent(ctx context.Context, event *core.FFIEventDefinition) error {
	if event.Name == "" {
		return i18n.NewError(ctx, coremsgs.MsgEventNameMustBeSet)
	}
	for _, param := range event.Params {
		if err := cm.validateFFIParam(ctx, param); err != nil {
			return err
		}
	}
	return nil
}

func (cm *contractManager) validateInvokeContractRequest(ctx context.Context, req *core.ContractCallRequest) error {
	if err := cm.validateFFIMethod(ctx, req.Method); err != nil {
		return err
	}

	for _, param := range req.Method.Params {
		value, ok := req.Input[param.Name]
		if !ok {
			return i18n.NewError(ctx, coremsgs.MsgContractMissingInputArgument, param.Name)
		}
		if err := cm.checkParamSchema(ctx, value, param); err != nil {
			return err
		}
	}

	return nil
}

func (cm *contractManager) resolveEvent(ctx context.Context, ns string, ffi *core.FFIReference, eventPath string) (*core.FFISerializedEvent, error) {
	if err := cm.resolveFFIReference(ctx, ns, ffi); err != nil {
		return nil, err
	}
	event, err := cm.database.GetFFIEvent(ctx, ns, ffi.ID, eventPath)
	if err != nil {
		return nil, err
	} else if event == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgEventNotFound, eventPath)
	}
	return &core.FFISerializedEvent{FFIEventDefinition: event.FFIEventDefinition}, nil
}

func (cm *contractManager) AddContractListener(ctx context.Context, ns string, listener *core.ContractListenerInput) (output *core.ContractListener, err error) {
	listener.ID = fftypes.NewUUID()
	listener.Namespace = ns

	if err := core.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}
	if listener.Name != "" {
		if err := core.ValidateFFNameField(ctx, listener.Name, "name"); err != nil {
			return nil, err
		}
	}
	if err := core.ValidateFFNameField(ctx, listener.Topic, "topic"); err != nil {
		return nil, err
	}
	if listener.Location, err = cm.blockchain.NormalizeContractLocation(ctx, listener.Location); err != nil {
		return nil, err
	}

	if listener.Options == nil {
		listener.Options = cm.getDefaultContractListenerOptions()
	} else if listener.Options.FirstEvent == "" {
		listener.Options.FirstEvent = cm.getDefaultContractListenerOptions().FirstEvent
	}

	err = cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		// Namespace + Name must be unique
		if listener.Name != "" {
			if existing, err := cm.database.GetContractListener(ctx, ns, listener.Name); err != nil {
				return err
			} else if existing != nil {
				return i18n.NewError(ctx, coremsgs.MsgContractListenerNameExists, ns, listener.Name)
			}
		}

		if listener.Event == nil {
			if listener.EventPath == "" || listener.Interface == nil {
				return i18n.NewError(ctx, coremsgs.MsgListenerNoEvent)
			}
			// Copy the event definition into the listener
			if listener.Event, err = cm.resolveEvent(ctx, ns, listener.Interface, listener.EventPath); err != nil {
				return err
			}
		} else {
			listener.Interface = nil
		}

		// Namespace + Topic + Location + Signature must be unique
		listener.Signature = cm.blockchain.GenerateEventSignature(ctx, &listener.Event.FFIEventDefinition)
		fb := database.ContractListenerQueryFactory.NewFilter(ctx)
		if existing, _, err := cm.database.GetContractListeners(ctx, fb.And(
			fb.Eq("namespace", listener.Namespace),
			fb.Eq("topic", listener.Topic),
			fb.Eq("location", listener.Location.Bytes()),
			fb.Eq("signature", listener.Signature),
		)); err != nil {
			return err
		} else if len(existing) > 0 {
			return i18n.NewError(ctx, coremsgs.MsgContractListenerExists)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := cm.validateFFIEvent(ctx, &listener.Event.FFIEventDefinition); err != nil {
		return nil, err
	}
	if err = cm.blockchain.AddContractListener(ctx, listener); err != nil {
		return nil, err
	}
	if listener.Name == "" {
		listener.Name = listener.BackendID
	}
	if err = cm.database.UpsertContractListener(ctx, &listener.ContractListener); err != nil {
		return nil, err
	}

	return &listener.ContractListener, err
}

func (cm *contractManager) AddContractAPIListener(ctx context.Context, ns, apiName, eventPath string, listener *core.ContractListener) (output *core.ContractListener, err error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	if err != nil {
		return nil, err
	} else if api == nil || api.Interface == nil {
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}

	input := &core.ContractListenerInput{ContractListener: *listener}
	input.Interface = &core.FFIReference{ID: api.Interface.ID}
	input.EventPath = eventPath
	if api.Location != nil {
		input.Location = api.Location
	}

	return cm.AddContractListener(ctx, ns, input)
}

func (cm *contractManager) GetContractListenerByNameOrID(ctx context.Context, ns, nameOrID string) (listener *core.ContractListener, err error) {
	id, err := fftypes.ParseUUID(ctx, nameOrID)
	if err != nil {
		if err := core.ValidateFFNameField(ctx, nameOrID, "name"); err != nil {
			return nil, err
		}
		if listener, err = cm.database.GetContractListener(ctx, ns, nameOrID); err != nil {
			return nil, err
		}
	} else if listener, err = cm.database.GetContractListenerByID(ctx, id); err != nil {
		return nil, err
	}
	if listener == nil {
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	return listener, nil
}

func (cm *contractManager) GetContractListeners(ctx context.Context, ns string, filter database.AndFilter) ([]*core.ContractListener, *database.FilterResult, error) {
	return cm.database.GetContractListeners(ctx, cm.scopeNS(ns, filter))
}

func (cm *contractManager) GetContractAPIListeners(ctx context.Context, ns string, apiName, eventPath string, filter database.AndFilter) ([]*core.ContractListener, *database.FilterResult, error) {
	api, err := cm.database.GetContractAPIByName(ctx, ns, apiName)
	if err != nil {
		return nil, nil, err
	} else if api == nil || api.Interface == nil {
		return nil, nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	event, err := cm.resolveEvent(ctx, ns, api.Interface, eventPath)
	if err != nil {
		return nil, nil, err
	}
	signature := cm.blockchain.GenerateEventSignature(ctx, &event.FFIEventDefinition)

	fb := database.ContractListenerQueryFactory.NewFilter(ctx)
	f := fb.And(
		fb.Eq("interface", api.Interface.ID),
		fb.Eq("signature", signature),
		filter,
	)
	if !api.Location.IsNil() {
		f = fb.And(f, fb.Eq("location", api.Location.Bytes()))
	}
	return cm.database.GetContractListeners(ctx, cm.scopeNS(ns, f))
}

func (cm *contractManager) DeleteContractListenerByNameOrID(ctx context.Context, ns, nameOrID string) error {
	return cm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		listener, err := cm.GetContractListenerByNameOrID(ctx, ns, nameOrID)
		if err != nil {
			return err
		}
		if err = cm.blockchain.DeleteContractListener(ctx, listener); err != nil {
			return err
		}
		return cm.database.DeleteContractListenerByID(ctx, listener.ID)
	})
}

func (cm *contractManager) checkParamSchema(ctx context.Context, input interface{}, param *core.FFIParam) error {
	// TODO: Cache the compiled schema?
	c := jsonschema.NewCompiler()
	err := c.AddResource(param.Name, strings.NewReader(param.Schema.String()))
	if err != nil {
		return i18n.WrapError(ctx, err, coremsgs.MsgFFISchemaParseFail, param.Name)
	}
	schema, err := c.Compile(param.Name)
	if err != nil {
		return i18n.WrapError(ctx, err, coremsgs.MsgFFIValidationFail, param.Name, param.Schema)
	}
	if err := schema.Validate(input); err != nil {
		return i18n.WrapError(ctx, err, coremsgs.MsgFFIValidationFail, param.Name)
	}
	return nil
}

func (cm *contractManager) GenerateFFI(ctx context.Context, ns string, generationRequest *core.FFIGenerationRequest) (*core.FFI, error) {
	generationRequest.Namespace = ns
	return cm.blockchain.GenerateFFI(ctx, generationRequest)
}

func (cm *contractManager) getDefaultContractListenerOptions() *core.ContractListenerOptions {
	return &core.ContractListenerOptions{
		FirstEvent: string(core.SubOptsFirstEventNewest),
	}
}
