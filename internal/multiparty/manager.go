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

package multiparty

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

type Manager interface {
	core.Named

	// RootOrg returns configuration details for the root organization identity
	RootOrg() RootOrg

	// ConfigureContract initializes the subscription to the FireFly contract
	// - Checks the provided contract info against the plugin's configuration, and updates it as needed
	// - Initializes the contract info for performing BatchPin transactions, and initializes subscriptions for BatchPin events
	ConfigureContract(ctx context.Context, contracts *core.FireFlyContracts) (err error)

	// TerminateContract marks the given event as the last one to be parsed on the current FireFly contract
	// - Validates that the event came from the currently active FireFly contract
	// - Re-initializes the plugin against the next configured FireFly contract
	// - Updates the provided contract info to record the point of termination and the newly active contract
	TerminateContract(ctx context.Context, contracts *core.FireFlyContracts, termination *blockchain.Event) (err error)

	// GetNetworkVersion returns the network version of the active FireFly contract
	GetNetworkVersion() int

	// SubmitBatchPin sequences a batch of message globally to all viewers of a given ledger
	SubmitBatchPin(ctx context.Context, batch *core.BatchPersisted, contexts []*fftypes.Bytes32, payloadRef string) error

	// SubmitNetworkAction writes a special "BatchPin" event which signals the plugin to take an action
	SubmitNetworkAction(ctx context.Context, signingKey string, action *core.NetworkAction) error

	// From operations.OperationHandler
	PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error)
	RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error)
}

type Config struct {
	Enabled   bool
	Org       RootOrg
	Contracts []Contract
}

type RootOrg struct {
	Name        string
	Description string
	Key         string
}

type Contract struct {
	Location   *fftypes.JSONAny
	FirstEvent string
}

type multipartyManager struct {
	ctx            context.Context
	namespace      string
	database       database.Plugin
	blockchain     blockchain.Plugin
	operations     operations.Manager
	metrics        metrics.Manager
	config         Config
	activeContract struct {
		location       *fftypes.JSONAny
		firstEvent     string
		networkVersion int
		subscription   string
	}
}

func NewMultipartyManager(ctx context.Context, ns string, config Config, di database.Plugin, bi blockchain.Plugin, om operations.Manager, mm metrics.Manager) (Manager, error) {
	if di == nil || bi == nil || mm == nil || om == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "MultipartyManager")
	}
	mp := &multipartyManager{
		ctx:        ctx,
		namespace:  ns,
		config:     config,
		database:   di,
		blockchain: bi,
		operations: om,
		metrics:    mm,
	}
	om.RegisterHandler(ctx, mp, []core.OpType{
		core.OpTypeBlockchainPinBatch,
	})
	return mp, nil
}

func (mm *multipartyManager) Name() string {
	return "MultipartyManager"
}

func (mm *multipartyManager) RootOrg() RootOrg {
	return mm.config.Org
}

func (mm *multipartyManager) ConfigureContract(ctx context.Context, contracts *core.FireFlyContracts) (err error) {
	log.L(ctx).Infof("Resolving FireFly contract at index %d", contracts.Active.Index)
	location, firstEvent, err := mm.resolveFireFlyContract(ctx, contracts.Active.Index)
	if err != nil {
		return err
	}
	version, err := mm.blockchain.GetNetworkVersion(ctx, location)
	if err != nil {
		return err
	}

	subID, err := mm.blockchain.AddFireflySubscription(ctx, mm.namespace, location, firstEvent)
	if err == nil {
		mm.activeContract.location = location
		mm.activeContract.firstEvent = firstEvent
		mm.activeContract.networkVersion = version
		mm.activeContract.subscription = subID
		contracts.Active = core.FireFlyContractInfo{
			Location:     location,
			FirstEvent:   firstEvent,
			Subscription: subID,
		}
	}
	return err
}

func (mm *multipartyManager) resolveFireFlyContract(ctx context.Context, contractIndex int) (location *fftypes.JSONAny, firstEvent string, err error) {
	if len(mm.config.Contracts) > 0 {
		if contractIndex >= len(mm.config.Contracts) {
			return nil, "", i18n.NewError(ctx, coremsgs.MsgInvalidFireFlyContractIndex, fmt.Sprintf("%s.multiparty.contracts[%d]", mm.namespace, contractIndex))
		}
		active := mm.config.Contracts[contractIndex]
		location = active.Location
		firstEvent = active.FirstEvent
	} else {
		// handle deprecated config here
		location, firstEvent, err = mm.blockchain.GetAndConvertDeprecatedContractConfig(ctx)
		if err != nil {
			return nil, "", err
		}
	}

	return location, firstEvent, err
}

func (mm *multipartyManager) TerminateContract(ctx context.Context, contracts *core.FireFlyContracts, termination *blockchain.Event) (err error) {
	// TODO: Investigate if it better to consolidate DB termination here
	log.L(ctx).Infof("Processing termination of contract at index %d", contracts.Active.Index)
	contracts.Active.FinalEvent = termination.ProtocolID
	contracts.Terminated = append(contracts.Terminated, contracts.Active)
	contracts.Active = core.FireFlyContractInfo{Index: contracts.Active.Index + 1}
	err = mm.blockchain.RemoveFireflySubscription(ctx, mm.activeContract.subscription)
	if err != nil {
		return err
	}
	return mm.ConfigureContract(ctx, contracts)
}

func (mm *multipartyManager) GetNetworkVersion() int {
	return mm.activeContract.networkVersion
}

func (mm *multipartyManager) SubmitNetworkAction(ctx context.Context, signingKey string, action *core.NetworkAction) error {
	if action.Type == core.NetworkActionTerminate {
		if mm.namespace != core.LegacySystemNamespace {
			// For now, "terminate" only works on ff_system
			return i18n.NewError(ctx, coremsgs.MsgTerminateNotSupported, mm.namespace)
		}
	} else {
		return i18n.NewError(ctx, coremsgs.MsgUnrecognizedNetworkAction, action.Type)
	}
	// TODO: This should be a new operation type
	po := &core.PreparedOperation{
		Namespace: mm.namespace,
		ID:        fftypes.NewUUID(),
	}
	return mm.blockchain.SubmitNetworkAction(ctx, po.NamespacedIDString(), signingKey, action.Type, mm.activeContract.location)
}

func (mm *multipartyManager) SubmitBatchPin(ctx context.Context, batch *core.BatchPersisted, contexts []*fftypes.Bytes32, payloadRef string) error {
	// The pending blockchain transaction
	op := core.NewOperation(
		mm.blockchain,
		batch.Namespace,
		batch.TX.ID,
		core.OpTypeBlockchainPinBatch)
	addBatchPinInputs(op, batch.ID, contexts, payloadRef)
	if err := mm.operations.AddOrReuseOperation(ctx, op); err != nil {
		return err
	}

	if mm.metrics.IsMetricsEnabled() {
		mm.metrics.CountBatchPin()
	}
	_, err := mm.operations.RunOperation(ctx, opBatchPin(op, batch, contexts, payloadRef))
	return err
}
