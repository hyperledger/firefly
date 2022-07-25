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
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

type Manager interface {
	core.Named

	// RootOrg returns configuration details for the root organization identity
	RootOrg() RootOrg

	// ConfigureContract initializes the subscription to the FireFly contract
	// - Checks the namespace contract info against the plugin's configuration, and updates it as needed
	// - Initializes the contract info for performing BatchPin transactions, and initializes subscriptions for BatchPin events
	ConfigureContract(ctx context.Context) (err error)

	// TerminateContract marks the given event as the last one to be parsed on the current FireFly contract
	// - Validates that the event came from the currently active FireFly contract
	// - Re-initializes the plugin against the next configured FireFly contract
	// - Updates the namespace contract info to record the point of termination and the newly active contract
	TerminateContract(ctx context.Context, location *fftypes.JSONAny, termination *blockchain.Event) (err error)

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
	namespace  *core.Namespace
	database   database.Plugin
	blockchain blockchain.Plugin
	operations operations.Manager
	metrics    metrics.Manager
	txHelper   txcommon.Helper
	config     Config
}

func NewMultipartyManager(ctx context.Context, ns *core.Namespace, config Config, di database.Plugin, bi blockchain.Plugin, om operations.Manager, mm metrics.Manager, th txcommon.Helper) (Manager, error) {
	if di == nil || bi == nil || mm == nil || om == nil || th == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "MultipartyManager")
	}
	mp := &multipartyManager{
		namespace:  ns,
		config:     config,
		database:   di,
		blockchain: bi,
		operations: om,
		metrics:    mm,
		txHelper:   th,
	}
	om.RegisterHandler(ctx, mp, []core.OpType{
		core.OpTypeBlockchainPinBatch,
		core.OpTypeBlockchainNetworkAction,
	})
	return mp, nil
}

func (mm *multipartyManager) Name() string {
	return "MultipartyManager"
}

func (mm *multipartyManager) RootOrg() RootOrg {
	return mm.config.Org
}

func (mm *multipartyManager) ConfigureContract(ctx context.Context) (err error) {
	contracts := &mm.namespace.Contracts
	log.L(ctx).Infof("Resolving FireFly contract at index %d", contracts.Active.Index)
	location, firstEvent, err := mm.resolveFireFlyContract(ctx, contracts.Active.Index)
	if err != nil {
		return err
	}

	version, err := mm.blockchain.GetNetworkVersion(ctx, location)
	if err != nil {
		return err
	}

	if !contracts.Active.Location.IsNil() && contracts.Active.Location.String() != location.String() {
		log.L(ctx).Warnf("FireFly contract location changed from %s to %s", contracts.Active.Location, location)
	}

	subID, err := mm.blockchain.AddFireflySubscription(ctx, mm.namespace.LocalName, location, firstEvent)
	if err == nil {
		contracts.Active = core.MultipartyContract{
			Location:   location,
			FirstEvent: firstEvent,
			Info: core.MultipartyContractInfo{
				Subscription: subID,
				Version:      version,
			},
		}
		err = mm.database.UpsertNamespace(ctx, mm.namespace, true)
	}
	return err
}

func (mm *multipartyManager) resolveFireFlyContract(ctx context.Context, contractIndex int) (location *fftypes.JSONAny, firstEvent string, err error) {
	if len(mm.config.Contracts) > 0 || contractIndex > 0 {
		if contractIndex >= len(mm.config.Contracts) {
			return nil, "", i18n.NewError(ctx, coremsgs.MsgInvalidFireFlyContractIndex,
				fmt.Sprintf("%s.multiparty.contracts[%d]", mm.namespace.LocalName, contractIndex))
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

func (mm *multipartyManager) TerminateContract(ctx context.Context, location *fftypes.JSONAny, termination *blockchain.Event) (err error) {
	contracts := &mm.namespace.Contracts
	if contracts.Active.Location.String() != location.String() {
		log.L(ctx).Warnf("Ignoring termination event from contract at '%s', which does not match active '%s'", location, contracts.Active.Location)
		return nil
	}
	log.L(ctx).Infof("Processing termination of contract #%d at '%s'", contracts.Active.Index, contracts.Active.Location)
	mm.blockchain.RemoveFireflySubscription(ctx, contracts.Active.Info.Subscription)
	contracts.Active.Info.FinalEvent = termination.ProtocolID
	contracts.Terminated = append(contracts.Terminated, contracts.Active)
	contracts.Active = core.MultipartyContract{Index: contracts.Active.Index + 1}
	return mm.ConfigureContract(ctx)
}

func (mm *multipartyManager) GetNetworkVersion() int {
	return mm.namespace.Contracts.Active.Info.Version
}

func (mm *multipartyManager) SubmitNetworkAction(ctx context.Context, signingKey string, action *core.NetworkAction) error {
	if action.Type != core.NetworkActionTerminate {
		return i18n.NewError(ctx, coremsgs.MsgUnrecognizedNetworkAction, action.Type)
	}

	txid, err := mm.txHelper.SubmitNewTransaction(ctx, core.TransactionTypeNetworkAction)
	if err != nil {
		return err
	}

	op := core.NewOperation(
		mm.blockchain,
		mm.namespace.LocalName,
		txid,
		core.OpTypeBlockchainNetworkAction)
	addNetworkActionInputs(op, action.Type, signingKey)
	if err := mm.operations.AddOrReuseOperation(ctx, op); err != nil {
		return err
	}

	_, err = mm.operations.RunOperation(ctx, opNetworkAction(op, action.Type, signingKey))
	return err
}

func (mm *multipartyManager) SubmitBatchPin(ctx context.Context, batch *core.BatchPersisted, contexts []*fftypes.Bytes32, payloadRef string) error {
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
