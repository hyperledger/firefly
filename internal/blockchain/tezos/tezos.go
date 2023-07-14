// Copyright © 2023 Kaleido, Inc.
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

package tezos

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"blockwatch.cc/tzgo/micheline"
	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/blockchain/common"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
)

type Tezos struct {
	ctx                  context.Context
	cancelCtx            context.CancelFunc
	callbacks            common.BlockchainCallbacks
	client               *resty.Client
	addressResolveAlways bool
	addressResolver      *addressResolver
	metrics              metrics.Manager
	tezosconnectConf     config.Section
}

type tezosError struct {
	Error string `json:"error,omitempty"`
}

type Location struct {
	Address string `json:"address"`
}

type TezosconnectMessageHeaders struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

type PayloadSchema struct {
	Type        string        `json:"type"`
	PrefixItems []*PrefixItem `json:"prefixItems"`
}

type PrefixItem struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	Details paramDetails `json:"details,omitempty"`
}

type paramDetails struct {
	Type           string             `json:"type"`
	InternalType   string             `json:"internalType"`
	InternalSchema fftypes.JSONObject `json:"internalSchema"`
	Kind           string             `json:"kind"`
	Variants       []string           `json:"variants"`
}
type ffiParamSchema struct {
	Type    string       `json:"type,omitempty"`
	Details paramDetails `json:"details,omitempty"`
}

type ffiMethodAndErrors struct {
	method *fftypes.FFIMethod
	errors []*fftypes.FFIError
}

var addressVerify = regexp.MustCompile("^(tz[1-4]|[Kk][Tt]1)[1-9A-Za-z]{33}$")

func (t *Tezos) Name() string {
	return "tezos"
}

func (t *Tezos) VerifierType() core.VerifierType {
	return core.VerifierTypeTezosAddress
}

func (t *Tezos) Init(ctx context.Context, cancelCtx context.CancelFunc, conf config.Section, metrics metrics.Manager, cacheManager cache.Manager) (err error) {
	t.InitConfig(conf)
	tezosconnectConf := t.tezosconnectConf
	addressResolverConf := conf.SubSection(AddressResolverConfigKey)

	t.ctx = log.WithLogField(ctx, "proto", "tezos")
	t.cancelCtx = cancelCtx
	t.metrics = metrics
	t.callbacks = common.NewBlockchainCallbacks()

	if addressResolverConf.GetString(AddressResolverURLTemplate) != "" {
		// Check if we need to invoke the address resolver (without caching) on every call
		t.addressResolveAlways = addressResolverConf.GetBool(AddressResolverAlwaysResolve)
		if t.addressResolver, err = newAddressResolver(ctx, addressResolverConf, cacheManager, !t.addressResolveAlways); err != nil {
			return err
		}
	}

	if tezosconnectConf.GetString(ffresty.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, coremsgs.MsgMissingPluginConfig, "url", tezosconnectConf)
	}

	t.client, err = ffresty.New(t.ctx, tezosconnectConf)
	if err != nil {
		return err
	}

	return nil
}

func (t *Tezos) SetHandler(namespace string, handler blockchain.Callbacks) {
	t.callbacks.SetHandler(namespace, handler)
}

func (t *Tezos) SetOperationHandler(namespace string, handler core.OperationCallbacks) {
	t.callbacks.SetOperationalHandler(namespace, handler)
}

func (t *Tezos) Start() (err error) {
	// TODO: impl
	return nil
}

func (t *Tezos) Capabilities() *blockchain.Capabilities {
	// TODO: impl
	return nil
}

func (t *Tezos) AddFireflySubscription(ctx context.Context, namespace *core.Namespace, contract *blockchain.MultipartyContract) (string, error) {
	// TODO: impl
	return "", nil
}

func (t *Tezos) RemoveFireflySubscription(ctx context.Context, subID string) {
	// TODO: impl
}

func (t *Tezos) ResolveSigningKey(ctx context.Context, key string, intent blockchain.ResolveKeyIntent) (resolved string, err error) {
	if !t.addressResolveAlways {
		// If there's no address resolver plugin, or addressResolveAlways is false,
		// we check if it's already an tezos address - in which case we can just return it.
		resolved, err = formatTezosAddress(ctx, key)
	}
	if t.addressResolveAlways || (err != nil && t.addressResolver != nil) {
		// Either it's not a valid tezos address,
		// or we've been configured to invoke the address resolver on every call
		resolved, err = t.addressResolver.ResolveSigningKey(ctx, key, intent)
		if err == nil {
			log.L(ctx).Infof("Key '%s' resolved to '%s'", key, resolved)
			return resolved, nil
		}
	}
	return resolved, err
}

