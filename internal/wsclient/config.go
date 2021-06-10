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

package wsclient

import (
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/restclient"
)

const (
	defaultIntialConnectAttempts = 5
	defaultBufferSize            = "16Kb"
)

const (
	// WSSpecificConfPrefix is the sub-section of the http config options that contains websocket specific config
	WSSpecificConfPrefix = "ws"

	// WSConfigKeyWriteBufferSize is the write buffer size
	WSConfigKeyWriteBufferSize = "ws.writeBufferSize"
	// WSConfigKeyReadBufferSize is the read buffer size
	WSConfigKeyReadBufferSize = "ws.readBufferSize"
	// WSConfigKeyInitialConnectAttempts sets how many times the websocket should attempt to connect on startup, before failing (after initial connection, retry is indefinite)
	WSConfigKeyInitialConnectAttempts = "ws.initialConnectAttempts"
	// WSConfigKeyPath if set will define the path to connect to - allows sharing of the same URL between HTTP and WebSocket connection info
	WSConfigKeyPath = "ws.path"
)

// InitPrefix ensures the prefix is initialized for HTTP too, as WS and HTTP
// can share the same tree of configuration (and all the HTTP options apply to the initial upgrade)
func InitPrefix(prefix config.Prefix) {
	restclient.InitPrefix(prefix)
	prefix.AddKnownKey(WSConfigKeyWriteBufferSize, defaultBufferSize)
	prefix.AddKnownKey(WSConfigKeyReadBufferSize, defaultBufferSize)
	prefix.AddKnownKey(WSConfigKeyInitialConnectAttempts, defaultIntialConnectAttempts)
	prefix.AddKnownKey(WSConfigKeyPath)
}
