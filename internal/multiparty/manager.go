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
	"sync"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
)

type Manager interface {
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
	SubmitBatchPin(ctx context.Context, nsOpID string, signingKey string, batch *blockchain.BatchPin) error

	// SubmitNetworkAction writes a special "BatchPin" event which signals the plugin to take an action
	SubmitNetworkAction(ctx context.Context, nsOpID string, signingKey string, action core.NetworkActionType) error

	core.Named
}

type Contract struct {
	Location   *fftypes.JSONAny
	FirstEvent string
}

type multipartyManager struct {
	ctx            context.Context
	blockchain     blockchain.Plugin
	contracts      []Contract
	namespace      string
	activeContract struct {
		location       *fftypes.JSONAny
		mux            sync.Mutex
		firstEvent     string
		networkVersion int
		subscription   string
	}
}

func NewMultipartyManager(ctx context.Context, ns string, contracts []Contract, bi blockchain.Plugin) Manager {
	return &multipartyManager{
		ctx:        ctx,
		namespace:  ns,
		contracts:  contracts,
		blockchain: bi,
	}
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
		if err == nil {
			mm.activeContract.mux.Lock()
			mm.activeContract.location = location
			mm.activeContract.firstEvent = firstEvent
			mm.activeContract.networkVersion = version
			mm.activeContract.subscription = subID
			mm.activeContract.mux.Unlock()
			contracts.Active.Info = fftypes.JSONObject{
				"location":     location,
				"firstEvent":   firstEvent,
				"subscription": subID,
			}
		}
	}
	return err
}

func (mm *multipartyManager) resolveFireFlyContract(ctx context.Context, contractIndex int) (location *fftypes.JSONAny, firstEvent string, err error) {
	if len(mm.contracts) > 0 {
		if contractIndex >= len(mm.contracts) {
			return nil, "", i18n.NewError(ctx, coremsgs.MsgInvalidFireFlyContractIndex, fmt.Sprintf("%s.multiparty.contracts[%d]", mm.namespace, contractIndex))
		}
		active := mm.contracts[contractIndex]
		location = active.Location
		firstEvent = active.FirstEvent
	} else {
		// handle deprecated config here
		location, firstEvent, err = mm.blockchain.GetAndConvertDeprecatedContractConfig(ctx)
		if err != nil {
			return nil, "", err
		}

		switch firstEvent {
		case string(core.SubOptsFirstEventOldest):
			firstEvent = "0"
		case string(core.SubOptsFirstEventNewest):
			firstEvent = "latest"
		}
	}

	return location, firstEvent, err
}

func (mm *multipartyManager) NetworkVersion() int {
	mm.activeContract.mux.Lock()
	defer mm.activeContract.mux.Unlock()
	return mm.activeContract.networkVersion
}

func (mm *multipartyManager) TerminateContract(ctx context.Context, contracts *core.FireFlyContracts, termination *blockchain.Event) (err error) {
	// possibly have DB transactions here instead of somewhere else
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
	mm.activeContract.mux.Lock()
	defer mm.activeContract.mux.Unlock()
	return mm.activeContract.networkVersion
}

func (mm *multipartyManager) SubmitNetworkAction(ctx context.Context, nsOpID string, signingKey string, action core.NetworkActionType) error {
	return mm.blockchain.SubmitNetworkAction(ctx, nsOpID, signingKey, action, mm.activeContract.location)
}

func (mm *multipartyManager) SubmitBatchPin(ctx context.Context, nsOpID string, signingKey string, batch *blockchain.BatchPin) error {
	return mm.blockchain.SubmitBatchPin(ctx, nsOpID, signingKey, batch, mm.activeContract.location)
}

func (mm *multipartyManager) Name() string {
	return mm.blockchain.Name()
}
