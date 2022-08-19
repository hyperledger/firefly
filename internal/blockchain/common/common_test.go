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

package common

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/coremocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCallbackOperationUpdate(t *testing.T) {
	nsOpID := "ns1:" + fftypes.NewUUID().String()

	mbi := &blockchainmocks.Plugin{}
	mcb := &coremocks.OperationCallbacks{}
	cb := NewBlockchainCallbacks()
	cb.SetOperationalHandler("ns1", mcb)

	mbi.On("Name").Return("utblockchain")
	mcb.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == nsOpID &&
			update.Status == core.OpStatusSucceeded &&
			update.BlockchainTXID == "tx1" &&
			update.ErrorMessage == "err" &&
			update.Plugin == "utblockchain"
	})).Return().Once()
	cb.OperationUpdate(context.Background(), mbi, nsOpID, core.OpStatusSucceeded, "tx1", "err", fftypes.JSONObject{})

	nsOpID = "ns2:" + fftypes.NewUUID().String()
	cb.OperationUpdate(context.Background(), mbi, nsOpID, core.OpStatusSucceeded, "tx1", "err", fftypes.JSONObject{})

	mcb.AssertExpectations(t)
}

func TestCallbackBlockchainEvent(t *testing.T) {
	event := &blockchain.EventWithSubscription{}

	mcb := &blockchainmocks.Callbacks{}
	cb := NewBlockchainCallbacks()
	cb.SetHandler("ns1", mcb)

	mcb.On("BlockchainEvent", event).Return(nil).Once()
	err := cb.BlockchainEvent(context.Background(), "ns1", event)
	assert.NoError(t, err)

	err = cb.BlockchainEvent(context.Background(), "ns2", event)
	assert.NoError(t, err)

	mcb.On("BlockchainEvent", event).Return(fmt.Errorf("pop")).Once()
	err = cb.BlockchainEvent(context.Background(), "", event)
	assert.EqualError(t, err, "pop")

	mcb.AssertExpectations(t)
}

func TestCallbackBatchPinBadBatch(t *testing.T) {
	event := &blockchain.Event{}
	verifier := &core.VerifierRef{}
	params := &BatchPinParams{}

	mcb := &blockchainmocks.Callbacks{}
	cb := NewBlockchainCallbacks()
	cb.SetHandler("ns1", mcb)

	sub := &SubscriptionInfo{
		Version:     2,
		V2Namespace: "ns1",
	}
	err := cb.BatchPinOrNetworkAction(context.Background(), "ns1", sub, fftypes.JSONAnyPtr("{}"), event, verifier, params)
	assert.NoError(t, err)

	mcb.AssertExpectations(t)
}

func TestCallbackBatchPin(t *testing.T) {
	event := &blockchain.Event{}
	verifier := &core.VerifierRef{}
	params := &BatchPinParams{
		UUIDs:     "0xe19af8b390604051812d7597d19adfb9847d3bfd074249efb65d3fed15f5b0a6",
		BatchHash: "0xd71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be",
		Contexts: []string{
			"0x68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a",
			"0x19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771",
		},
		PayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
	}

	mcb := &blockchainmocks.Callbacks{}
	cb := NewBlockchainCallbacks()
	cb.SetHandler("ns1", mcb)

	sub := &SubscriptionInfo{
		Version:     2,
		V2Namespace: "ns1",
	}
	mcb.On("BatchPinComplete", "ns1", mock.Anything, verifier).Return(nil).Once()
	err := cb.BatchPinOrNetworkAction(context.Background(), "", sub, fftypes.JSONAnyPtr("{}"), event, verifier, params)
	assert.NoError(t, err)

	mcb.On("BatchPinComplete", "ns1", mock.Anything, verifier).Return(fmt.Errorf("pop")).Once()
	err = cb.BatchPinOrNetworkAction(context.Background(), "", sub, fftypes.JSONAnyPtr("{}"), event, verifier, params)
	assert.EqualError(t, err, "pop")

	sub = &SubscriptionInfo{
		Version:     1,
		V1Namespace: map[string][]string{"ns2": {"ns1", "ns"}},
	}
	mcb.On("BatchPinComplete", "ns1", mock.Anything, verifier).Return(nil).Once()
	err = cb.BatchPinOrNetworkAction(context.Background(), "ns2", sub, fftypes.JSONAnyPtr("{}"), event, verifier, params)
	assert.NoError(t, err)

	err = cb.BatchPinOrNetworkAction(context.Background(), "ns3", sub, fftypes.JSONAnyPtr("{}"), event, verifier, params)
	assert.NoError(t, err)

	mcb.AssertExpectations(t)
}

