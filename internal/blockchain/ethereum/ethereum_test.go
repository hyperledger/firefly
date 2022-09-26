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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly/internal/blockchain/common"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/cachemocks"
	"github.com/hyperledger/firefly/mocks/coremocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/wsmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfig = config.RootSection("eth_unit_tests")
var utEthconnectConf = utConfig.SubSection(EthconnectConfigKey)
var utAddressResolverConf = utConfig.SubSection(AddressResolverConfigKey)
var utFFTMConf = utConfig.SubSection(FFTMConfigKey)

func testFFIMethod() *fftypes.FFIMethod {
	return &fftypes.FFIMethod{
		Name: "sum",
		Params: []*fftypes.FFIParam{
			{
				Name:   "x",
				Schema: fftypes.JSONAnyPtr(`{"oneOf":[{"type":"string"},{"type":"integer"}],"details":{"type":"uint256"}}`),
			},
			{
				Name:   "y",
				Schema: fftypes.JSONAnyPtr(`{"oneOf":[{"type":"string"},{"type":"integer"}],"details":{"type":"uint256"}}`),
			},
		},
		Returns: []*fftypes.FFIParam{
			{
				Name:   "z",
				Schema: fftypes.JSONAnyPtr(`{"oneOf":[{"type":"string"},{"type":"integer"}],"details":{"type":"uint256"}}`),
			},
		},
	}
}

func resetConf(e *Ethereum) {
	coreconfig.Reset()
	e.InitConfig(utConfig)
}

func newTestEthereum() (*Ethereum, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	wsm := &wsmocks.WSClient{}
	mm := &metricsmocks.Manager{}
	mm.On("IsMetricsEnabled").Return(true)
	mm.On("BlockchainTransaction", mock.Anything, mock.Anything).Return(nil)
	mm.On("BlockchainQuery", mock.Anything, mock.Anything).Return(nil)
	e := &Ethereum{
		ctx:         ctx,
		cancelCtx:   cancel,
		client:      resty.New().SetBaseURL("http://localhost:12345"),
		topic:       "topic1",
		prefixShort: defaultPrefixShort,
		prefixLong:  defaultPrefixLong,
		wsconn:      wsm,
		metrics:     mm,
		cache:       cache.NewUmanagedCache(ctx, 100, 5*time.Minute),
		callbacks:   common.NewBlockchainCallbacks(),
		subs:        common.NewFireflySubscriptions(),
	}
	return e, func() {
		cancel()
		if e.closed != nil {
			// We've init'd, wait to close
			<-e.closed
		}
	}
}

func newTestStreamManager(client *resty.Client) *streamManager {
	return newStreamManager(client, cache.NewUmanagedCache(context.Background(), 100, 5*time.Minute))
}

func mockNetworkVersion(t *testing.T, version int) func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		readBody, _ := req.GetBody()
		var body map[string]interface{}
		json.NewDecoder(readBody).Decode(&body)
		headers := body["headers"].(map[string]interface{})
		method := body["method"].(map[string]interface{})
		if headers["type"] == "Query" && method["name"] == "networkVersion" {
			return httpmock.NewJsonResponderOrPanic(200, queryOutput{
				Output: fmt.Sprintf("%d", version),
			})(req)
		}
		return nil, nil
	}
}

func TestInitMissingURL(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestInitBadAddressResolver(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	utAddressResolverConf.Set(AddressResolverURLTemplate, "{{unclosed}")
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.Regexp(t, "FF10337.*urlTemplate", err)
}

func TestInitMissingTopic(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x12345")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.Regexp(t, "FF10138.*topic", err)
}

func TestInitAndStartWithEthConnect(t *testing.T) {

	log.SetLevel("trace")
	e, cancel := newTestEthereum()
	defer cancel()

	toServer, fromServer, wsURL, done := wsclient.NewTestWSServer(nil)
	defer done()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	u, _ := url.Parse(wsURL)
	u.Scheme = "http"
	httpURL := u.String()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/eventstreams", httpURL),
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/eventstreams", httpURL),
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, httpURL)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")
	utFFTMConf.Set(ffresty.HTTPConfigURL, "http://ethc.example.com:12345")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
	assert.NotNil(t, e.fftmClient)

	assert.Equal(t, "ethereum", e.Name())
	assert.Equal(t, core.VerifierTypeEthAddress, e.VerifierType())

	assert.NoError(t, err)

	assert.Equal(t, 2, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
	assert.NotNil(t, e.Capabilities())

	err = e.Start()
	assert.NoError(t, err)

	startupMessage := <-toServer
	assert.Equal(t, `{"type":"listen","topic":"topic1"}`, startupMessage)
	startupMessage = <-toServer
	assert.Equal(t, `{"type":"listenreplies"}`, startupMessage)
	fromServer <- `[]` // empty batch, will be ignored, but acked
	reply := <-toServer
	assert.Equal(t, `{"type":"ack","topic":"topic1"}`, reply)

	// Bad data will be ignored
	fromServer <- `!json`
	fromServer <- `{"not": "a reply"}`
	fromServer <- `42`

}

func TestInitAndStartWithFFTM(t *testing.T) {

	log.SetLevel("trace")
	e, cancel := newTestEthereum()
	defer cancel()

	toServer, fromServer, wsURL, done := wsclient.NewTestWSServer(nil)
	defer done()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	u, _ := url.Parse(wsURL)
	u.Scheme = "http"
	httpURL := u.String()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/eventstreams", httpURL),
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/eventstreams", httpURL),
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, httpURL)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")
	utFFTMConf.Set(ffresty.HTTPConfigURL, "http://fftm.example.com:12345")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
	assert.NotNil(t, e.fftmClient)

	assert.Equal(t, "ethereum", e.Name())
	assert.Equal(t, core.VerifierTypeEthAddress, e.VerifierType())

	assert.NoError(t, err)

	assert.Equal(t, 2, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
	assert.NotNil(t, e.Capabilities())

	err = e.Start()
	assert.NoError(t, err)

	startupMessage := <-toServer
	assert.Equal(t, `{"type":"listen","topic":"topic1"}`, startupMessage)
	startupMessage = <-toServer
	assert.Equal(t, `{"type":"listenreplies"}`, startupMessage)
	fromServer <- `{"batchNumber":12345,"events":[]}` // empty batch, will be ignored, but acked
	reply := <-toServer
	assert.Equal(t, `{"type":"ack","topic":"topic1","batchNumber":12345}`, reply)

	// Bad data will be ignored
	fromServer <- `!json`
	fromServer <- `{"not": "a reply"}`
	fromServer <- `42`

}

func TestWSInitFail(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "!!!://")
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.Regexp(t, "FF00149", err)

}

func TestEthCacheInitFail(t *testing.T) {
	cacheInitError := errors.New("Initialization error.")
	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "es12345", body["stream"])
			return httpmock.NewJsonResponderOrPanic(200, subscription{ID: "sub12345"})(req)
		})
	httpmock.RegisterResponder("GET", "http://localhost:12345/contracts/firefly",
		httpmock.NewJsonResponderOrPanic(200, map[string]string{
			"created":      "2022-02-08T22:10:10Z",
			"address":      "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"path":         "/contracts/firefly",
			"abi":          "fc49dec3-0660-4dc7-61af-65af4c3ac456",
			"openapi":      "/contracts/firefly?swagger",
			"registeredAs": "firefly",
		}),
	)
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 1))

	e, cancel := newTestEthereum()
	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/contracts/firefly")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(nil, cacheInitError)

	defer cancel()
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.Equal(t, cacheInitError, err)
}

