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

package tokens

import (
	"context"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

// Plugin is the interface implemented by each tokens plugin
type Plugin interface {
	fftypes.Named

	// InitPrefix initializes the set of configuration options that are valid, with defaults. Called on all plugins.
	InitPrefix(prefix config.PrefixArray)

	// Init initializes the plugin, with configuration
	// Returns the supported featureset of the interface
	Init(ctx context.Context, name string, prefix config.Prefix, callbacks Callbacks) error

	// Blockchain interface must not deliver any events until start is called
	Start() error

	// Capabilities returns capabilities - not called until after Init
	Capabilities() *Capabilities

	// CreateTokenPool creates a new (fungible or non-fungible) pool of tokens
	CreateTokenPool(ctx context.Context, operationID *fftypes.UUID, pool *fftypes.TokenPool) error

	// ActivateTokenPool activates a pool in order to begin receiving events
	ActivateTokenPool(ctx context.Context, operationID *fftypes.UUID, pool *fftypes.TokenPool, event *fftypes.BlockchainEvent) error

	// MintTokens mints new tokens in a pool and adds them to the recipient's account
	MintTokens(ctx context.Context, operationID *fftypes.UUID, poolProtocolID string, mint *fftypes.TokenTransfer) error

	// BurnTokens burns tokens from an account
	BurnTokens(ctx context.Context, operationID *fftypes.UUID, poolProtocolID string, burn *fftypes.TokenTransfer) error

	// TransferTokens transfers tokens within a pool from one account to another
	TransferTokens(ctx context.Context, operationID *fftypes.UUID, poolProtocolID string, transfer *fftypes.TokenTransfer) error
}

// Callbacks is the interface provided to the tokens plugin, to allow it to pass events back to firefly.
//
// Events must be delivered sequentially, such that event 2 is not delivered until the callback invoked for event 1
// has completed. However, it does not matter if these events are workload balance between the firefly core
// cluster instances of the node.
type Callbacks interface {
	// TokensOpUpdate notifies firefly of an update to this plugin's operation within a transaction.
	// Only success/failure and errorMessage (for errors) are modeled.
	// opOutput can be used to add opaque protocol specific JSON from the plugin (protocol transaction ID etc.)
	// Note this is an optional hook information, and stored separately to the confirmation of the actual event that was being submitted/sequenced.
	// Only the party submitting the transaction will see this data.
	//
	// Error should will only be returned in shutdown scenarios
	TokenOpUpdate(plugin Plugin, operationID *fftypes.UUID, txState fftypes.OpStatus, errorMessage string, opOutput fftypes.JSONObject) error

	// TokenPoolCreated notifies on the creation of a new token pool, which might have been
	// submitted by us, or by any other authorized party in the network.
	//
	// Error should will only be returned in shutdown scenarios
	TokenPoolCreated(plugin Plugin, pool *TokenPool) error

	// TokensTransferred notifies on a transfer between token accounts.
	//
	// Error should will only be returned in shutdown scenarios
	TokensTransferred(plugin Plugin, transfer *TokenTransfer) error
}

// Capabilities the supported featureset of the tokens
// interface implemented by the plugin, with the specified config
type Capabilities struct {
}

// TokenPool is the set of data returned from the connector when a token pool is created.
type TokenPool struct {
	// Type is the type of tokens (fungible, non-fungible, etc) in this pool
	Type fftypes.TokenType

	// ProtocolID is the ID assigned to this pool by the connector (must be unique for this connector)
	ProtocolID string

	// TransactionID is the FireFly-assigned ID to correlate this to a transaction (optional)
	// Not guaranteed to be set for pool creation events triggered outside of FireFly
	TransactionID *fftypes.UUID

	// Key is the chain-specific identifier for the user that created the pool
	Key string

	// Connector is the configured name of this connector
	Connector string

	// Standard is the well-defined token standard that this pool conforms to (optional)
	Standard string

	// Event contains info on the underlying blockchain event for this pool creation
	Event blockchain.Event
}

type TokenTransfer struct {
	fftypes.TokenTransfer

	// PoolProtocolID is the ID assigned to the token pool by the connector
	PoolProtocolID string

	// Event contains info on the underlying blockchain event for this transfer
	Event blockchain.Event
}
