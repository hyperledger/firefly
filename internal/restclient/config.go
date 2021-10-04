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

package restclient

import "github.com/hyperledger/firefly/internal/config"

const (
	defaultRetryEnabled     = false
	defaultRetryCount       = 5
	defaultRetryWaitTime    = "250ms"
	defaultRetryMaxWaitTime = "30s"
	defaultRequestTimeout   = "30s"
)

const (
	// HTTPConfigURL is the url to connect to for this HTTP configuration
	HTTPConfigURL = "url"
	// HTTPConfigProxyURL adds a proxy
	HTTPConfigProxyURL = "proxy.url"
	// HTTPConfigHeaders adds custom headers to the requests
	HTTPConfigHeaders = "headers"
	// HTTPConfigAuthUsername HTTPS Basic Auth configuration - username
	HTTPConfigAuthUsername = "auth.username"
	// HTTPConfigAuthPassword HTTPS Basic Auth configuration - secret / password
	HTTPConfigAuthPassword = "auth.password"
	// HTTPConfigRetryEnabled whether retry is enabled on the actions performed over this HTTP request (does not disable retry at higher layers)
	HTTPConfigRetryEnabled = "retry.enabled"
	// HTTPConfigRetryCount the maximum number of retries
	HTTPConfigRetryCount = "retry.count"
	// HTTPConfigRetryInitDelay the initial retry delay
	HTTPConfigRetryInitDelay = "retry.initWaitTime"
	// HTTPConfigRetryMaxDelay the maximum retry delay
	HTTPConfigRetryMaxDelay = "retry.maxWaitTime"
	// HTTPConfigRequestTimeout the request timeout
	HTTPConfigRequestTimeout = "requestTimeout"

	// HTTPCustomClient - unit test only - allows injection of a custom HTTP client to resty
	HTTPCustomClient = "customClient"
)

func InitPrefix(prefix config.KeySet) {
	prefix.AddKnownKey(HTTPConfigURL)
	prefix.AddKnownKey(HTTPConfigProxyURL)
	prefix.AddKnownKey(HTTPConfigHeaders)
	prefix.AddKnownKey(HTTPConfigAuthUsername)
	prefix.AddKnownKey(HTTPConfigAuthPassword)
	prefix.AddKnownKey(HTTPConfigRetryEnabled, defaultRetryEnabled)
	prefix.AddKnownKey(HTTPConfigRetryCount, defaultRetryCount)
	prefix.AddKnownKey(HTTPConfigRetryInitDelay, defaultRetryWaitTime)
	prefix.AddKnownKey(HTTPConfigRetryMaxDelay, defaultRetryMaxWaitTime)
	prefix.AddKnownKey(HTTPConfigRequestTimeout, defaultRequestTimeout)

	prefix.AddKnownKey(HTTPCustomClient)
}