func TestInitOldInstancePathContracts(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "es12345", body["stream"])
			return httpmock.NewJsonResponderOrPanic(200, subscription{ID: "sub12345"})(req)
		})
	httpmock.RegisterResponder("GET", "http://localhost:12345/contracts/firefly",
		httpmock.NewJsonResponderOrPanic(200, map[string]string{
			"created":      "2022-02-08T22:10:10Z",
			"address":      "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"path":         "/contracts/firefly",
			"abi":          "fc49dec3-0660-4dc7-61af-65af4c3ac456",
			"openapi":      "/contracts/firefly?swagger",
			"registeredAs": "firefly",
		}),
	)
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 1))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/contracts/firefly")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
	assert.NoError(t, err)
}

func TestInitOldInstancePathInstances(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "es12345", body["stream"])
			return httpmock.NewJsonResponderOrPanic(200, subscription{ID: "sub12345"})(req)
		})
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 1))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
	assert.NoError(t, err)
}

func TestInitNewConfig(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.Equal(t, 2, httpmock.GetTotalCallCount())
	assert.NoError(t, err)
}

func TestInitNewConfigBadIndex(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
}

func TestInitNetworkVersionNotFound(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/",
		httpmock.NewJsonResponderOrPanic(500, ethError{Error: "FFEC100148"}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
	assert.NoError(t, err)
}

func TestStreamQueryError(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewStringResponder(500, `pop`))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.Regexp(t, "FF10111.*pop", err)

}

func TestStreamCreateError(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewStringResponder(500, `pop`))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.Regexp(t, "FF10111.*pop", err)

}

func TestStreamUpdateError(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))
	httpmock.RegisterResponder("PATCH", "http://localhost:12345/eventstreams/es12345",
		httpmock.NewStringResponder(500, `pop`))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.Regexp(t, "FF10111.*pop", err)
}

func TestInitAllExistingStreams(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Stream: "es12345", Name: "ns1_BatchPin_3078373143373635" /* this is the subname for our combo of instance path and BatchPin */},
		}))
	httpmock.RegisterResponder("PATCH", "http://localhost:12345/eventstreams/es12345",
		httpmock.NewJsonResponderOrPanic(200, &eventStream{ID: "es12345", Name: "topic1"}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 2))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, subscription{}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.NoError(t, err)

	assert.Equal(t, 4, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
}

func TestInitAllExistingStreamsV1(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Stream: "es12345", Name: "BatchPin_3078373143373635" /* this is the subname for our combo of instance path and BatchPin */},
		}))
	httpmock.RegisterResponder("PATCH", "http://localhost:12345/eventstreams/es12345",
		httpmock.NewJsonResponderOrPanic(200, &eventStream{ID: "es12345", Name: "topic1"}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 1))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, subscription{}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.NoError(t, err)

	assert.Equal(t, 4, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
}

func TestInitAllExistingStreamsOld(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Stream: "es12345", Name: "BatchPin"},
		}))
	httpmock.RegisterResponder("PATCH", "http://localhost:12345/eventstreams/es12345",
		httpmock.NewJsonResponderOrPanic(200, &eventStream{ID: "es12345", Name: "topic1"}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 1))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, subscription{}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.NoError(t, err)

	assert.Equal(t, 4, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
}

func TestInitAllExistingStreamsInvalidName(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Stream: "es12345", Name: "BatchPin_3078373143373635" /* this is the subname for our combo of instance path and BatchPin */},
		}))
	httpmock.RegisterResponder("PATCH", "http://localhost:12345/eventstreams/es12345",
		httpmock.NewJsonResponderOrPanic(200, &eventStream{ID: "es12345", Name: "topic1"}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 2))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, subscription{}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.Regexp(t, "FF10416", err)
}

func TestSubQueryError(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewStringResponder(500, `pop`))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
}

func TestSubQueryCreateError(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewStringResponder(500, `pop`))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71C7656EC7ab88b098defB751B7401B5f6d8976F")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)
}

