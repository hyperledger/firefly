// Copyright © 2025 Kaleido, Inc.
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

package paladin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly-signer/pkg/abi"
	"github.com/hyperledger/firefly-signer/pkg/ffi2abi"
	"github.com/hyperledger/firefly/internal/blockchain/common"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	pldconf "github.com/kaleido-io/paladin/config/pkg/pldconf"
	"github.com/kaleido-io/paladin/toolkit/pkg/pldapi"
	pldclient "github.com/kaleido-io/paladin/toolkit/pkg/pldclient"
	"github.com/kaleido-io/paladin/toolkit/pkg/rpcclient"
	"github.com/kaleido-io/paladin/toolkit/pkg/tktypes"
)

// TODO Breakdown
// Build RPC Client
// Create a receipts listener
type Paladin struct {
	ctx              context.Context
	cancelCtx        context.CancelFunc
	metrics          metrics.Manager
	capabilities     *blockchain.Capabilities
	callbacks        common.BlockchainCallbacks
	httpClientConfig pldconf.HTTPClientConfig
	httpClient       pldclient.PaladinClient
	wsClientConfig   pldconf.WSClientConfig
	wsconn           map[string]pldclient.PaladinWSClient
}

func (p *Paladin) Name() string {
	return "paladin"
}

// TODO what the hell is this?
func (t *Paladin) VerifierType() core.VerifierType {
	return core.VerifierTypeEthAddress
}

func (p *Paladin) Init(ctx context.Context, cancelCtx context.CancelFunc, config config.Section, metrics metrics.Manager, cacheManager cache.Manager) error {
	p.InitConfig(config)

	p.ctx = log.WithLogField(ctx, "proto", "ethereum")
	p.cancelCtx = cancelCtx
	p.metrics = metrics
	// TODO might be great to introduce this in the future
	p.capabilities = &blockchain.Capabilities{}
	// TODO probably need to register a new callback to confirm X amount of receipts
	p.callbacks = common.NewBlockchainCallbacks()

	// TODO Create a receipt listener here
	// Run an event loop to fetch receipts and update an operation

	// ptx_startReceiptListener

	// This is not great
	// - Convert from ffresty to Paladin HTTP Config
	// - Then Paladin just converts back to FF Resty
	clientConfig := config.SubSection(PaladinRPCClientConfigKey)

	httpRestyConfig, err := ffresty.GenerateConfig(ctx, clientConfig.SubSection(PaladinRPCHTTPClientConfigKey))
	if err != nil {
		return err
	}

	p.httpClientConfig = pldconf.HTTPClientConfig{
		URL:         httpRestyConfig.URL,
		HTTPHeaders: httpRestyConfig.HTTPHeaders,
		Auth: pldconf.HTTPBasicAuthConfig{
			Username: httpRestyConfig.AuthUsername,
			Password: httpRestyConfig.AuthPassword,
		},
	}

	httpClient, err := pldclient.New().HTTP(ctx, &p.httpClientConfig)
	if err != nil {
		return err
	}

	p.httpClient = httpClient

	wsRestyConfig, err := wsclient.GenerateConfig(ctx, clientConfig.SubSection(PaladinRPCWSClientConfigKey))
	if err != nil {
		return err
	}

	p.wsClientConfig = pldconf.WSClientConfig{
		HTTPClientConfig: pldconf.HTTPClientConfig{
			URL:         wsRestyConfig.WebSocketURL,
			HTTPHeaders: wsRestyConfig.HTTPHeaders,
			Auth: pldconf.HTTPBasicAuthConfig{
				Username: wsRestyConfig.AuthUsername,
				Password: wsRestyConfig.AuthPassword,
			},
		},
	}

	p.wsconn = make(map[string]pldclient.PaladinWSClient)

	return nil
}

func (p *Paladin) StartNamespace(ctx context.Context, namespace string) error {
	// Websocket Client per Namespace

	var err error
	// This connects the websocket
	p.wsconn[namespace], err = pldclient.New().WebSocket(ctx, &p.wsClientConfig)
	if err != nil {
		return err
	}

	// TODO Figure out this listener, make sure it's in place?
	listenerName := "publiclistener"
	sub, err := p.wsconn[namespace].PTX().SubscribeReceipts(ctx, listenerName)
	if err != nil {
		return err
	}

	// TODO: Figure out if this is the right context
	go p.batchEventLoop(namespace, sub)
	return nil
}

