// Copyright © 2022 Kaleido, Inc.
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

package ethereum

import (
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/config/wsconfig"
)

const (
	defaultBatchSize    = 50
	defaultBatchTimeout = 500
	defaultPrefixShort  = "fly"
	defaultPrefixLong   = "firefly"

	defaultAddressResolverMethod        = "GET"
	defaultAddressResolverResponseField = "address"
	defaultAddressResolverCacheSize     = 1000
	defaultAddressResolverCacheTTL      = "24h"
)

const (
	// EthconnectConfigKey is a sub-key in the config to contain all the ethconnect specific config,
	EthconnectConfigKey = "ethconnect"
	// EthconnectConfigInstancePath is the /contracts/0x12345 or /instances/0x12345 path of the REST API exposed by ethconnect for the contract
	EthconnectConfigInstancePath = "instance"
	// EthconnectConfigTopic is the websocket listen topic that the node should register on, which is important if there are multiple
	// nodes using a single ethconnect
	EthconnectConfigTopic = "topic"
	// EthconnectConfigBatchSize is the batch size to configure on event streams, when auto-defining them
	EthconnectConfigBatchSize = "batchSize"
	// EthconnectConfigBatchTimeout is the batch timeout to configure on event streams, when auto-defining them
	EthconnectConfigBatchTimeout = "batchTimeout"
	// EthconnectPrefixShort is used in the query string in requests to ethconnect
	EthconnectPrefixShort = "prefixShort"
	// EthconnectPrefixLong is used in HTTP headers in requests to ethconnect
	EthconnectPrefixLong = "prefixLong"

	// AddressResolverConfigKey is a sub-key in the config to contain an address resolver config.
	AddressResolverConfigKey = "addressResolver"
	// AddressResolverRetainOriginal when true the original pre-resolved string is retained after the lookup, and passed down to EthConnect as the from address
	AddressResolverRetainOriginal = "retainOriginal"
	// AddressResolverMethod the HTTP method to use to call the address resolver (default GET)
	AddressResolverMethod = "method"
	// AddressResolverURLTemplate the URL go template string to use when calling the address resolver
	AddressResolverURLTemplate = "urlTemplate"
	// AddressResolverBodyTemplate the body go template string to use when calling the address resolver
	AddressResolverBodyTemplate = "bodyTemplate"
	// AddressResolverResponseField the name of a JSON field that is provided in the response, that contains the ethereum address (default "address")
	AddressResolverResponseField = "responseField"
	// AddressResolverCacheSize the size of the LRU cache
	AddressResolverCacheSize = "cache.size"
	// AddressResolverCacheTTL the TTL on cache entries
	AddressResolverCacheTTL = "cache.ttl"
)

func (e *Ethereum) InitPrefix(prefix config.Prefix) {
	ethconnectConf := prefix.SubPrefix(EthconnectConfigKey)
	wsconfig.InitPrefix(ethconnectConf)
	ethconnectConf.AddKnownKey(EthconnectConfigInstancePath)
	ethconnectConf.AddKnownKey(EthconnectConfigTopic)
	ethconnectConf.AddKnownKey(EthconnectConfigBatchSize, defaultBatchSize)
	ethconnectConf.AddKnownKey(EthconnectConfigBatchTimeout, defaultBatchTimeout)
	ethconnectConf.AddKnownKey(EthconnectPrefixShort, defaultPrefixShort)
	ethconnectConf.AddKnownKey(EthconnectPrefixLong, defaultPrefixLong)

	addressResolverConf := prefix.SubPrefix(AddressResolverConfigKey)
	wsconfig.InitPrefix(addressResolverConf)
	addressResolverConf.AddKnownKey(AddressResolverRetainOriginal)
	addressResolverConf.AddKnownKey(AddressResolverMethod, defaultAddressResolverMethod)
	addressResolverConf.AddKnownKey(AddressResolverURLTemplate)
	addressResolverConf.AddKnownKey(AddressResolverBodyTemplate)
	addressResolverConf.AddKnownKey(AddressResolverResponseField, defaultAddressResolverResponseField)
	addressResolverConf.AddKnownKey(AddressResolverCacheSize, defaultAddressResolverCacheSize)
	addressResolverConf.AddKnownKey(AddressResolverCacheTTL, defaultAddressResolverCacheTTL)
}
