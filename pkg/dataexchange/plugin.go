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

package dataexchange

import (
	"context"
	"io"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

// Plugin is the interface implemented by each data exchange plugin
//
// Data exchange plugins are responsible for qualities of service for:
// - Storage of large data payloads
// - Security of transfer of data between participants (transport and payload authorization & encryption)
// - Reliability of transfer of data between participants (many transports can be supported - HTTPs/AMQP/MQTT etc.)
//
// Each plugin must handle network addressing, as well as transfer of messages and BLOBs.
//
// Network addressing:
// - Each node must have a "peerID" (<=256b) that uniquely identifies a node within the data exchange network
// - A node must provide a JSON package of data for connectivity to a node, which FireFly can distribute to all other nodes to address this node
//
// Messages:
// - Are provided in-line in requests to transfer to a peer
// - Must drive events on the target node that contain the input data
// - No requirement to retain the data beyond the confirmation of receipt of the event at the target
//
// BLOBS
// - Can be stored and retrieved separately from their transfer
// - Transfers are initiated via reference (not in-line data)
// - Are hashed by the DX plugin using the same hashing algorithm as FireFly (SHA256)
// - DX plugins can mainain their own internal IDs for BLOBs within the following requirements:
//   - Given a namespace and ID, map to a "payloadRef" string (<1024chars) that allows that same payload to be retrieved using only that payloadRef
//     - Example would be a logical filesystem path like "local/namespace/ID"
//   - When data is recevied from other members in the network, be able to return the hash when provided with the remote peerID string, namespace and ID
//     - Could be done by having a data store to resolve the transfers, or simply a deterministic path to metadata like "receive/peerID/namespace/ID"
//   - Events triggered for arrival of blobs must contain the payloadRef, and the hash
//
type Plugin interface {
	fftypes.Named

	// InitPrefix initializes the set of configuration options that are valid, with defaults. Called on all plugins.
	InitPrefix(prefix config.Prefix)

	// Init initializes the plugin, with configuration
	// Returns the supported featureset of the interface
	Init(ctx context.Context, prefix config.Prefix, callbacks Callbacks) error

	// Data exchange interface must not deliver any events until start is called
	Start() error

	// Capabilities returns capabilities - not called until after Init
	Capabilities() *Capabilities

	// GetEndpointInfo returns the information about the local endpoint
	GetEndpointInfo(ctx context.Context) (peerID string, endpoint fftypes.JSONObject, err error)

	// AddPeer translates the configuration published by another peer, into a reference string that is used between DX and FireFly to refer to the peer
	AddPeer(ctx context.Context, peerID string, endpoint fftypes.JSONObject) (err error)

	// UploadBLOB streams a blob to storage, and returns the hash to confirm the hash calculated in Core matches the hash calculated in the plugin
	UploadBLOB(ctx context.Context, ns string, id fftypes.UUID, content io.Reader) (payloadRef string, hash *fftypes.Bytes32, err error)

	// DownloadBLOB streams a received blob out of storage
	DownloadBLOB(ctx context.Context, payloadRef string) (content io.ReadCloser, err error)

	// CheckBLOBReceived confirms that a blob with the specified hash has been received from the specified peer
	CheckBLOBReceived(ctx context.Context, peerID, ns string, id fftypes.UUID) (hash *fftypes.Bytes32, err error)

	// SendMessage sends an in-line package of data to another network node.
	// Should return as quickly as possible for parallelsim, then report completion asynchronously via the operation ID
	SendMessage(ctx context.Context, peerID string, data []byte) (trackingID string, err error)

	// TransferBLOB initiates a transfer of a previoiusly stored blob to another node
	TransferBLOB(ctx context.Context, peerID string, payloadRef string) (trackingID string, err error)
}

// Callbacks is the interface provided to the data exchange plugin, to allow it to pass events back to firefly.
type Callbacks interface {

	// MessageReceived notifies of a message received from another node in the network
	MessageReceived(peerID string, data []byte)

	// BLOBReceived notifies of the ID of a BLOB that has been stored by DX after being received from another node in the network
	BLOBReceived(peerID string, hash *fftypes.Bytes32, payloadRef string)

	// TransferResult notifies of a status update of a transfer
	TransferResult(trackingID string, status fftypes.OpStatus, info string, additionalInfo fftypes.JSONObject)
}

// Capabilities the supported featureset of the data exchange
// interface implemented by the plugin, with the specified config
type Capabilities struct {
}
