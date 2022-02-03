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

package restclient

import "github.com/hyperledger/firefly/internal/config"

const (
	defaultRetryEnabled              = false
	defaultRetryCount                = 5
	defaultRetryWaitTime             = "250ms"
	defaultRetryMaxWaitTime          = "30s"
	defaultRequestTimeout            = "30s"
	defaultHTTPIdleTimeout           = "475ms" // Node.js default keepAliveTimeout is 5 seconds, so we have to set a base below this
	defaultHTTPMaxIdleConns          = 100     // match Go's default
	defaultHTTPConnectionTimeout     = "30s"
	defaultHTTPTLSHandshakeTimeout   = "10s" // match Go's default
	defaultHTTPExpectContinueTimeout = "1s"  // match Go's default
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
	// HTTPIdleTimeout the max duration to hold a HTTP keepalive connection between calls
	HTTPIdleTimeout = "idleTimeout"
	// HTTPMaxIdleConns the max number of idle connections to hold pooled
	HTTPMaxIdleConns = "maxIdleConns"
	// HTTPConnectionTimeout the connection timeout for new connections
	HTTPConnectionTimeout = "connectionTimeout"
	// HTTPTLSHandshakeTimeout the TLS handshake connection timeout
	HTTTPTLSHandshakeTimeout = "tlsHandshakeTimeout"
	// HTTPExpectContinueTimeout see ExpectContinueTimeout in Go docs
	HTTPExpectContinueTimeout = "expectContinueTimeout"

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
	prefix.AddKnownKey(HTTPIdleTimeout, defaultHTTPIdleTimeout)
	prefix.AddKnownKey(HTTPMaxIdleConns, defaultHTTPMaxIdleConns)
	prefix.AddKnownKey(HTTPConnectionTimeout, defaultHTTPConnectionTimeout)
	prefix.AddKnownKey(HTTTPTLSHandshakeTimeout, defaultHTTPTLSHandshakeTimeout)
	prefix.AddKnownKey(HTTPExpectContinueTimeout, defaultHTTPExpectContinueTimeout)

	prefix.AddKnownKey(HTTPCustomClient)
}