func TestSubmitBatchPinOK(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BatchPin{
		TransactionID:   fftypes.MustParseUUID("9ffc50ff-6bfe-4502-adc7-93aea54cc059"),
		BatchID:         fftypes.MustParseUUID("c5df767c-fe44-4e03-8eb5-1c5523097db5"),
		BatchHash:       fftypes.NewRandB32(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			res, err := mockNetworkVersion(t, 2)(req)
			if res != nil || err != nil {
				return res, err
			}

			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "SendTransaction", headers["type"])
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", params[0])
			assert.Equal(t, ethHexFormatB32(batch.BatchHash), params[1])
			assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", params[2])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	err := e.SubmitBatchPin(context.Background(), "", "ns1", addr, batch, location)

	assert.NoError(t, err)
}

func TestSubmitBatchPinV1(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BatchPin{
		TransactionID:   fftypes.MustParseUUID("9ffc50ff-6bfe-4502-adc7-93aea54cc059"),
		BatchID:         fftypes.MustParseUUID("c5df767c-fe44-4e03-8eb5-1c5523097db5"),
		BatchHash:       fftypes.NewRandB32(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			res, err := mockNetworkVersion(t, 1)(req)
			if res != nil || err != nil {
				return res, err
			}

			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "SendTransaction", headers["type"])
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", params[1])
			assert.Equal(t, ethHexFormatB32(batch.BatchHash), params[2])
			assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", params[3])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())
	err := e.SubmitBatchPin(context.Background(), "", "ns1", addr, batch, location)

	assert.NoError(t, err)
}

func TestSubmitBatchPinBadLocation(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BatchPin{
		TransactionID:   fftypes.MustParseUUID("9ffc50ff-6bfe-4502-adc7-93aea54cc059"),
		BatchID:         fftypes.MustParseUUID("c5df767c-fe44-4e03-8eb5-1c5523097db5"),
		BatchHash:       fftypes.NewRandB32(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"bad": "bad",
	}.String())
	err := e.SubmitBatchPin(context.Background(), "", "ns1", addr, batch, location)

	assert.Regexp(t, "FF10310", err)
}

func TestSubmitBatchEmptyPayloadRef(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BatchPin{
		TransactionID: fftypes.MustParseUUID("9ffc50ff-6bfe-4502-adc7-93aea54cc059"),
		BatchID:       fftypes.MustParseUUID("c5df767c-fe44-4e03-8eb5-1c5523097db5"),
		BatchHash:     fftypes.NewRandB32(),
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			res, err := mockNetworkVersion(t, 2)(req)
			if res != nil || err != nil {
				return res, err
			}

			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "SendTransaction", headers["type"])
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", params[0])
			assert.Equal(t, ethHexFormatB32(batch.BatchHash), params[1])
			assert.Equal(t, "", params[2])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	err := e.SubmitBatchPin(context.Background(), "", "ns1", addr, batch, location)

	assert.NoError(t, err)

}

func TestSubmitBatchPinVersionFail(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BatchPin{
		TransactionID:   fftypes.NewUUID(),
		BatchID:         fftypes.NewUUID(),
		BatchHash:       fftypes.NewRandB32(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		httpmock.NewStringResponder(500, "pop"))

	err := e.SubmitBatchPin(context.Background(), "", "ns1", addr, batch, location)

	assert.Regexp(t, "FF10111.*pop", err)

}

func TestSubmitBatchPinFail(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BatchPin{
		TransactionID:   fftypes.NewUUID(),
		BatchID:         fftypes.NewUUID(),
		BatchHash:       fftypes.NewRandB32(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			res, err := mockNetworkVersion(t, 2)(req)
			if res != nil || err != nil {
				return res, err
			}
			return httpmock.NewStringResponder(500, "pop")(req)
		})

	err := e.SubmitBatchPin(context.Background(), "", "ns1", addr, batch, location)

	assert.Regexp(t, "FF10111.*pop", err)

}

func TestSubmitBatchPinError(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BatchPin{
		TransactionID:   fftypes.NewUUID(),
		BatchID:         fftypes.NewUUID(),
		BatchHash:       fftypes.NewRandB32(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			res, err := mockNetworkVersion(t, 2)(req)
			if res != nil || err != nil {
				return res, err
			}
			return httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{
				"error": "Unknown error",
			})(req)
		})

	err := e.SubmitBatchPin(context.Background(), "", "ns1", addr, batch, location)

	assert.Regexp(t, "FF10111.*Unknown error", err)
}

func TestVerifyEthAddress(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	_, err := e.NormalizeSigningKey(context.Background(), "0x12345")
	assert.Regexp(t, "FF10141", err)

	key, err := e.NormalizeSigningKey(context.Background(), "0x2a7c9D5248681CE6c393117E641aD037F5C079F6")
	assert.NoError(t, err)
	assert.Equal(t, "0x2a7c9d5248681ce6c393117e641ad037f5c079f6", key)

}

func TestHandleMessageBatchPinOK(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"namespace": "ns1",
			"uuids": "0xe19af8b390604051812d7597d19adfb9847d3bfd074249efb65d3fed15f5b0a6",
			"batchHash": "0xd71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be",
			"payloadRef": "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
			"contexts": [
				"0x68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a",
				"0x19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771"
			]
    },
		"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"signature": "0x1C197604587F046FD40684A8f21f4609FB811A7b:BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
		"logIndex": "50",
		"timestamp": "1620576488"
  },
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x1",
		"transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
			"author": "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
			"namespace": "ns1",
			"uuids": "0x8a578549e56b49f9bd78d731f22b08d7a04c7cc37d444c2ba3b054e21326697e",
			"batchHash": "0x20e6ef9b9c4df7fdb77a7de1e00347f4b02d996f2e56a7db361038be7b32a154",
			"payloadRef": "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
			"contexts": [
				"0x8a63eb509713b0cf9250a8eee24ee2dfc4b37225e3ad5c29c95127699d382f85"
			]
    },
		"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
		"logIndex": "51",
		"timestamp": "1620576488"
  },
	{
		"address": "0x06d34B270F15a0d82913EFD0627B0F62Fd22ecd5",
		"blockNumber": "38011",
		"transactionIndex": "0x2",
		"transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
			"author": "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
			"namespace": "ns1",
			"uuids": "0x8a578549e56b49f9bd78d731f22b08d7a04c7cc37d444c2ba3b054e21326697e",
			"batchHash": "0x892b31099b8476c0692a5f2982ea23a0614949eacf292a64a358aa73ecd404b4",
			"payloadRef": "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
			"contexts": [
				"0xdab67320f1a0d0f1da572975e3a9ab6ef0fed315771c99fea0bfb54886c1aa94"
			]
    },
		"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"signature": "Random(address,uint256,bytes32,bytes32,bytes32)",
		"logIndex": "51",
		"timestamp": "1620576488"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)

	expectedSigningKeyRef := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
	}

	em.On("BatchPinComplete", "ns1", mock.Anything, expectedSigningKeyRef, mock.Anything).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	em.AssertExpectations(t)

	b := em.Calls[0].Arguments[1].(*blockchain.BatchPin)
	assert.Equal(t, "e19af8b3-9060-4051-812d-7597d19adfb9", b.TransactionID.String())
	assert.Equal(t, "847d3bfd-0742-49ef-b65d-3fed15f5b0a6", b.BatchID.String())
	assert.Equal(t, "d71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be", b.BatchHash.String())
	assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", b.BatchPayloadRef)
	assert.Equal(t, expectedSigningKeyRef, em.Calls[0].Arguments[2])
	assert.Len(t, b.Contexts, 2)
	assert.Equal(t, "68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a", b.Contexts[0].String())
	assert.Equal(t, "19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771", b.Contexts[1].String())

	info1 := fftypes.JSONObject{
		"address":          "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber":      "38011",
		"logIndex":         "50",
		"signature":        "0x1C197604587F046FD40684A8f21f4609FB811A7b:BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
		"subId":            "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"transactionHash":  "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"transactionIndex": "0x0",
		"timestamp":        "1620576488",
	}
	assert.Equal(t, info1, b.Event.Info)

	b2 := em.Calls[1].Arguments[1].(*blockchain.BatchPin)
	info2 := fftypes.JSONObject{
		"address":          "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber":      "38011",
		"logIndex":         "51",
		"signature":        "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
		"subId":            "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"transactionHash":  "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"transactionIndex": "0x1",
		"timestamp":        "1620576488",
	}
	assert.Equal(t, info2, b2.Event.Info)

}

