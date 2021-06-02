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

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

// Plugin is the interface implemented by each data exchange plugin
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
	GetEndpointInfo(ctx context.Context) (endpoint fftypes.JSONObject, err error)

	// UploadBLOB streams a blob to storage
	UploadBLOB(ctx context.Context, ns string, id fftypes.UUID, reader io.Reader) error
}

// Callbacks is the interface provided to the data exchange plugin, to allow it to pass events back to firefly.
type Callbacks interface {
}

// Capabilities the supported featureset of the data exchange
// interface implemented by the plugin, with the specified config
type Capabilities struct {
}