func (p *Paladin) batchEventLoop(namespace string, sub rpcclient.Subscription) {
	l := log.L(p.ctx).WithField("role", "event-loop").WithField("namespace", namespace)
	defer p.wsconn[namespace].Close()
	ctx := log.WithLogger(p.ctx, l)
	l.Debugf("Starting event loop for namespace '%s'", namespace)
	for {
		select {
		case <-ctx.Done():
			l.Debugf("Event loop exiting (context cancelled)")
			return
		case subNotification, ok := <-sub.Notifications():
			if !ok {
				l.Debugf("Event loop exiting (receive channel closed). Terminating server!")
				// This cancels the context of the whole plugin!
				p.cancelCtx()
				return
			}

			// Handle websocket event
			var batch pldapi.TransactionReceiptBatch
			json.Unmarshal(subNotification.Result, &batch)

			updates := []*core.OperationUpdate{}
			for _, receipt := range batch.Receipts {
				fmt.Printf("%v", receipt.TransactionReceipt.ID)
				// For now, we need to get the transaction to get the idempotency Key
				// Might be fixed through https://github.com/LF-Decentralized-Trust-labs/paladin/issues/551
				tx, err := p.httpClient.PTX().GetTransactionFull(ctx, receipt.TransactionReceipt.ID)
				if err != nil {
					// Figure out if the sub will resubmit events if not ack'd after certain or
					// We need to send a nack?
					l.Error(err)
					continue
				}
				var status core.OpStatus
				if receipt.TransactionReceipt.Success {
					status = core.OpStatusSucceeded
				} else {
					// TODO more thinking here
					status = core.OpStatusFailed
				}

				// The IdempotencyKey is the FireFly Operation ID
				nsOpID := tx.IdempotencyKey

				// We need to cross check and see if we actually care about this operation!
				// Not all transaction receipts in Paladin are for those that this FireFly has submitted!
				opNamespace, _, _ := core.ParseNamespacedOpID(ctx, nsOpID)
				if opNamespace != namespace {
					l.Debugf("Ignoring operation update for transaction request=%s tx=%s", nsOpID, receipt.TransactionHash)
					continue
				}

				output := map[string]interface{}{}
				if receipt.ContractAddress != nil {
					output["ContractAddress"] = receipt.ContractAddress
				}

				updates = append(updates, &core.OperationUpdate{
					Plugin:         p.Name(),
					NamespacedOpID: nsOpID,
					Status:         status,
					BlockchainTXID: receipt.TransactionHash.String(),
					ErrorMessage:   "", // TODO: Handle error message from Paladin
					Output:         output,
				})
			}

			if len(updates) > 0 {
				// We have a batch of operation updates
				err := p.callbacks.BulkOperationUpdates(ctx, namespace, updates)
				if err != nil {
					l.Errorf("Failed to commit batch updates: %s", err.Error())
					continue
				}
			}

			l.Debug("All events from websocket handled")

			// You only ever sending this ACK once we have committed all the DB writes
			// Which is done as part of BulkOperationUpdates
			err := subNotification.Ack(ctx)
			if err != nil {
				l.Errorf("Failed to ack batch receipt: %s", err.Error())
				// TODO: We want to do better here and hanlde the error correctly
				// a `nack`?
				// return err
				continue
			}
		}
	}
}

func (p *Paladin) StopNamespace(ctx context.Context, namespace string) error {
	wsconn, ok := p.wsconn[namespace]
	if ok {
		wsconn.Close()
	}
	delete(p.wsconn, namespace)

	return nil
}

func (p *Paladin) SetHandler(namespace string, handler blockchain.Callbacks) {
	p.callbacks.SetHandler(namespace, handler)
}

func (p *Paladin) SetOperationHandler(namespace string, handler core.OperationCallbacks) {
	p.callbacks.SetOperationalHandler(namespace, handler)
}

func (p *Paladin) Capabilities() *blockchain.Capabilities {
	return p.capabilities
}