func (t *Tezos) SubmitBatchPin(ctx context.Context, nsOpID, networkNamespace, signingKey string, batch *blockchain.BatchPin, location *fftypes.JSONAny) error {
	// TODO: impl
	return nil
}

func (t *Tezos) SubmitNetworkAction(ctx context.Context, nsOpID string, signingKey string, action core.NetworkActionType, location *fftypes.JSONAny) error {
	// TODO: impl
	return nil
}

func (t *Tezos) DeployContract(ctx context.Context, nsOpID, signingKey string, definition, contract *fftypes.JSONAny, input []interface{}, options map[string]interface{}) error {
	return i18n.NewError(ctx, coremsgs.MsgNotSupportedByBlockchainPlugin)
}

func (t *Tezos) ValidateInvokeRequest(ctx context.Context, parsedMethod interface{}, input map[string]interface{}, hasMessage bool) error {
	// No additional validation beyond what is enforced by Contract Manager
	_, _, err := t.recoverFFI(ctx, parsedMethod)
	return err
}

func (t *Tezos) InvokeContract(ctx context.Context, nsOpID string, signingKey string, location *fftypes.JSONAny, parsedMethod interface{}, input map[string]interface{}, options map[string]interface{}, batch *blockchain.BatchPin) error {
	method, _, err := t.recoverFFI(ctx, parsedMethod)
	if err != nil {
		return err
	}

	tezosLocation, err := t.parseContractLocation(ctx, location)
	if err != nil {
		return err
	}

	// Build the payload schema for the method parameters
	prefixItems := make([]*PrefixItem, len(method.Params))
	for i, param := range method.Params {
		var paramSchema ffiParamSchema
		if err := json.Unmarshal(param.Schema.Bytes(), &paramSchema); err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, fmt.Sprintf("%s.schema", param.Name))
		}

		prefixItems[i] = &PrefixItem{
			Name:    param.Name,
			Type:    paramSchema.Type,
			Details: paramSchema.Details,
		}
	}

	payloadSchema := &PayloadSchema{
		Type:        "array",
		PrefixItems: prefixItems,
	}

	schemaBytes, _ := json.Marshal(payloadSchema)
	var processSchemaReq map[string]interface{}
	_ = json.Unmarshal(schemaBytes, &processSchemaReq)

	michelsonInput, err := processArgs(processSchemaReq, input, method.Name)

	if batch != nil {
		// TODO: add batch pin support
	}

	return t.invokeContractMethod(ctx, tezosLocation.Address, method.Name, signingKey, nsOpID, michelsonInput, options)
}

func (t *Tezos) QueryContract(ctx context.Context, signingKey string, location *fftypes.JSONAny, parsedMethod interface{}, input map[string]interface{}, options map[string]interface{}) (interface{}, error) {
	// TODO: impl
	return nil, nil
}

func (f *Tezos) ParseInterface(ctx context.Context, method *fftypes.FFIMethod, errors []*fftypes.FFIError) (interface{}, error) {
	return &ffiMethodAndErrors{
		method: method,
		errors: errors,
	}, nil
}

func (t *Tezos) NormalizeContractLocation(ctx context.Context, ntype blockchain.NormalizeType, location *fftypes.JSONAny) (result *fftypes.JSONAny, err error) {
	parsed, err := t.parseContractLocation(ctx, location)
	if err != nil {
		return nil, err
	}
	return t.encodeContractLocation(ctx, parsed)
}

func (e *Tezos) AddContractListener(ctx context.Context, listener *core.ContractListener) (err error) {
	// TODO: impl
	return nil
}

func (t *Tezos) DeleteContractListener(ctx context.Context, subscription *core.ContractListener, okNotFound bool) error {
	// TODO: impl
	return nil
}

func (t *Tezos) GetContractListenerStatus(ctx context.Context, subID string, okNotFound bool) (found bool, status interface{}, err error) {
	// TODO: impl
	return false, nil, nil
}

func (t *Tezos) GetFFIParamValidator(ctx context.Context) (fftypes.FFIParamValidator, error) {
	// TODO: impl
	return nil, nil
}

func (t *Tezos) GenerateEventSignature(ctx context.Context, event *fftypes.FFIEventDefinition) string {
	// TODO: impl
	return ""
}

func (t *Tezos) GenerateErrorSignature(ctx context.Context, event *fftypes.FFIErrorDefinition) string {
	// TODO: impl
	return ""
}

func (t *Tezos) GenerateFFI(ctx context.Context, generationRequest *fftypes.FFIGenerationRequest) (*fftypes.FFI, error) {
	// TODO: impl
	return nil, i18n.NewError(ctx, coremsgs.MsgFFIGenerationUnsupported)
}

