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

package wsclient

import (
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/restclient"
)

const (
	defaultIntialConnectAttempts = 5
	defaultBufferSizeKB          = 1024
)

const (
	WSSpecificConfPrefix = "ws"

	WSConfigKeyWriteBufferSizeKB      = "ws.writeBufferSizeKB"
	WSConfigKeyReadBufferSizeKB       = "ws.readBufferSizeKB"
	WSConfigKeyInitialConnectAttempts = "ws.initialConnectAttempts"
	WSConfigKeyPath                   = "ws.path"
)

// InitConfigPrefix ensures the prefix is initialized for HTTP too, as WS and HTTP
// can share the same tree of configuration (and all the HTTP options apply to the initial upgrade)
func InitConfigPrefix(prefix config.ConfigPrefix) {
	restclient.InitConfigPrefix(prefix)
	prefix.AddKnownKey(WSConfigKeyWriteBufferSizeKB, defaultBufferSizeKB)
	prefix.AddKnownKey(WSConfigKeyReadBufferSizeKB, defaultBufferSizeKB)
	prefix.AddKnownKey(WSConfigKeyInitialConnectAttempts, defaultIntialConnectAttempts)
	prefix.AddKnownKey(WSConfigKeyPath)
}