func TestHandleMessageBatchPinMissingAddress(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"namespace": "ns1",
			"uuids": "0xe19af8b390604051812d7597d19adfb9847d3bfd074249efb65d3fed15f5b0a6",
			"batchHash": "0xd71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be",
			"payloadRef": "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
			"contexts": [
				"0x68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a",
				"0x19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771"
			]
    },
		"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
		"logIndex": "50",
		"timestamp": "1620576488"
  }
]`)

	em := &blockchainmocks.Callbacks{}

	e := &Ethereum{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.Regexp(t, "FF10141", err)

}

func TestHandleMessageBatchPinEmpty(t *testing.T) {
	e := &Ethereum{
		subs: common.NewFireflySubscriptions(),
	}
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)

	var events []interface{}
	err := json.Unmarshal([]byte(`
	[
		{
			"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
			"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
			"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])"
		}
	]`), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
}

func TestHandleMessageBatchMissingData(t *testing.T) {
	e := &Ethereum{
		subs: common.NewFireflySubscriptions(),
	}
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)

	var events []interface{}
	err := json.Unmarshal([]byte(`
	[
		{
			"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
			"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
			"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
			"timestamp": "1620576488"
		}
	]`), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
}

func TestHandleMessageBatchPinBadTransactionID(t *testing.T) {
	e := &Ethereum{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)

	data := fftypes.JSONAnyPtr(`[{
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
		"blockNumber": "38011",
		"transactionIndex": "0x1",
		"transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
			"author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"namespace": "ns1",
			"uuids": "!good",
			"batchHash": "0xd71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be",
			"payloadRef": "0xeda586bd8f3c4bc1db5c4b5755113b9a9b4174abe28679fdbc219129400dd7ae",
			"contexts": [
				"0xb41753f11522d4ef5c4a467972cf54744c04628ff84a1c994f1b288b2f6ec836",
				"0xc6c683a0fbe15e452e1ecc3751657446e2f645a8231e3ef9f3b4a8eae03c4136"
			]
		},
		"timestamp": "1620576488"
	}]`)
	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
}

func TestHandleMessageBatchPinBadIDentity(t *testing.T) {
	e := &Ethereum{
		subs: common.NewFireflySubscriptions(),
	}
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)

	data := fftypes.JSONAnyPtr(`[{
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
		"blockNumber": "38011",
		"transactionIndex": "0x1",
		"transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
			"author": "!good",
			"namespace": "ns1",
			"uuids": "0xe19af8b390604051812d7597d19adfb9847d3bfd074249efb65d3fed15f5b0a6",
			"batchHash": "0xd71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be",
			"payloadRef": "0xeda586bd8f3c4bc1db5c4b5755113b9a9b4174abe28679fdbc219129400dd7ae",
			"contexts": [
				"0xb41753f11522d4ef5c4a467972cf54744c04628ff84a1c994f1b288b2f6ec836",
				"0xc6c683a0fbe15e452e1ecc3751657446e2f645a8231e3ef9f3b4a8eae03c4136"
			]
		},
		"timestamp": "1620576488"
	}]`)
	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
}

func TestHandleMessageBatchBadJSON(t *testing.T) {
	e := &Ethereum{}
	err := e.handleMessageBatch(context.Background(), []interface{}{10, 20})
	assert.NoError(t, err)
}

func TestEventLoopContextCancelled(t *testing.T) {
	e, cancel := newTestEthereum()
	cancel()
	r := make(<-chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	wsm.On("Receive").Return(r)
	wsm.On("Close").Return()
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
	wsm.AssertExpectations(t)
}

func TestEventLoopReceiveClosed(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	close(r)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Close").Return()
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
	wsm.AssertExpectations(t)
}

func TestEventLoopSendClosed(t *testing.T) {
	e, cancel := newTestEthereum()
	s := make(chan []byte, 1)
	s <- []byte(`[]`)
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	wsm.On("Receive").Return((<-chan []byte)(s))
	wsm.On("Send", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		go cancel()
		close(r)
	}).Return(fmt.Errorf("pop"))
	wsm.On("Close").Return()
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
	wsm.AssertExpectations(t)
}

func TestHandleReceiptTXSuccess(t *testing.T) {
	em := &coremocks.OperationCallbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Ethereum{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: common.NewBlockchainCallbacks(),
		wsconn:    wsm,
	}
	e.SetOperationHandler("ns1", em)

	var reply fftypes.JSONObject
	operationID := fftypes.NewUUID()
	data := fftypes.JSONAnyPtr(`{
		"_id": "4373614c-e0f7-47b0-640e-7eacec417a9e",
		"blockHash": "0xad269b2b43481e44500f583108e8d24bd841fb767c7f526772959d195b9c72d5",
		"blockNumber": "209696",
		"cumulativeGasUsed": "24655",
		"from": "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
		"gasUsed": "24655",
		"headers": {
			"id": "4603a151-f212-446e-5c15-0f36b57cecc7",
			"requestId": "ns1:` + operationID.String() + `",
			"requestOffset": "zzn4y4v4si-zzjjepe9x4-requests:0:12",
			"timeElapsed": 3.966414429,
			"timeReceived": "2021-05-28T20:54:27.481245697Z",
			"type": "TransactionSuccess"
    },
		"nonce": "0",
		"receivedAt": 1622235271565,
		"status": "1",
		"to": "0xd3266a857285fb75eb7df37353b4a15c8bb828f5",
		"transactionHash": "0x71a38acb7a5d4a970854f6d638ceb1fa10a4b59cbf4ed7674273a1a8dc8b36b8",
		"transactionIndex": "0"
  }`)

	em.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == "ns1:"+operationID.String() &&
			update.Status == core.OpStatusSucceeded &&
			update.BlockchainTXID == "0x71a38acb7a5d4a970854f6d638ceb1fa10a4b59cbf4ed7674273a1a8dc8b36b8" &&
			update.Plugin == "ethereum"
	})).Return(nil)

	err := json.Unmarshal(data.Bytes(), &reply)
	assert.NoError(t, err)
	e.handleReceipt(context.Background(), reply)

	em.AssertExpectations(t)
}

func TestHandleReceiptTXUpdateEVMConnect(t *testing.T) {
	em := &coremocks.OperationCallbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Ethereum{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: common.NewBlockchainCallbacks(),
		wsconn:    wsm,
	}
	e.SetOperationHandler("ns1", em)

	var reply fftypes.JSONObject
	operationID := fftypes.NewUUID()
	data := fftypes.JSONAnyPtr(`{
		"created": "2022-08-03T18:55:42.671166Z",
		"errorHistory": null,
		"firstSubmit": "2022-08-03T18:55:42.762254Z",
		"gas": "48049",
		"gasPrice": 0,
		"headers": {
			"requestId": "ns1:` + operationID.String() + `",
			"type": "TransactionUpdate"
		},
		"id": "ns1:` + operationID.String() + `",
		"lastSubmit": "2022-08-03T18:55:42.762254Z",
		"nonce": "1",
		"policyInfo": null,
		"receipt": {
			"blockHash": "0x972713d879efd32573fe4d88ed0cde94a094367d50f7e5bc8262dd41fe07d9e6",
			"blockNumber": "3",
			"extraInfo": {
				"blockHash": "0x972713d879efd32573fe4d88ed0cde94a094367d50f7e5bc8262dd41fe07d9e6",
				"blockNumber": "0x3",
				"contractAddress": null,
				"cumulativeGasUsed": "0x7d21",
				"from": "0x081afaa6792a524ff2fb0654e615d19f9a600e57",
				"gasUsed": "0x7d21",
				"logs": [
					{
						"address": "0x9da7ecba282387ecd696dd9aecfc14efeb5f5fce",
						"blockHash": "0x972713d879efd32573fe4d88ed0cde94a094367d50f7e5bc8262dd41fe07d9e6",
						"blockNumber": "0x3",
						"data": "0x000000000000000000000000081afaa6792a524ff2fb0654e615d19f9a600e570000000000000000000000000000000000000000000000000000000062eac4ae00000000000000000000000000000000000000000000000000000000000000e05fbe3d02be9341f492917ec5aecfb151d243287ac95c49f780a4259ab53a8d14ec218052055a2b541712ab3246354c36f3aa189e1f31a7a8b33cbf957014d2e6000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002e516d50535551486f5852617553346a5434544b55356f414167616d334d4132315642575777634c6e66536e35696d00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000018319d70666de18ed7d1c17e1bbd8ac0b16716fbe2059a8ae356a09d3bd067e14",
						"logIndex": "0x0",
						"removed": false,
						"topics": [
							"0x805721bc246bccc732581be0c0aa2dd8f7ec93e97ba4b307be84428c98b0a12f"
						],
						"transactionHash": "0x929c898a46762d91e9f4b0b8e2800863dcf4a40f694109dc4cd19dbd334fa4cc",
						"transactionIndex": "0x0"
					}
				],
				"status": "0x1",
				"to": "0x9da7ecba282387ecd696dd9aecfc14efeb5f5fce",
				"transactionHash": "0x929c898a46762d91e9f4b0b8e2800863dcf4a40f694109dc4cd19dbd334fa4cc",
				"transactionIndex": "0x0"
			},
			"success": true,
			"transactionIndex": "0"
		},
		"sequenceId": "dfd7c5e6-135d-11ed-80ff-b67a78953577",
		"status": "Succeeded",
		"transactionData": "0x48ce1dcc5fbe3d02be9341f492917ec5aecfb151d243287ac95c49f780a4259ab53a8d14ec218052055a2b541712ab3246354c36f3aa189e1f31a7a8b33cbf957014d2e6000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000002e516d50535551486f5852617553346a5434544b55356f414167616d334d4132315642575777634c6e66536e35696d00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000018319d70666de18ed7d1c17e1bbd8ac0b16716fbe2059a8ae356a09d3bd067e14",
		"transactionHash": "0x929c898a46762d91e9f4b0b8e2800863dcf4a40f694109dc4cd19dbd334fa4cc",
		"transactionHeaders": {
			"from": "0x081afaa6792a524ff2fb0654e615d19f9a600e57",
			"to": "0x9da7ecba282387ecd696dd9aecfc14efeb5f5fce"
		},
		"updated": "2022-08-03T18:55:43.781941Z"
	}`)

	em.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == "ns1:"+operationID.String() &&
			update.Status == core.OpStatusPending &&
			update.BlockchainTXID == "0x929c898a46762d91e9f4b0b8e2800863dcf4a40f694109dc4cd19dbd334fa4cc" &&
			update.Plugin == "ethereum"
	})).Return(nil)

	err := json.Unmarshal(data.Bytes(), &reply)
	assert.NoError(t, err)
	e.handleReceipt(context.Background(), reply)

	em.AssertExpectations(t)
}

func TestHandleBadPayloadsAndThenReceiptFailure(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	e.closed = make(chan struct{})

	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Close").Return()
	operationID := fftypes.NewUUID()
	data := fftypes.JSONAnyPtr(`{
		"_id": "6fb94fff-81d3-4094-567d-e031b1871694",
		"errorMessage": "Packing arguments for method 'broadcastBatch': abi: cannot use [3]uint8 as type [32]uint8 as argument",
		"headers": {
			"id": "3a37b17b-13b6-4dc5-647a-07c11eae0be3",
			"requestId": "ns1:` + operationID.String() + `",
			"requestOffset": "zzn4y4v4si-zzjjepe9x4-requests:0:0",
			"timeElapsed": 0.020969053,
			"timeReceived": "2021-05-31T02:35:11.458880504Z",
			"type": "Error"
		},
		"receivedAt": 1622428511616,
		"requestPayload": "{\"from\":\"0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635\",\"gas\":0,\"gasPrice\":0,\"headers\":{\"id\":\"6fb94fff-81d3-4094-567d-e031b1871694\",\"type\":\"SendTransaction\"},\"method\":{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"txnId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"batchId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"payloadRef\",\"type\":\"bytes32\"}],\"name\":\"broadcastBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},\"params\":[\"12345\",\"!\",\"!\"],\"to\":\"0xd3266a857285fb75eb7df37353b4a15c8bb828f5\",\"value\":0}"
	}`)

	em := &coremocks.OperationCallbacks{}
	e.SetOperationHandler("ns1", em)
	txsu := em.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == "ns1:"+operationID.String() &&
			update.Status == core.OpStatusFailed &&
			update.ErrorMessage == "Packing arguments for method 'broadcastBatch': abi: cannot use [3]uint8 as type [32]uint8 as argument" &&
			update.Plugin == "ethereum"
	})).Return(fmt.Errorf("Shutdown"))
	done := make(chan struct{})
	txsu.RunFn = func(a mock.Arguments) {
		close(done)
	}

	go e.eventLoop()
	r <- []byte(`!badjson`)        // ignored bad json
	r <- []byte(`"not an object"`) // ignored wrong type
	r <- data.Bytes()
	<-done

	em.AssertExpectations(t)
}

func TestHandleMsgBatchBadData(t *testing.T) {
	wsm := &wsmocks.WSClient{}
	e := &Ethereum{
		ctx:    context.Background(),
		topic:  "topic1",
		wsconn: wsm,
	}

	var reply fftypes.JSONObject
	data := fftypes.JSONAnyPtr(`{}`)
	err := json.Unmarshal(data.Bytes(), &reply)
	assert.NoError(t, err)
	e.handleReceipt(context.Background(), reply)
}

func TestFormatNil(t *testing.T) {
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", ethHexFormatB32(nil))
}

func TestAddSubscription(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	e.streamID = "es-1"
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &core.ContractListenerInput{
		ContractListener: core.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &core.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "Changed",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: fftypes.JSONAnyPtr(`{"type": "string", "details": {"type": "string"}}`),
						},
					},
				},
			},
			Options: &core.ContractListenerOptions{
				FirstEvent: string(core.SubOptsFirstEventOldest),
			},
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/subscriptions`,
		httpmock.NewJsonResponderOrPanic(200, &subscription{}))

	err := e.AddContractListener(context.Background(), sub)

	assert.NoError(t, err)
}