// TODO: should return string instead of int
// Mainnet: NetXdQprcVkpaWU
// Delphi Testnet: NetXm8tYqnMWky1
// Edo Testnet: NetXjD3HPJJjmcd
func (t *Tezos) GetNetworkVersion(ctx context.Context, location *fftypes.JSONAny) (version int, err error) {
	// TODO: impl
	return 2, nil
}

func (t *Tezos) GetAndConvertDeprecatedContractConfig(ctx context.Context) (location *fftypes.JSONAny, fromBlock string, err error) {
	return nil, "", nil
}

func (t *Tezos) GetTransactionStatus(ctx context.Context, operation *core.Operation) (interface{}, error) {
	// TODO: impl
	return nil, nil
}

func (f *Tezos) recoverFFI(ctx context.Context, parsedMethod interface{}) (*fftypes.FFIMethod, []*fftypes.FFIError, error) {
	methodInfo, ok := parsedMethod.(*ffiMethodAndErrors)
	if !ok || methodInfo.method == nil {
		return nil, nil, i18n.NewError(ctx, coremsgs.MsgUnexpectedInterfaceType, parsedMethod)
	}
	return methodInfo.method, methodInfo.errors, nil
}

func (t *Tezos) invokeContractMethod(ctx context.Context, address, methodName string, signingKey, requestID string, michelsonInput micheline.Parameters, options map[string]interface{}) error {
	if t.metrics.IsMetricsEnabled() {
		t.metrics.BlockchainTransaction(address, methodName)
	}
	messageType := "SendTransaction"
	body, err := t.buildTezosconnectRequestBody(ctx, messageType, address, methodName, signingKey, requestID, michelsonInput, options)
	if err != nil {
		return err
	}

	var resErr tezosError
	res, err := t.client.R().
		SetContext(ctx).
		SetBody(body).
		SetError(&resErr).
		Post("/")
	if err != nil || !res.IsSuccess() {
		return wrapError(ctx, &resErr, res, err)
	}
	return nil
}

func (t *Tezos) buildTezosconnectRequestBody(ctx context.Context, messageType, address, methodName, signingKey, requestID string, michelsonInput micheline.Parameters, options map[string]interface{}) (map[string]interface{}, error) {
	headers := TezosconnectMessageHeaders{
		Type: messageType,
	}
	if requestID != "" {
		headers.ID = requestID
	}

	body := map[string]interface{}{
		"headers": &headers,
		"to":      address,
		"method":  methodName,
		"params":  []interface{}{michelsonInput},
	}
	if signingKey != "" {
		body["from"] = signingKey
	}

	return t.applyOptions(ctx, body, options)
}

func (t *Tezos) applyOptions(ctx context.Context, body, options map[string]interface{}) (map[string]interface{}, error) {
	for k, v := range options {
		// Set the new field if it's not already set. Do not allow overriding of existing fields
		if _, ok := body[k]; !ok {
			body[k] = v
		} else {
			return nil, i18n.NewError(ctx, coremsgs.MsgOverrideExistingFieldCustomOption, k)
		}
	}
	return body, nil
}

func (t *Tezos) parseContractLocation(ctx context.Context, location *fftypes.JSONAny) (*Location, error) {
	tezosLocation := Location{}
	if err := json.Unmarshal(location.Bytes(), &tezosLocation); err != nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgContractLocationInvalid, err)
	}
	if tezosLocation.Address == "" {
		return nil, i18n.NewError(ctx, coremsgs.MsgContractLocationInvalid, "'address' not set")
	}
	return &tezosLocation, nil
}

func (t *Tezos) encodeContractLocation(ctx context.Context, location *Location) (result *fftypes.JSONAny, err error) {
	location.Address, err = formatTezosAddress(ctx, location.Address)
	if err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(location)
	if err == nil {
		result = fftypes.JSONAnyPtrBytes(normalized)
	}
	return result, err
}

func wrapError(ctx context.Context, errRes *tezosError, res *resty.Response, err error) error {
	if errRes != nil && errRes.Error != "" {
		return i18n.WrapError(ctx, err, coremsgs.MsgTezosconnectRESTErr, errRes.Error)
	}
	return ffresty.WrapRestErr(ctx, res, err, coremsgs.MsgTezosconnectRESTErr)
}

func formatTezosAddress(ctx context.Context, key string) (string, error) {
	if addressVerify.MatchString(key) {
		return key, nil
	}
	return "", i18n.NewError(ctx, coremsgs.MsgInvalidTezosAddress)
}
