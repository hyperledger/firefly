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

package ethereum

import (
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/ffresty"
	"github.com/kaleido-io/firefly/internal/wsclient"
)

const (
	defaultBatchSize    = 50
	defaultBatchTimeout = 500
)

const (
	EthconnectConfigKey = "ethconnect"

	EthconnectConfigInstancePath        = "instance"
	EthconnectConfigTopic               = "topic"
	EthconnectConfigBatchSize           = "batchSize"
	EthconnectConfigBatchTimeoutMS      = "batchTimeoutMS"
	EthconnectConfigSkipEventstreamInit = "skipEventstreamInit"
)

// AddEthconnectConfig extends config already initialized with ffresty.AddHTTPConfig()
func AddEthconnectConfig(conf config.Config) config.Config {
	ethconnectConf := conf.SubKey(EthconnectConfigKey)
	ffresty.AddHTTPConfig(ethconnectConf)
	wsclient.AddWSConfig(ethconnectConf)

	ethconnectConf.AddKey(EthconnectConfigInstancePath)
	ethconnectConf.AddKey(EthconnectConfigTopic)
	ethconnectConf.AddKey(EthconnectConfigSkipEventstreamInit)
	ethconnectConf.AddKey(EthconnectConfigBatchSize, defaultBatchSize)
	ethconnectConf.AddKey(EthconnectConfigBatchTimeoutMS, defaultBatchTimeout)

	return ethconnectConf
}
