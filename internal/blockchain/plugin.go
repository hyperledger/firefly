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

package plugin

import (
	"context"
	"encoding/hex"

	"github.com/google/uuid"
)

// BlockchainPlugin is the interface implemented by each blockchain plugin
type BlockchainPlugin interface {

	// ConfigInterface returns the structure into which to marshal the plugin config
	ConfigInterface() interface{}

	// Init initializes the plugin, with the config marshaled into the return of ConfigInterface
	// Returns the supported featureset of the interface
	Init(ctx context.Context, config interface{}, events BlockchainEvents) (*BlockchainCapabilities, error)

	// SubmitBroadcastBatch sequences a broadcast globally to all viewers of the blockchain
	// The returned tracking ID will be used to correlate with any subsequent transaction tracking updates
	SubmitBroadcastBatch(identity string, broadcast BroadcastBatch) (txTrackingID string, err error)
}

// BlockchainEvents is the interface provided to the blockchain plugin, to allow it to pass events back to firefly.
// Note: It does not matter which Firefly core runtime the event is dispatched to (it does not need to be
//       the same runtime as the one that performed the submit).
type BlockchainEvents interface {
	// TransactionUpdate notifies firefly of an update to a transaction. Only success/failure and errorMessage (for errors) are modeled.
	// additionalInfo can be used to add opaque protocol specific JSON from the plugin (protocol transaction ID etc.)
	// Note this is an optional hook information, and stored separately to the confirmation of the actual event that was being submitted/sequenced.
	// Only the party submitting the transaction will see this data.
	TransactionUpdate(txTrackingID string, txState TransactionState, errorMessage string, additionalInfo map[string]interface{})

	// SequencedBroadcastBatch notifies on the arrival of a sequenced batch of broadcast messages, which might have been
	// submitted by us, or by any other authorized party in the network.
	// additionalInfo can be used to add opaque protocol specific JSON from the plugin (block numbers etc.)
	SequencedBroadcastBatch(batch BroadcastBatch, additionalInfo map[string]interface{})
}

// BlockchainCapabilities the supported featureset of the blockchain
// interface implemented by the plugin, with the specified config
type BlockchainCapabilities struct {
	// GlobalSequencer means submitting an ordered piece of data visible to all
	// participants of the network (requires an all-participant chain)
	GlobalSequencer bool
}

// Bytes32 is a holder of a hash, that can be used to correlate onchain data with off-chain data.
type Bytes32 = [32]byte

// HexUUID is 32 character ASCII string containing the hex representation of UUID, with the dashes of the canonical representation removed
type HexUUID = [32]byte

// HexUUIDFromUUID returns the bytes of a UUID as a compressed hex string
func HexUUIDFromUUID(u uuid.UUID) HexUUID {
	var d HexUUID
	hex.Encode(d[0:15], u[0:7])
	return d
}

// TransactionState is the only architecturally significant thing that Firefly tracks on blockchain transactions.
// All other data is consider protocol specific, and hence stored as opaque data.
type TransactionState string

const (
	// TransactionStateSubmitted the transaction has been submitted
	TransactionStateSubmitted TransactionState = "submitted"
	// TransactionStateSubmitted the transaction is considered final per the rules of the blockchain technnology
	TransactionStateConfirmed TransactionState = "confirmed"
	// TransactionStateSubmitted the transaction has encountered, and is unlikely to ever become final on the blockchain. However, it is not impossible it will still be mined.
	TransactionStateFailed TransactionState = "error"
)

// BroadcastBatch is the set of data pinned to the blockchain for a batch of broadcasts.
// Broadcasts are batched where possible, as the storage of the off-chain data is expensive as it must be propagated to all members
// of the network (via a technology like IPFS).
type BroadcastBatch struct {

	// Timestamp is the time of submission, from the pespective of the submitter
	Timestamp uint64

	// BatchPaylodRef is a 32 byte fixed length binary value that can be passed to the storage interface to retrieve the payload
	BatchPaylodRef HexUUID

	// BatchID is the id of the batch - writing this in plain text to the blockchain makes for easy correlation on-chain/off-chain
	BatchID Bytes32
}
