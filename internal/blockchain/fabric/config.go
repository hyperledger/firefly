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

package fabric

import (
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/wsclient"
)

const (
	defaultBatchSize    = 50
	defaultBatchTimeout = 500
	defaultPrefixShort  = "fly"
	defaultPrefixLong   = "firefly"
)

const (
	// FabconnectConfigKey is a sub-key in the config to contain all the ethconnect specific config,
	FabconnectConfigKey = "fabconnect"

	// FabconnectConfigDefaultChannel is the default Fabric channel to use if no "ledger" is specified in requests
	FabconnectConfigDefaultChannel = "channel"
	// FabconnectConfigChaincode is the Fabric Firefly chaincode deployed to the Firefly channels
	FabconnectConfigChaincode = "chaincode"
	// FabconnectConfigTopic is the websocket listen topic that the node should register on, which is important if there are multiple
	// nodes using a single fabconnect
	FabconnectConfigTopic = "topic"
	// FabconnectConfigBatchSize is the batch size to configure on event streams, when auto-defining them
	FabconnectConfigBatchSize = "batchSize"
	// FabconnectConfigBatchTimeout is the batch timeout to configure on event streams, when auto-defining them
	FabconnectConfigBatchTimeout = "batchTimeout"
	// FabconnectConfigSkipEventstreamInit disables auto-configuration of event streams
	FabconnectConfigSkipEventstreamInit = "skipEventstreamInit"
	// FabconnectPrefixShort is used in the query string in requests to ethconnect
	FabconnectPrefixShort = "prefixShort"
	// FabconnectPrefixLong is used in HTTP headers in requests to ethconnect
	FabconnectPrefixLong = "prefixLong"
)

func (f *Fabric) InitPrefix(prefix config.Prefix) {
	fabconnectConf := prefix.SubPrefix(FabconnectConfigKey)
	wsclient.InitPrefix(fabconnectConf)
	fabconnectConf.AddKnownKey(FabconnectConfigDefaultChannel)
	fabconnectConf.AddKnownKey(FabconnectConfigChaincode)
	fabconnectConf.AddKnownKey(FabconnectConfigTopic)
	fabconnectConf.AddKnownKey(FabconnectConfigSkipEventstreamInit)
	fabconnectConf.AddKnownKey(FabconnectConfigBatchSize, defaultBatchSize)
	fabconnectConf.AddKnownKey(FabconnectConfigBatchTimeout, defaultBatchTimeout)
	fabconnectConf.AddKnownKey(FabconnectPrefixShort, defaultPrefixShort)
	fabconnectConf.AddKnownKey(FabconnectPrefixLong, defaultPrefixLong)
}
