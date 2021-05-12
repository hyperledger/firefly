// Copyright © 2021 Kaleido, Inc.
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

package blockchain

import (
	"context"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/fftypes"
)

// Plugin is the interface implemented by each blockchain plugin
type Plugin interface {

	// InitConfigPrefix initializes the set of configuration options that are valid, with defaults. Called on all plugins.
	InitConfigPrefix(prefix config.ConfigPrefix)

	// Init initializes the plugin, with the config marshaled into the return of ConfigInterface
	// Returns the supported featureset of the interface
	Init(ctx context.Context, prefix config.ConfigPrefix, events Events) error

	// Capabilities returns capabilities - not called until after Init
	Capabilities() *Capabilities

	// VerifyIdentitySyntax verifies that the supplied identity string is valid syntax acording to the protocol.
	// Also applies any transformations, such as lower case
	VerifyIdentitySyntax(ctx context.Context, identity string) (string, error)

	// SubmitBroadcastBatch sequences a broadcast globally to all viewers of the blockchain
	// The returned tracking ID will be used to correlate with any subsequent transaction tracking updates
	SubmitBroadcastBatch(ctx context.Context, identity string, batch *BroadcastBatch) (txTrackingID string, err error)
}

// BlockchainEvents is the interface provided to the blockchain plugin, to allow it to pass events back to firefly.
//
// Events must be delivered sequentially, such that event 2 is not delivered until the callback invoked for event 1
// has completed. However, it does not matter if these events are workload balance between the firefly core
// cluster instances of the node.
type Events interface {
	// TransactionUpdate notifies firefly of an update to a transaction. Only success/failure and errorMessage (for errors) are modeled.
	// additionalInfo can be used to add opaque protocol specific JSON from the plugin (protocol transaction ID etc.)
	// Note this is an optional hook information, and stored separately to the confirmation of the actual event that was being submitted/sequenced.
	// Only the party submitting the transaction will see this data.
	TransactionUpdate(txTrackingID string, txState TransactionState, protocolTxId, errorMessage string, additionalInfo map[string]interface{})

	// SequencedBroadcastBatch notifies on the arrival of a sequenced batch of broadcast messages, which might have been
	// submitted by us, or by any other authorized party in the network.
	// Will be combined with he index within the batch, to allocate a sequence to each message in the batch.
	// For example a padded block number, followed by a padded transaction index within that block.
	// additionalInfo can be used to add opaque protocol specific JSON from the plugin (block numbers etc.)
	SequencedBroadcastBatch(batch *BroadcastBatch, author string, protocolTxId string, additionalInfo map[string]interface{})
}

// BlockchainCapabilities the supported featureset of the blockchain
// interface implemented by the plugin, with the specified config
type Capabilities struct {
	// GlobalSequencer means submitting an ordered piece of data visible to all
	// participants of the network (requires an all-participant chain)
	GlobalSequencer bool
}

// TransactionState is the only architecturally significant thing that Firefly tracks on blockchain transactions.
// All other data is consider protocol specific, and hence stored as opaque data.
type TransactionState = fftypes.TransactionState

// BroadcastBatch is the set of data pinned to the blockchain for a batch of broadcasts.
// Broadcasts are batched where possible, as the storage of the off-chain data is expensive as it must be propagated to all members
// of the network (via a technology like IPFS).
type BroadcastBatch struct {

	// Timestamp is the millisecond resolution timestamp of submission, from the pespective of the submitter
	Timestamp int64

	// BatchID is the id of the batch - writing this in plain text to the blockchain makes for easy correlation on-chain/off-chain
	BatchID *uuid.UUID

	// BatchPaylodRef is a 32 byte fixed length binary value that can be passed to the storage interface to retrieve the payload
	BatchPaylodRef *fftypes.Bytes32
}
