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

package publicstorage

import (
	"context"
	"io"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

// Plugin is the interface implemented by each Public Storage plugin
type Plugin interface {
	fftypes.Named

	// InitPrefix initializes the set of configuration options that are valid, with defaults. Called on all plugins.
	InitPrefix(prefix config.Prefix)

	// Init initializes the plugin, with configuration
	// Returns the supported featureset of the interface
	Init(ctx context.Context, prefix config.Prefix, callbacks Callbacks) error

	// Capabilities returns capabilities - not called until after Init
	Capabilities() *Capabilities

	// PublishData publishes data to the Public Storage, and returns a payload reference ID
	PublishData(ctx context.Context, data io.Reader) (payloadRef string, err error)

	// RetrieveData reads data back from IPFS using the payload reference format returned from PublishData
	RetrieveData(ctx context.Context, payloadRef string) (data io.ReadCloser, err error)
}

type Callbacks interface {
}

type Capabilities struct {
}
