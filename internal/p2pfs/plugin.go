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

package p2pfs

import (
	"context"
	"io"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/fftypes"
)

// Plugin is the interface implemented by each P2P Filesystem plugin
type Plugin interface {

	// InitConfigPrefix initializes the set of configuration options that are valid, with defaults. Called on all plugins.
	InitConfigPrefix(prefix config.ConfigPrefix)

	// Init initializes the plugin, with the config marshaled into the return of ConfigInterface
	// Returns the supported featureset of the interface
	Init(ctx context.Context, prefix config.ConfigPrefix, events Events) error

	// Capabilities returns capabilities - not called until after Init
	Capabilities() *Capabilities

	// PublishData publishes data to the P2P Filesystem, and returns a Bytes32 payload reference ID
	PublishData(ctx context.Context, data io.Reader) (payloadRef *fftypes.Bytes32, err error)

	// RetrieveData reads data back from IPFS using the payload reference format returned from PublishData
	RetrieveData(ctx context.Context, payloadRef *fftypes.Bytes32) (data io.ReadCloser, err error)
}

type Events interface {
}

type Capabilities struct {
}
