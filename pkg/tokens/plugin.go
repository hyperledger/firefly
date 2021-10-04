// Copyright © 2021 Kaleido, Inc.
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
	"github.com/hyperledger/firefly/pkg/fftypes"
)

// Plugin is the interface implemented by each tokens plugin
type Plugin interface {
	fftypes.Named

	// InitPrefix initializes the set of configuration options that are valid, with defaults. Called on all plugins.
	InitPrefix(prefix config.PrefixArray)

	// Init initializes the plugin, with configuration
	// Returns the supported featureset of the interface
	Init(ctx context.Context, prefix config.Prefix, callbacks Callbacks) error

	// Blockchain interface must not deliver any events until start is called
	Start() error

	// Capabilities returns capabilities - not called until after Init
	Capabilities() *Capabilities

	// CreateTokenPool creates a new (fungible or non-fungible) pool of tokens
	// The returned tracking ID will be used to correlate with any subsequent transaction tracking updates
	CreateTokenPool(ctx context.Context, pool *fftypes.TokenPool) error
}

// Callbacks is the interface provided to the tokens plugin, to allow it to pass events back to firefly.
//
// Events must be delivered sequentially, such that event 2 is not delivered until the callback invoked for event 1
// has completed. However, it does not matter if these events are workload balance between the firefly core
// cluster instances of the node.
type Callbacks interface {
	// TokensTxUpdate notifies firefly of an update to a transaction. Only success/failure and errorMessage (for errors) are modeled.
	// additionalInfo can be used to add opaque protocol specific JSON from the plugin (protocol transaction ID etc.)
	// Note this is an optional hook information, and stored separately to the confirmation of the actual event that was being submitted/sequenced.
	// Only the party submitting the transaction will see this data.
	//
	// Error should will only be returned in shutdown scenarios
	TokensTxUpdate(plugin Plugin, txTrackingID string, txState fftypes.OpStatus, errorMessage string, additionalInfo fftypes.JSONObject) error

	// TokenPoolCreated notifies on the creation of a new token pool, which might have been
	// submitted by us, or by any other authorized party in the network.
	//
	// Error should will only be returned in shutdown scenarios
	TokenPoolCreated(plugin Plugin, pool *fftypes.TokenPool, signingIdentity string, protocolTxID string, additionalInfo fftypes.JSONObject) error
}

// Capabilities the supported featureset of the tokens
// interface implemented by the plugin, with the specified config
type Capabilities struct {
}