func TestAddSubscriptionWithoutLocation(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	e.streamID = "es-1"
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &core.ContractListenerInput{
		ContractListener: core.ContractListener{
			Event: &core.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "Changed",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: fftypes.JSONAnyPtr(`{"type": "string", "details": {"type": "string"}}`),
						},
					},
				},
			},
			Options: &core.ContractListenerOptions{
				FirstEvent: string(core.SubOptsFirstEventOldest),
			},
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/subscriptions`,
		httpmock.NewJsonResponderOrPanic(200, &subscription{}))

	err := e.AddContractListener(context.Background(), sub)

	assert.NoError(t, err)
}

func TestAddSubscriptionBadParamDetails(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	e.streamID = "es-1"
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &core.ContractListenerInput{
		ContractListener: core.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &core.FFISerializedEvent{
				FFIEventDefinition: fftypes.FFIEventDefinition{
					Name: "Changed",
					Params: fftypes.FFIParams{
						{
							Name:   "value",
							Schema: fftypes.JSONAnyPtr(`{"type": "string", "details": {"type": ""}}`),
						},
					},
				},
			},
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/subscriptions`,
		httpmock.NewJsonResponderOrPanic(200, &subscription{}))

	err := e.AddContractListener(context.Background(), sub)

	assert.Regexp(t, "FF10311", err)
}

func TestAddSubscriptionBadLocation(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.streamID = "es-1"
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &core.ContractListenerInput{
		ContractListener: core.ContractListener{
			Location: fftypes.JSONAnyPtr(""),
			Event:    &core.FFISerializedEvent{},
		},
	}

	err := e.AddContractListener(context.Background(), sub)

	assert.Regexp(t, "FF10310", err)
}

func TestAddSubscriptionFail(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.streamID = "es-1"
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &core.ContractListenerInput{
		ContractListener: core.ContractListener{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
			Event: &core.FFISerializedEvent{},
			Options: &core.ContractListenerOptions{
				FirstEvent: string(core.SubOptsFirstEventNewest),
			},
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/subscriptions`,
		httpmock.NewStringResponder(500, "pop"))

	err := e.AddContractListener(context.Background(), sub)

	assert.Regexp(t, "FF10111", err)
	assert.Regexp(t, "pop", err)
}

func TestDeleteSubscription(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.streamID = "es-1"
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &core.ContractListener{
		BackendID: "sb-1",
	}

	httpmock.RegisterResponder("DELETE", `http://localhost:12345/subscriptions/sb-1`,
		httpmock.NewStringResponder(204, ""))

	err := e.DeleteContractListener(context.Background(), sub)

	assert.NoError(t, err)
}

func TestDeleteSubscriptionFail(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.streamID = "es-1"
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &core.ContractListener{
		BackendID: "sb-1",
	}

	httpmock.RegisterResponder("DELETE", `http://localhost:12345/subscriptions/sb-1`,
		httpmock.NewStringResponder(500, ""))

	err := e.DeleteContractListener(context.Background(), sub)

	assert.Regexp(t, "FF10111", err)
}

func TestHandleMessageContractEventOldSubscription(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"from": "0x91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"value": "1"
    },
		"subId": "sub2",
		"signature": "Changed(address,uint256)",
		"logIndex": "50",
		"timestamp": "1640811383"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sub2",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sub2", Stream: "es12345", Name: "ff-sub-1132312312312",
		}))

	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)
	e.streams = newTestStreamManager(e.client)

	em.On("BlockchainEvent", mock.MatchedBy(func(e *blockchain.EventWithSubscription) bool {
		assert.Equal(t, "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628", e.BlockchainTXID)
		assert.Equal(t, "000000038011/000000/000050", e.Event.ProtocolID)
		return true
	})).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	ev := em.Calls[0].Arguments[0].(*blockchain.EventWithSubscription)
	assert.Equal(t, "sub2", ev.Subscription)
	assert.Equal(t, "Changed", ev.Event.Name)

	outputs := fftypes.JSONObject{
		"from":  "0x91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
		"value": "1",
	}
	assert.Equal(t, outputs, ev.Event.Output)

	info := fftypes.JSONObject{
		"address":          "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber":      "38011",
		"logIndex":         "50",
		"signature":        "Changed(address,uint256)",
		"subId":            "sub2",
		"transactionHash":  "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"transactionIndex": "0x0",
		"timestamp":        "1640811383",
	}
	assert.Equal(t, info, ev.Event.Info)

	em.AssertExpectations(t)
}

