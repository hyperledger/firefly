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

package identity

import (
	"context"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

// Plugin is the interface implemented by each identity plugin
type Plugin interface {
	fftypes.Named

	// InitPrefix initializes the set of configuration options that are valid, with defaults. Called on all plugins.
	InitPrefix(prefix config.Prefix)

	// Init initializes the plugin, with configuration
	// Returns the supported featureset of the interface
	Init(ctx context.Context, prefix config.Prefix, callbacks Callbacks) error

	// Blockchain interface must not deliver any events until start is called
	Start() error

	// Capabilities returns capabilities - not called until after Init
	Capabilities() *Capabilities

	// Resolve maps an identity identifier passed in a message to a full identity that includes the signing identity
	Resolve(ctx context.Context, identifier string) (identity *fftypes.Identity, err error)
}

// Callbacks is the interface provided to the identity plugin, to allow it to request information from firefly, or pass events.
type Callbacks interface {
}

// Capabilities the supported featureset of the identity
// interface implemented by the plugin, with the specified config
type Capabilities struct {
}