func (p *Paladin) ResolveSigningKey(ctx context.Context, keyRef string, intent blockchain.ResolveKeyIntent) (string, error) {
	// Address always resolve for now, probably call Paladin to get the key is valid in the future?
	// keymgr_resolveEthAddress

	return keyRef, nil
}

func (p *Paladin) SubmitBatchPin(ctx context.Context, nsOpID, networkNamespace, signingKey string, batch *blockchain.BatchPin, location *fftypes.JSONAny) error {
	// Not applicable for now
	return nil
}

func (p *Paladin) SubmitNetworkAction(ctx context.Context, nsOpID, signingKey string, action core.NetworkActionType, location *fftypes.JSONAny) error {
	// Not applicable for now
	return nil
}

func (p *Paladin) DeployContract(ctx context.Context, nsOpID, signingKey string, definition, contract *fftypes.JSONAny, input []interface{}, options map[string]interface{}) (submissionRejected bool, err error) {
	if p.metrics.IsMetricsEnabled() {
		p.metrics.BlockchainContractDeployment()
	}

	bytecode, err := tktypes.ParseHexBytes(ctx, contract.AsString())

	if err != nil {
		return true, err
	}

	// Parse the ABI
	var a *abi.ABI
	err = json.Unmarshal(definition.Bytes(), &a)
	if err != nil {
		// Handle this error better
		return true, err
	}

	tx := &pldapi.TransactionInput{
		TransactionBase: pldapi.TransactionBase{
			IdempotencyKey: nsOpID,
			Type:           pldapi.TransactionTypePublic.Enum(),
			From:           signingKey,
		},
		ABI:      *a,
		Bytecode: bytecode,
	}

	// TODO solve this
	_, err = p.httpClient.PTX().SendTransaction(ctx, tx)

	return false, err
}

func (p *Paladin) ValidateInvokeRequest(ctx context.Context, parsedMethod interface{}, input map[string]interface{}, hasMessage bool) error {
	// No op
	// TODO implement this!
	return nil
}

type Location struct {
	Address string `json:"address"`
}
type parsedFFIMethod struct {
	methodABI *abi.Entry
	errorsABI []*abi.Entry
}

func (p *Paladin) prepareRequest(ctx context.Context, parsedMethod interface{}, input map[string]interface{}) (*parsedFFIMethod, []interface{}, error) {
	methodInfo, ok := parsedMethod.(*parsedFFIMethod)
	if !ok || methodInfo.methodABI == nil {
		return nil, nil, i18n.NewError(ctx, coremsgs.MsgUnexpectedInterfaceType, parsedMethod)
	}
	inputs := methodInfo.methodABI.Inputs
	orderedInput := make([]interface{}, len(inputs))
	for i, param := range inputs {
		orderedInput[i] = input[param.Name]
	}
	return methodInfo, orderedInput, nil
}

func (p *Paladin) InvokeContract(ctx context.Context, nsOpID, signingKey string, location *fftypes.JSONAny, parsedMethod interface{}, input map[string]interface{}, options map[string]interface{}, batch *blockchain.BatchPin) (submissionRejected bool, err error) {

	// TODO we want to decode and define a format for this
	to, err := tktypes.ParseEthAddress(location.AsString())
	if err != nil {
		return true, err
	}

	// Needs cleaning up
	methodInfo, orderedInput, err := p.prepareRequest(ctx, parsedMethod, input)
	if err != nil {
		return true, err
	}

	// Parse the ABI
	var a abi.ABI = []*abi.Entry{}
	a = append(a, methodInfo.methodABI)

	// if p.metrics.IsMetricsEnabled() {
	// 	p.metrics.BlockchainTransaction(to.String(), a.Constructor().Name)
	// }

	tx := &pldapi.TransactionInput{
		TransactionBase: pldapi.TransactionBase{
			IdempotencyKey: nsOpID,
			Type:           pldapi.TransactionTypePublic.Enum(),
			From:           signingKey,
			To:             to,
			Data:           tktypes.JSONString(orderedInput),
		},
		ABI: a,
	}

	// TODO solve this
	_, err = p.httpClient.PTX().SendTransaction(ctx, tx)

	return false, err
}

func (p *Paladin) QueryContract(ctx context.Context, signingKey string, location *fftypes.JSONAny, parsedMethod interface{}, input map[string]interface{}, options map[string]interface{}) (interface{}, error) {
	// No op
	return nil, nil
}