func TestHandleMessageContractEventErrorOldSubscription(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"from": "0x91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"value": "1"
    },
		"subId": "sub2",
		"signature": "Changed(address,uint256)",
		"logIndex": "50",
		"timestamp": "1640811383"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sub2",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sub2", Stream: "es12345", Name: "ff-sub-1132312312312",
		}))

	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)
	e.streams = newTestStreamManager(e.client)
	em.On("BlockchainEvent", mock.Anything).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

	em.AssertExpectations(t)
}

func TestHandleMessageContractEventWithNamespace(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"from": "0x91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"value": "1"
    },
		"subId": "sub2",
		"signature": "Changed(address,uint256)",
		"logIndex": "50",
		"timestamp": "1640811383"
  },
	{
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72638",
		"data": {
			"from": "0x91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"value": "1"
    },
		"subId": "sub2",
		"signature": "Changed(address,uint256)",
		"logIndex": "50",
		"timestamp": "1640811384"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sub2",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sub2", Stream: "es12345", Name: "ff-sub-ns1-1132312312312",
		}))

	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)
	e.streams = newTestStreamManager(e.client)

	em.On("BlockchainEvent", mock.MatchedBy(func(e *blockchain.EventWithSubscription) bool {
		assert.Equal(t, "000000038011/000000/000050", e.Event.ProtocolID)
		return true
	})).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	ev := em.Calls[0].Arguments[0].(*blockchain.EventWithSubscription)
	assert.Equal(t, "sub2", ev.Subscription)
	assert.Equal(t, "Changed", ev.Event.Name)

	outputs := fftypes.JSONObject{
		"from":  "0x91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
		"value": "1",
	}
	assert.Equal(t, outputs, ev.Event.Output)

	info := fftypes.JSONObject{
		"address":          "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber":      "38011",
		"logIndex":         "50",
		"signature":        "Changed(address,uint256)",
		"subId":            "sub2",
		"transactionHash":  "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"transactionIndex": "0x0",
		"timestamp":        "1640811383",
	}
	assert.Equal(t, info, ev.Event.Info)

	em.AssertExpectations(t)
}

func TestHandleMessageContractEventNoNamespaceHandlers(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"from": "0x91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"value": "1"
    },
		"subId": "sub2",
		"signature": "Changed(address,uint256)",
		"logIndex": "50",
		"timestamp": "1640811383"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sub2",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sub2", Stream: "es12345", Name: "ff-sub-ns1-1132312312312",
		}))

	e.SetHandler("ns2", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)
	e.streams = newTestStreamManager(e.client)

	em.On("BlockchainEvent", mock.MatchedBy(func(e *blockchain.EventWithSubscription) bool {
		assert.Equal(t, "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628", e.BlockchainTXID)
		assert.Equal(t, "000000038011/000000/000050", e.Event.ProtocolID)
		return true
	})).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageContractEventSubNameError(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"from": "0x91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"value": "1"
    },
		"subId": "sub2",
		"signature": "Changed(address,uint256)",
		"logIndex": "50",
		"timestamp": "1640811383"
  }
]`)
	em := &blockchainmocks.Callbacks{}

	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sub2",
		httpmock.NewJsonResponderOrPanic(500, ethError{Error: "pop"}))

	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)
	e.streams = newTestStreamManager(e.client)

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.Regexp(t, "FF10111", err)

	em.AssertExpectations(t)
}

func TestHandleMessageContractEventError(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"from": "0x91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"value": "1"
    },
		"subId": "sub2",
		"signature": "Changed(address,uint256)",
		"logIndex": "50",
		"timestamp": "1640811383"
  }
]`)

	em := &blockchainmocks.Callbacks{}

	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sub2",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sub2", Stream: "es12345", Name: "ff-sub-ns1-1132312312312",
		}))

	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)
	e.streams = newTestStreamManager(e.client)

	em.On("BlockchainEvent", mock.Anything).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

	em.AssertExpectations(t)
}

func TestInvokeContractOK(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := ethHexFormatB32(fftypes.NewRandB32())
	location := &Location{
		Address: "0x12345",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": "1000000000000000000000000",
	}
	options := map[string]interface{}{
		"customOption": "customValue",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "SendTransaction", headers["type"])
			assert.Equal(t, float64(1), params[0])
			assert.Equal(t, "1000000000000000000000000", params[1])
			assert.Equal(t, body["customOption"].(string), "customValue")
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.NoError(t, err)
}

func TestInvokeContractInvalidOption(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := ethHexFormatB32(fftypes.NewRandB32())
	location := &Location{
		Address: "0x12345",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{
		"params": "shouldn't be allowed",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "SendTransaction", headers["type"])
			assert.Equal(t, float64(1), params[0])
			assert.Equal(t, "1000000000000000000000000", params[1])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF10398", err)
}

func TestInvokeContractInvalidInput(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := ethHexFormatB32(fftypes.NewRandB32())
	location := &Location{
		Address: "0x12345",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": map[bool]bool{true: false},
		"y": float64(2),
	}
	options := map[string]interface{}{
		"customOption": "customValue",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "SendTransaction", headers["type"])
			assert.Equal(t, float64(1), params[0])
			assert.Equal(t, float64(2), params[1])
			assert.Equal(t, body["customOption"].(string), "customValue")
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "unsupported type", err)
}

func TestInvokeContractAddressNotSet(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	signingKey := ethHexFormatB32(fftypes.NewRandB32())
	location := &Location{}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "'address' not set", err)
}

func TestInvokeContractEthconnectError(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := ethHexFormatB32(fftypes.NewRandB32())
	location := &Location{
		Address: "0x12345",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponderOrPanic(400, "")(req)
		})
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF10111", err)
}

func TestInvokeContractPrepareFail(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := ethHexFormatB32(fftypes.NewRandB32())
	location := &Location{
		Address: "0x12345",
	}
	method := &fftypes.FFIMethod{
		Name: "set",
		Params: fftypes.FFIParams{
			{
				Schema: fftypes.JSONAnyPtr("{bad schema!"),
			},
		},
	}
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "invalid json", err)
}

func TestQueryContractOK(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := &Location{
		Address: "0x12345",
	}
	method := testFFIMethod()
	params := map[string]interface{}{}
	options := map[string]interface{}{
		"customOption": "customValue",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "Query", headers["type"])
			assert.Equal(t, body["customOption"].(string), "customValue")
			return httpmock.NewJsonResponderOrPanic(200, queryOutput{Output: "3"})(req)
		})
	result, err := e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.NoError(t, err)
	j, err := json.Marshal(result)
	assert.NoError(t, err)
	assert.Equal(t, `{"output":"3"}`, string(j))
}

func TestQueryContractInvalidOption(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := &Location{
		Address: "0x12345",
	}
	method := testFFIMethod()
	params := map[string]interface{}{}
	options := map[string]interface{}{
		"params": "shouldn't be allowed",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "Query", headers["type"])
			return httpmock.NewJsonResponderOrPanic(200, queryOutput{Output: "3"})(req)
		})
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF10398", err)
}

func TestQueryContractErrorPrepare(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := &Location{
		Address: "0x12345",
	}
	method := &fftypes.FFIMethod{
		Params: fftypes.FFIParams{
			{
				Name:   "bad",
				Schema: fftypes.JSONAnyPtr("{badschema}"),
			},
		},
	}
	params := map[string]interface{}{}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "invalid json", err)
}

func TestQueryContractAddressNotSet(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	location := &Location{}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "'address' not set", err)
}

func TestQueryContractEthconnectError(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := &Location{
		Address: "0x12345",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponderOrPanic(400, queryOutput{})(req)
		})
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF10111", err)
}