func TestCallbackNetworkAction(t *testing.T) {
	event := &blockchain.Event{}
	verifier := &core.VerifierRef{}
	params := &BatchPinParams{}

	mcb := &blockchainmocks.Callbacks{}
	cb := NewBlockchainCallbacks()
	cb.SetHandler("ns1", mcb)

	sub := &SubscriptionInfo{
		Version:     2,
		V2Namespace: "ns1",
	}
	mcb.On("BlockchainNetworkAction", "terminate", mock.Anything, mock.Anything, verifier).Return(nil).Once()
	err := cb.BatchPinOrNetworkAction(context.Background(), "firefly:terminate", sub, fftypes.JSONAnyPtr("{}"), event, verifier, params)
	assert.NoError(t, err)

	mcb.On("BlockchainNetworkAction", "terminate", mock.Anything, mock.Anything, verifier).Return(fmt.Errorf("pop")).Once()
	err = cb.BatchPinOrNetworkAction(context.Background(), "firefly:terminate", sub, fftypes.JSONAnyPtr("{}"), event, verifier, params)
	assert.EqualError(t, err, "pop")

	sub = &SubscriptionInfo{
		Version:     1,
		V1Namespace: map[string][]string{"ns2": {"ns1", "ns"}},
	}
	mcb.On("BlockchainNetworkAction", "terminate", mock.Anything, mock.Anything, verifier).Return(nil).Once()
	err = cb.BatchPinOrNetworkAction(context.Background(), "firefly:terminate", sub, fftypes.JSONAnyPtr("{}"), event, verifier, params)
	assert.NoError(t, err)

	mcb.AssertExpectations(t)
}

func TestBuildBatchPinErrors(t *testing.T) {
	event := &blockchain.Event{}
	params := &BatchPinParams{}

	_, err := buildBatchPin(context.Background(), event, params)
	assert.Regexp(t, "missing data", err)

	params.BatchHash = "0xd71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be"
	params.UUIDs = "BAD"
	_, err = buildBatchPin(context.Background(), event, params)
	assert.Regexp(t, "bad uuids", err)

	params.BatchHash = "BAD"
	params.UUIDs = "0xe19af8b390604051812d7597d19adfb9847d3bfd074249efb65d3fed15f5b0a6"
	_, err = buildBatchPin(context.Background(), event, params)
	assert.Regexp(t, "odd length hex string", err)

	params.BatchHash = "0xd71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be"
	params.Contexts = append(params.Contexts, "BAD")
	_, err = buildBatchPin(context.Background(), event, params)
	assert.Regexp(t, "odd length hex string", err)
}

func TestGetNamespaceFromSubName(t *testing.T) {
	ns := GetNamespaceFromSubName("ff-sub-ns1-123")
	assert.Equal(t, "ns1", ns)

	ns = GetNamespaceFromSubName("BAD")
	assert.Equal(t, "", ns)
}

func TestSubscriptionsAddRemoveSubscription(t *testing.T) {
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	subs := NewFireflySubscriptions()

	subs.AddSubscription(context.Background(), ns, 2, "sub1", nil)
	assert.NotNil(t, subs.GetSubscription("sub1"))
	subs.RemoveSubscription(context.Background(), "sub1")
	assert.Nil(t, subs.GetSubscription("sub1"))
}

func TestSubscriptionsAddRemoveSubscriptionV1(t *testing.T) {
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	subs := NewFireflySubscriptions()

	subs.AddSubscription(context.Background(), ns, 1, "sub1", nil)
	assert.NotNil(t, subs.GetSubscription("sub1"))
	subs.RemoveSubscription(context.Background(), "sub1")
	assert.Nil(t, subs.GetSubscription("sub1"))
}

func TestSubscriptionsRemoveInvalid(t *testing.T) {
	subs := NewFireflySubscriptions()
	subs.RemoveSubscription(context.Background(), "sub1")
}