func (p *Paladin) AddContractListener(ctx context.Context, subscription *core.ContractListener, lastProtocolID string) error {
	// Not support for now
	return nil
}

func (p *Paladin) DeleteContractListener(ctx context.Context, subscription *core.ContractListener, okNotFound bool) error {
	// Not support for now
	return nil
}

func (p *Paladin) GetContractListenerStatus(ctx context.Context, namespace, subID string, okNotFound bool) (bool, interface{}, core.ContractListenerStatus, error) {
	// Not support for now
	return false, nil, "", nil
}

func (p *Paladin) GetFFIParamValidator(ctx context.Context) (fftypes.FFIParamValidator, error) {
	// TODO
	return nil, nil
}

type FFIGenerationInput struct {
	ABI *abi.ABI `json:"abi,omitempty"`
}

func (p *Paladin) GenerateFFI(ctx context.Context, generationRequest *fftypes.FFIGenerationRequest) (*fftypes.FFI, error) {
	var input FFIGenerationInput
	err := json.Unmarshal(generationRequest.Input.Bytes(), &input)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, coremsgs.MsgFFIGenerationFailed, "unable to deserialize JSON as ABI")
	}
	if input.ABI == nil || len(*input.ABI) == 0 {
		return nil, i18n.NewError(ctx, coremsgs.MsgFFIGenerationFailed, "ABI is empty")
	}
	return ffi2abi.ConvertABIToFFI(ctx, generationRequest.Namespace, generationRequest.Name, generationRequest.Version, generationRequest.Description, input.ABI)
}

func (p *Paladin) NormalizeContractLocation(ctx context.Context, ntype blockchain.NormalizeType, location *fftypes.JSONAny) (*fftypes.JSONAny, error) {
	// TODO
	// Figure out contract location of Paladin
	return nil, nil
}

func (p *Paladin) GenerateEventSignature(ctx context.Context, event *fftypes.FFIEventDefinition) (string, error) {
	return "", nil
}

func (p *Paladin) GenerateEventSignatureWithLocation(ctx context.Context, event *fftypes.FFIEventDefinition, location *fftypes.JSONAny) (string, error) {
	return "", nil
}

func (p *Paladin) CheckOverlappingLocations(ctx context.Context, left *fftypes.JSONAny, right *fftypes.JSONAny) (bool, error) {
	return false, nil
}

func (p *Paladin) GenerateErrorSignature(ctx context.Context, errorDef *fftypes.FFIErrorDefinition) string {
	return ""
}

func (p *Paladin) GetNetworkVersion(ctx context.Context, location *fftypes.JSONAny) (int, error) {
	return 0, nil
}

func (p *Paladin) GetAndConvertDeprecatedContractConfig(ctx context.Context) (location *fftypes.JSONAny, fromBlock string, err error) {
	return nil, "", nil
}

func (p *Paladin) AddFireflySubscription(ctx context.Context, namespace *core.Namespace, contract *blockchain.MultipartyContract, lastProtocolID string) (subID string, err error) {
	return "", nil
}

func (p *Paladin) RemoveFireflySubscription(ctx context.Context, subID string) {}

func (p *Paladin) GetTransactionStatus(ctx context.Context, operation *core.Operation) (interface{}, error) {
	// TODO just an RPC CALL to get status but the mapping between operation and TX is the difficult part here...
	return nil, nil
}

func (p *Paladin) ParseInterface(ctx context.Context, method *fftypes.FFIMethod, errors []*fftypes.FFIError) (interface{}, error) {
	methodABI, err := ffi2abi.ConvertFFIMethodToABI(ctx, method)
	if err != nil {
		return nil, err
	}
	methodInfo := &parsedFFIMethod{
		methodABI: methodABI,
		errorsABI: make([]*abi.Entry, len(errors)),
	}
	for i, ffiError := range errors {
		errorABI, err := ffi2abi.ConvertFFIErrorDefinitionToABI(ctx, &ffiError.FFIErrorDefinition)
		if err != nil {
			return nil, err
		}
		methodInfo.errorsABI[i] = errorABI
	}
	return methodInfo, nil
}
