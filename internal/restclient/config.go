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

package restclient

import "github.com/kaleido-io/firefly/internal/config"

const (
	defaultRetryEnabled     = false
	defaultRetryCount       = 5
	defaultRetryWaitTime    = "250ms"
	defaultRetryMaxWaitTime = "30s"
)

const (
	HTTPConfigURL              = "url"
	HTTPConfigHeaders          = "headers"
	HTTPConfigAuthUsername     = "auth.username"
	HTTPConfigAuthPassword     = "auth.password"
	HTTPConfigRetryEnabled     = "retry.enabled"
	HTTPConfigRetryCount       = "retry.count"
	HTTPConfigRetryWaitTime    = "retry.waitTime"
	HTTPConfigRetryMaxWaitTime = "retry.maxWaitTime"

	// Unit test only
	HTTPCustomClient = "customClient"
)

func InitConfigPrefix(prefix config.ConfigPrefix) {
	prefix.AddKnownKey(HTTPConfigURL)
	prefix.AddKnownKey(HTTPConfigHeaders)
	prefix.AddKnownKey(HTTPConfigAuthUsername)
	prefix.AddKnownKey(HTTPConfigAuthPassword)
	prefix.AddKnownKey(HTTPConfigRetryEnabled, defaultRetryEnabled)
	prefix.AddKnownKey(HTTPConfigRetryCount, defaultRetryCount)
	prefix.AddKnownKey(HTTPConfigRetryWaitTime, defaultRetryWaitTime)
	prefix.AddKnownKey(HTTPConfigRetryMaxWaitTime, defaultRetryMaxWaitTime)

	prefix.AddKnownKey(HTTPCustomClient)
}