func TestQueryContractUnmarshalResponseError(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := &Location{
		Address: "0x12345",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "Query", headers["type"])
			return httpmock.NewStringResponder(200, "[definitely not JSON}")(req)
		})
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "invalid character", err)
}

func TestNormalizeContractLocation(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	location := &Location{
		Address: "3081D84FD367044F4ED453F2024709242470388C",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	result, err := e.NormalizeContractLocation(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes))
	assert.NoError(t, err)
	assert.Equal(t, "0x3081d84fd367044f4ed453f2024709242470388c", result.JSONObject()["address"])
}

func TestNormalizeContractLocationInvalid(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	location := &Location{
		Address: "bad",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.NormalizeContractLocation(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes))
	assert.Regexp(t, "FF10141", err)
}

func TestNormalizeContractLocationBlank(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	location := &Location{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.NormalizeContractLocation(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes))
	assert.Regexp(t, "FF10310", err)
}

func TestGetContractAddressBadJSON(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/contracts/firefly",
		httpmock.NewBytesResponder(200, []byte("{not json!")),
	)

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	e.client = ffresty.New(e.ctx, utEthconnectConf)

	_, err := e.getContractAddress(context.Background(), "/contracts/firefly")

	assert.Regexp(t, "invalid character 'n' looking for beginning of object key string", err)
}

func TestGenerateFFI(t *testing.T) {
	e, _ := newTestEthereum()
	_, err := e.GenerateFFI(context.Background(), &fftypes.FFIGenerationRequest{
		Name:        "Simple",
		Version:     "v0.0.1",
		Description: "desc",
		Input:       fftypes.JSONAnyPtr(`{"abi": [{}]}`),
	})
	assert.NoError(t, err)
}

func TestGenerateFFIInlineNamespace(t *testing.T) {
	e, _ := newTestEthereum()
	ffi, err := e.GenerateFFI(context.Background(), &fftypes.FFIGenerationRequest{
		Name:        "Simple",
		Version:     "v0.0.1",
		Description: "desc",
		Namespace:   "ns1",
		Input:       fftypes.JSONAnyPtr(`{"abi":[{}]}`),
	})
	assert.NoError(t, err)
	assert.Equal(t, ffi.Namespace, "ns1")
}

func TestGenerateFFIEmptyABI(t *testing.T) {
	e, _ := newTestEthereum()
	_, err := e.GenerateFFI(context.Background(), &fftypes.FFIGenerationRequest{
		Name:        "Simple",
		Version:     "v0.0.1",
		Description: "desc",
		Input:       fftypes.JSONAnyPtr(`{"abi": []}`),
	})
	assert.Regexp(t, "FF10346", err)
}

func TestGenerateFFIBadABI(t *testing.T) {
	e, _ := newTestEthereum()
	_, err := e.GenerateFFI(context.Background(), &fftypes.FFIGenerationRequest{
		Name:        "Simple",
		Version:     "v0.0.1",
		Description: "desc",
		Input:       fftypes.JSONAnyPtr(`{"abi": "not an array"}`),
	})
	assert.Regexp(t, "FF10346", err)
}

func TestGenerateEventSignature(t *testing.T) {
	e, _ := newTestEthereum()
	complexParam := fftypes.JSONObject{
		"type": "object",
		"details": fftypes.JSONObject{
			"type": "tuple",
		},
		"properties": fftypes.JSONObject{
			"prop1": fftypes.JSONObject{
				"type": "integer",
				"details": fftypes.JSONObject{
					"type":  "uint256",
					"index": 0,
				},
			},
			"prop2": fftypes.JSONObject{
				"type": "integer",
				"details": fftypes.JSONObject{
					"type":  "uint256",
					"index": 1,
				},
			},
		},
	}.String()

	event := &fftypes.FFIEventDefinition{
		Name: "Changed",
		Params: []*fftypes.FFIParam{
			{
				Name:   "x",
				Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
			},
			{
				Name:   "y",
				Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`),
			},
			{
				Name:   "z",
				Schema: fftypes.JSONAnyPtr(complexParam),
			},
		},
	}

	signature := e.GenerateEventSignature(context.Background(), event)
	assert.Equal(t, "Changed(uint256,uint256,(uint256,uint256))", signature)
}

func TestGenerateEventSignatureInvalid(t *testing.T) {
	e, _ := newTestEthereum()
	event := &fftypes.FFIEventDefinition{
		Name: "Changed",
		Params: []*fftypes.FFIParam{
			{
				Name:   "x",
				Schema: fftypes.JSONAnyPtr(`{"!bad": "bad"`),
			},
		},
	}

	signature := e.GenerateEventSignature(context.Background(), event)
	assert.Equal(t, "", signature)
}

func TestSubmitNetworkAction(t *testing.T) {
	e, _ := newTestEthereum()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			res, err := mockNetworkVersion(t, 2)(req)
			if res != nil || err != nil {
				return res, err
			}

			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "SendTransaction", headers["type"])
			assert.Equal(t, "firefly:terminate", params[0])
			assert.Equal(t, "", params[1])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	err := e.SubmitNetworkAction(context.Background(), "ns1:"+fftypes.NewUUID().String(), "0x123", core.NetworkActionTerminate, location)
	assert.NoError(t, err)
}

func TestSubmitNetworkActionV1(t *testing.T) {
	e, _ := newTestEthereum()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			res, err := mockNetworkVersion(t, 1)(req)
			if res != nil || err != nil {
				return res, err
			}

			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "SendTransaction", headers["type"])
			assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", params[1])
			assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", params[2])
			assert.Equal(t, "", params[3])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	err := e.SubmitNetworkAction(context.Background(), "ns1:"+fftypes.NewUUID().String(), "0x123", core.NetworkActionTerminate, location)
	assert.NoError(t, err)
}

func TestSubmitNetworkActionBadLocation(t *testing.T) {
	e, _ := newTestEthereum()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "SendTransaction", headers["type"])
			assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", params[1])
			assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", params[2])
			assert.Equal(t, "", params[3])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"bad": "pop",
	}.String())

	err := e.SubmitNetworkAction(context.Background(), "ns1:"+fftypes.NewUUID().String(), "0x123", core.NetworkActionTerminate, location)
	assert.Regexp(t, "FF10310", err)
}

func TestSubmitNetworkActionVersionFail(t *testing.T) {
	e, _ := newTestEthereum()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		httpmock.NewJsonResponderOrPanic(500, ethError{Error: "unknown"}))

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	err := e.SubmitNetworkAction(context.Background(), "ns1:"+fftypes.NewUUID().String(), "0x123", core.NetworkActionTerminate, location)
	assert.Regexp(t, "FF10111", err)
}

func TestHandleNetworkAction(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"namespace": "firefly:terminate",
			"uuids": "0x0000000000000000000000000000000000000000000000000000000000000000",
			"batchHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
			"payloadRef": "",
			"contexts": []
    },
		"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
		"logIndex": "50",
		"timestamp": "1620576488"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)

	expectedSigningKeyRef := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
	}

	em.On("BlockchainNetworkAction", "terminate", mock.AnythingOfType("*fftypes.JSONAny"), mock.AnythingOfType("*blockchain.Event"), expectedSigningKeyRef).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	em.AssertExpectations(t)

}

func TestHandleNetworkActionFail(t *testing.T) {
	data := fftypes.JSONAnyPtr(`
[
  {
		"address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
		"blockNumber": "38011",
		"transactionIndex": "0x0",
		"transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
		"data": {
			"author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"namespace": "firefly:terminate",
			"uuids": "0x0000000000000000000000000000000000000000000000000000000000000000",
			"batchHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
			"payloadRef": "",
			"contexts": []
    },
		"subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
		"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
		"logIndex": "50",
		"timestamp": "1620576488"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", nil,
	)

	expectedSigningKeyRef := &core.VerifierRef{
		Type:  core.VerifierTypeEthAddress,
		Value: "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
	}

	em.On("BlockchainNetworkAction", "terminate", mock.AnythingOfType("*fftypes.JSONAny"), mock.AnythingOfType("*blockchain.Event"), expectedSigningKeyRef).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data.Bytes(), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

	em.AssertExpectations(t)
}

func TestGetFFIParamValidator(t *testing.T) {
	e, _ := newTestEthereum()
	v, err := e.GetFFIParamValidator(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)
}

func TestGetNetworkVersion(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "Query", headers["type"])
			return httpmock.NewJsonResponderOrPanic(200, queryOutput{Output: "1"})(req)
		})

	version, err := e.GetNetworkVersion(context.Background(), location)
	assert.NoError(t, err)
	assert.Equal(t, 1, version)

	// second call is cached
	version, err = e.GetNetworkVersion(context.Background(), location)
	assert.NoError(t, err)
	assert.Equal(t, 1, version)

	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

func TestGetNetworkVersionBadFormat(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "Query", headers["type"])
			return httpmock.NewJsonResponderOrPanic(200, queryOutput{Output: nil})(req)
		})

	_, err := e.GetNetworkVersion(context.Background(), location)
	assert.Regexp(t, "FF10412", err)
}

