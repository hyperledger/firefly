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
	"regexp"

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

type Location struct {
	Address string `json:"address"`
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
	// TODO: impl
	return nil
}

func (t *Tezos) ValidateInvokeRequest(ctx context.Context, method *fftypes.FFIMethod, input map[string]interface{}, errors []*fftypes.FFIError, hasMessage bool) error {
	// TODO: impl
	return nil
}

func (t *Tezos) InvokeContract(ctx context.Context, nsOpID string, signingKey string, location *fftypes.JSONAny, method *fftypes.FFIMethod, input map[string]interface{}, errors []*fftypes.FFIError, options map[string]interface{}, batch *blockchain.BatchPin) error {
	// TODO: impl
	return nil
}

func (t *Tezos) QueryContract(ctx context.Context, signingKey string, location *fftypes.JSONAny, method *fftypes.FFIMethod, input map[string]interface{}, errors []*fftypes.FFIError, options map[string]interface{}) (interface{}, error) {
	// TODO: impl
	return nil, nil
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

func formatTezosAddress(ctx context.Context, key string) (string, error) {
	if addressVerify.MatchString(key) {
		return key, nil
	}
	return "", i18n.NewError(ctx, coremsgs.MsgInvalidTezosAddress)
}