func TestGetNetworkVersionMethodNotFound(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	httpmock.RegisterResponder("POST", "http://localhost:12345/",
		httpmock.NewJsonResponderOrPanic(500, ethError{Error: "FFEC100148"}))

	version, err := e.GetNetworkVersion(context.Background(), location)

	assert.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestGetNetworkVersionQueryFail(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	httpmock.RegisterResponder("POST", "http://localhost:12345/",
		httpmock.NewJsonResponderOrPanic(500, ethError{Error: "pop"}))

	version, err := e.GetNetworkVersion(context.Background(), location)

	assert.Regexp(t, "pop", err)
	assert.Equal(t, 0, version)
}

func TestGetNetworkVersionBadLocation(t *testing.T) {
	e, _ := newTestEthereum()
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"bad": "pop",
	}.String())

	version, err := e.GetNetworkVersion(context.Background(), location)

	assert.Regexp(t, "FF10310", err)
	assert.Equal(t, 0, version)
}

func TestGetNetworkVersionUnmarshalFail(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "Query", headers["type"])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	version, err := e.GetNetworkVersion(context.Background(), location)

	assert.Regexp(t, "cannot unmarshal", err)
	assert.Equal(t, 0, version)
}

func TestConvertDeprecatedContractConfig(t *testing.T) {
	e, _ := newTestEthereum()
	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/0x71c7656ec7ab88b098defb751b7401b5f6d8976f")

	locationBytes, fromBlock, err := e.GetAndConvertDeprecatedContractConfig(e.ctx)
	assert.Equal(t, "0", fromBlock)
	assert.NoError(t, err)

	location, err := parseContractLocation(e.ctx, locationBytes)
	assert.NoError(t, err)

	assert.Equal(t, "0x71c7656ec7ab88b098defb751b7401b5f6d8976f", location.Address)
}

func TestConvertDeprecatedContractConfigNoInstance(t *testing.T) {
	e, _ := newTestEthereum()
	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	_, _, err := e.GetAndConvertDeprecatedContractConfig(e.ctx)
	assert.Regexp(t, "F10138", err)
}

func TestConvertDeprecatedContractConfigContractURL(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/contracts/firefly",
		httpmock.NewJsonResponderOrPanic(200, map[string]string{
			"created":      "2022-02-08T22:10:10Z",
			"address":      "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
			"path":         "/contracts/firefly",
			"abi":          "fc49dec3-0660-4dc7-61af-65af4c3ac456",
			"openapi":      "/contracts/firefly?swagger",
			"registeredAs": "firefly",
		}),
	)

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/contracts/firefly")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	e.client = ffresty.New(e.ctx, utEthconnectConf)

	locationBytes, fromBlock, err := e.GetAndConvertDeprecatedContractConfig(e.ctx)
	assert.NoError(t, err)
	assert.Equal(t, "0", fromBlock)

	location, err := parseContractLocation(e.ctx, locationBytes)
	assert.NoError(t, err)

	assert.Equal(t, "0x71c7656ec7ab88b098defb751b7401b5f6d8976f", location.Address)
}

func TestConvertDeprecatedContractConfigContractURLBadQuery(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/contracts/firefly",
		httpmock.NewJsonResponderOrPanic(500, ethError{Error: "FFEC100148"}))

	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/contracts/firefly")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	e.client = ffresty.New(e.ctx, utEthconnectConf)

	_, _, err := e.GetAndConvertDeprecatedContractConfig(e.ctx)
	assert.Regexp(t, "FFEC100148", err)
}

func TestConvertDeprecatedContractConfigBadAddress(t *testing.T) {
	e, _ := newTestEthereum()
	resetConf(e)
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")
	utEthconnectConf.Set(EthconnectConfigInstanceDeprecated, "/instances/bad")

	_, _, err := e.GetAndConvertDeprecatedContractConfig(e.ctx)
	assert.Regexp(t, "FF10141", err)
}

func TestAddSubBadLocation(t *testing.T) {
	e, _ := newTestEthereum()
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"bad": "bad",
	}.String())

	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err := e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.Regexp(t, "FF10310", err)
}

func TestAddAndRemoveFireflySubscription(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sub1",
		}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 2))

	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	originalContext := e.ctx
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	cmi.AssertCalled(t, "GetCache", cache.NewCacheConfig(
		originalContext,
		coreconfig.CacheBlockchainLimit,
		coreconfig.CacheBlockchainTTL,
		"",
	))
	assert.NoError(t, err)
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	subID, err := e.AddFireflySubscription(e.ctx, ns, location, "newest")
	assert.NoError(t, err)
	assert.NotNil(t, e.subs.GetSubscription("sub1"))

	e.RemoveFireflySubscription(e.ctx, subID)
	assert.Nil(t, e.subs.GetSubscription("sub1"))
}

func TestAddFireflySubscriptionV1(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sub1",
		}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 1))

	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "newest")
	assert.NoError(t, err)
	assert.NotNil(t, e.subs.GetSubscription("sub1"))
}

func TestAddFireflySubscriptionQuerySubsFail(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewStringResponder(500, `pop`))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 2))

	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.Regexp(t, "FF10111", err)
}

func TestAddFireflySubscriptionCreateError(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/",
		httpmock.NewJsonResponderOrPanic(500, `pop`))

	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.Regexp(t, "FF10111", err)
}

func TestAddFireflySubscriptionGetVersionError(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(500, `pop`))
	httpmock.RegisterResponder("POST", "http://localhost:12345/", mockNetworkVersion(t, 2))

	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String())

	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.Regexp(t, "FF10111", err)
}

func TestGetContractListenerStatus(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	checkpoint := ListenerCheckpoint{
		Block:            0,
		TransactionIndex: -1,
		LogIndex:         -1,
	}

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sub1",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sub1", Stream: "es12345", Name: "ff-sub-1132312312312", subscriptionCheckpoint: subscriptionCheckpoint{
				Catchup:    false,
				Checkpoint: checkpoint,
			},
		}))

	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)

	status, err := e.GetContractListenerStatus(context.Background(), "sub1")
	assert.NotNil(t, status)
	assert.NoError(t, err)
}

func TestGetContractListenerStatusGetSubFail(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf(e)

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sub1",
		httpmock.NewJsonResponderOrPanic(500, `pop`))

	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, e.metrics, cmi)
	assert.NoError(t, err)

	status, err := e.GetContractListenerStatus(context.Background(), "sub1")
	assert.Nil(t, status)
	assert.Regexp(t, "FF10111", err)
}
