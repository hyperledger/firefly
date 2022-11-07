// Copyright © 2022 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//		 http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fabric

import (
	"context"
	"encoding/base64"
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

var utConfig = config.RootSection("fab_unit_tests")
var utFabconnectConf = utConfig.SubSection(FabconnectConfigKey)
var signer = "orgMSP::x509::CN=signer001,OU=client::CN=fabric-ca"

func resetConf(e *Fabric) {
	coreconfig.Reset()
	e.InitConfig(utConfig)
}

func newTestFabric() (*Fabric, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:            ctx,
		cancelCtx:      cancel,
		client:         resty.New().SetBaseURL("http://localhost:12345"),
		defaultChannel: "firefly",
		topic:          "topic1",
		prefixShort:    defaultPrefixShort,
		prefixLong:     defaultPrefixLong,
		wsconn:         wsm,
		cache:          cache.NewUmanagedCache(ctx, 100, 5*time.Minute),
		callbacks:      common.NewBlockchainCallbacks(),
		subs:           common.NewFireflySubscriptions(),
	}
	return e, func() {
		cancel()
		if e.closed != nil {
			// We've init'd, wait to close
			<-e.closed
		}
	}
}

func newTestStreamManager(client *resty.Client, signer string) *streamManager {
	return newStreamManager(client, signer, cache.NewUmanagedCache(context.Background(), 100, 5*time.Minute))
}

func testFFIMethod() *fftypes.FFIMethod {
	return &fftypes.FFIMethod{
		Name: "sum",
		Params: []*fftypes.FFIParam{
			{
				Name:   "x",
				Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
			},
			{
				Name:   "y",
				Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
			},
			{
				Name:   "description",
				Schema: fftypes.JSONAnyPtr(`{"type": "string"}`),
			},
		},
		Returns: []*fftypes.FFIParam{
			{
				Name:   "z",
				Schema: fftypes.JSONAnyPtr(`{"type": "integer"}`),
			},
		},
	}
}

func mockNetworkVersion(t *testing.T, version float64) func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		var body map[string]interface{}
		json.NewDecoder(req.Body).Decode(&body)
		if body["func"] == "NetworkVersion" {
			return httpmock.NewJsonResponderOrPanic(200, fabQueryNamedOutput{
				Result: version,
			})(req)
		}
		return nil, nil
	}
}

func TestInitMissingURL(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf(e)

	cmi := &cachemocks.Manager{}
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.Regexp(t, "FF10138.*url", err)
}

func TestInitMissingTopic(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "Firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.Regexp(t, "FF10138.*topic", err)
}

func TestInitAllNewStreamsAndWSEvent(t *testing.T) {

	log.SetLevel("trace")
	e, cancel := newTestFabric()
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
	utFabconnectConf.Set(ffresty.HTTPConfigURL, httpURL)
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	originalContext := e.ctx
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	cmi.AssertCalled(t, "GetCache", cache.NewCacheConfig(
		originalContext,
		coreconfig.CacheBlockchainLimit,
		coreconfig.CacheBlockchainTTL,
		"",
	))
	assert.NoError(t, err)

	assert.Equal(t, "fabric", e.Name())
	assert.Equal(t, core.VerifierTypeMSPIdentity, e.VerifierType())

	assert.NoError(t, err)
	err = e.Start()
	assert.NoError(t, err)

	assert.Equal(t, 2, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
	assert.NotNil(t, e.Capabilities())

	startupMessage := <-toServer
	assert.Equal(t, `{"type":"listen","topic":"topic1"}`, startupMessage)
	startupMessage = <-toServer
	assert.Equal(t, `{"type":"listenreplies"}`, startupMessage)
	fromServer <- `[]` // empty batch, will be ignored, but acked
	reply := <-toServer
	assert.Equal(t, `{"topic":"topic1","type":"ack"}`, reply)

	// Bad data will be ignored
	fromServer <- `!json`
	fromServer <- `{"not": "a reply"}`
	fromServer <- `42`

}

func TestWSInitFail(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "!!!://")
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.Regexp(t, "FF00149", err)

}

func TestCacheInitFail(t *testing.T) {
	cacheInitError := errors.New("Initialization error.")
	e, cancel := newTestFabric()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Stream: "es12345", Name: "ns1_BatchPin"},
		}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 2))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")
	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(nil, cacheInitError)

	defer cancel()
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.Equal(t, cacheInitError, err)
}

func TestInitAllExistingStreams(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Stream: "es12345", Name: "ns1_BatchPin"},
		}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 2))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.NoError(t, err)

	assert.Equal(t, 3, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
}

func TestInitAllExistingStreamsV1(t *testing.T) {
	e, cancel := newTestFabric()
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
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.NoError(t, err)

	assert.Equal(t, 3, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
}

func TestAddFireflySubscriptionQuerySubsFail(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf(e)

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(500, "pop"))
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "newest")
	assert.Regexp(t, "pop", err)
}

func TestAddFireflySubscriptionGetVersionError(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf(e)

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Stream: "es12345", Name: "ns1_BatchPin"},
		}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		httpmock.NewJsonResponderOrPanic(500, "pop"))

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "newest")
	assert.Regexp(t, "pop", err)
}

func TestAddAndRemoveFireflySubscriptionDeprecatedSubName(t *testing.T) {
	e, cancel := newTestFabric()
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
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	subID, err := e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.NoError(t, err)

	assert.Equal(t, 3, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
	assert.NotNil(t, e.subs.GetSubscription(subID))

	e.RemoveFireflySubscription(e.ctx, subID)
	assert.Nil(t, e.subs.GetSubscription(subID))
}

func TestAddFireflySubscriptionInvalidSubName(t *testing.T) {
	e, cancel := newTestFabric()
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
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 2))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.Regexp(t, "FF10416", err)
}

func TestAddFFSubscriptionBadLocation(t *testing.T) {
	e, _ := newTestFabric()
	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"bad": "bad",
	}.String())
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err := e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.Regexp(t, "F10310", err)
}

func TestInitNewConfig(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", Name: "topic1"}}))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.NoError(t, err)

	assert.Equal(t, 1, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.streamID)
}

func TestStreamQueryError(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewStringResponder(500, `pop`))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.Regexp(t, "FF10284.*pop", err)

}

func TestStreamCreateError(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewStringResponder(500, `pop`))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.Regexp(t, "FF10284.*pop", err)

}

func TestSubQueryCreateError(t *testing.T) {

	e, cancel := newTestFabric()
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
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.Regexp(t, "FF10284.*pop", err)

}

func TestSubQueryCreate(t *testing.T) {

	e, cancel := newTestFabric()
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
		httpmock.NewJsonResponderOrPanic(200, subscription{ID: "sb-123"}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	cmi := &cachemocks.Manager{}
	cmi.On("GetCache", mock.Anything).Return(cache.NewUmanagedCache(e.ctx, 100, 5*time.Minute), nil)
	err := e.Init(e.ctx, e.cancelCtx, utConfig, &metricsmocks.Manager{}, cmi)
	assert.NoError(t, err)
	ns := &core.Namespace{Name: "ns1", NetworkName: "ns1"}
	_, err = e.AddFireflySubscription(e.ctx, ns, location, "oldest")
	assert.NoError(t, err)

}

func TestSubmitBatchPinOK(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
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
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 2))

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, (body["headers"].(map[string]interface{}))["signer"])
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", (body["args"].(map[string]interface{}))["uuids"])
			assert.Equal(t, hexFormatB32(batch.BatchHash), (body["args"].(map[string]interface{}))["batchHash"])
			assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", (body["args"].(map[string]interface{}))["payloadRef"])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	err := e.SubmitBatchPin(context.Background(), "", "ns1", signer, batch, location)

	assert.NoError(t, err)

}

func TestSubmitBatchPinV1(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
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
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, (body["headers"].(map[string]interface{}))["signer"])
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", (body["args"].(map[string]interface{}))["uuids"])
			assert.Equal(t, hexFormatB32(batch.BatchHash), (body["args"].(map[string]interface{}))["batchHash"])
			assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", (body["args"].(map[string]interface{}))["payloadRef"])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	err := e.SubmitBatchPin(context.Background(), "", "ns1", signer, batch, location)

	assert.NoError(t, err)

}

func TestSubmitBatchPinBadLocation(t *testing.T) {

	e, _ := newTestFabric()

	signer := "signer001"
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
		"bad": "simplestorage",
	}.String())

	err := e.SubmitBatchPin(context.Background(), "", "ns1", signer, batch, location)

	assert.Regexp(t, "FF10310", err)
}

func TestSubmitBatchEmptyPayloadRef(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
	batch := &blockchain.BatchPin{
		TransactionID: fftypes.MustParseUUID("9ffc50ff-6bfe-4502-adc7-93aea54cc059"),
		BatchID:       fftypes.MustParseUUID("c5df767c-fe44-4e03-8eb5-1c5523097db5"),
		BatchHash:     fftypes.NewRandB32(),
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, (body["headers"].(map[string]interface{}))["signer"])
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", (body["args"].(map[string]interface{}))["uuids"])
			assert.Equal(t, hexFormatB32(batch.BatchHash), (body["args"].(map[string]interface{}))["batchHash"])
			assert.Equal(t, "", (body["args"].(map[string]interface{}))["payloadRef"])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	err := e.SubmitBatchPin(context.Background(), "", "ns1", signer, batch, location)

	assert.NoError(t, err)

}

func TestSubmitBatchPinVersionFail(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
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
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		httpmock.NewStringResponder(500, "pop"))

	err := e.SubmitBatchPin(context.Background(), "", "ns1", signer, batch, location)

	assert.Regexp(t, "FF10284.*pop", err)

}

func TestSubmitBatchPinFail(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
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
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		httpmock.NewStringResponder(500, "pop"))

	err := e.SubmitBatchPin(context.Background(), "", "ns1", signer, batch, location)

	assert.Regexp(t, "FF10284.*pop", err)

}

func TestSubmitBatchPinError(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
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
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{
			"error": "Invalid",
		}))

	err := e.SubmitBatchPin(context.Background(), "", "ns1", signer, batch, location)

	assert.Regexp(t, "FF10284.*Invalid", err)

}

func TestResolveFullIDSigner(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()

	id := "org1MSP::x509::CN=admin,OU=client::CN=fabric-ca-server"
	signKey, err := e.NormalizeSigningKey(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "org1MSP::x509::CN=admin,OU=client::CN=fabric-ca-server", signKey)

}

func TestResolveSigner(t *testing.T) {
	e, cancel := newTestFabric()
	e.idCache = make(map[string]*fabIdentity)
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	res := make(map[string]string)
	res["name"] = "signer001"
	res["mspId"] = "org1MSP"
	res["enrollmentCert"] = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJ4ekNDQVcyZ0F3SUJBZ0lVTUpXdUJBcHl0eXdNVU13cC82b3o3Qk0wS3dJd0NnWUlLb1pJemowRUF3SXcKR3pFWk1CY0dBMVVFQXhNUVptRmljbWxqTFdOaExYTmxjblpsY2pBZUZ3MHlNVEEzTWpreE5UUXdNREJhRncweQpNakEzTWpreE5UUTFNREJhTUNFeER6QU5CZ05WQkFzVEJtTnNhV1Z1ZERFT01Bd0dBMVVFQXhNRllXUnRhVzR3CldUQVRCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFUTUxMR2VwR2oyWEo3aWFhU1hXWXBpSGtCc3RqbXUKcStzd3hIOTdxWi9vS0JWMHFoa21kcUlkTmNNaTdwNHNYQzM1NTN6Nm5DUHpqSWtjQzdqWi9IVDBvNEdJTUlHRgpNQTRHQTFVZER3RUIvd1FFQXdJQkJqQU1CZ05WSFJNQkFmOEVBakFBTUIwR0ExVWREZ1FXQkJRZUdkWDNVdUxMCnZWVHpDVkdwcVVJQjFFdEhMREFmQmdOVkhTTUVHREFXZ0JUcTdoVzQ5Yno0WjAyK2YyM3hVSGxCbzd5eGFqQWwKQmdOVkhSRUVIakFjZ2hwTFlXeGxhV1J2Y3kxTllXTkNiMjlyTFZCeWJ5NXNiMk5oYkRBS0JnZ3Foa2pPUFFRRApBZ05JQURCRkFpRUF1bzVtbGh6UXc4RnIrcUFhUzAxcCsxTlVaNEF5ZmdQb21kQ2RKTzJUYXJRQ0lIUG1pTUhuCk9jekc5cS9kT3NiQUQ1c3dZbWcyTEZpM05mQkswK0cvUC9TUAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	res["caCert"] = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJmVENDQVNPZ0F3SUJBZ0lVYndac0FnK2Zac0FmSUF2VWFlWXBpOXF3NG9jd0NnWUlLb1pJemowRUF3SXcKR3pFWk1CY0dBMVVFQXhNUVptRmljbWxqTFdOaExYTmxjblpsY2pBZUZ3MHlNVEEzTWpNd01URTRNREJhRncwegpOakEzTVRrd01URTRNREJhTUJzeEdUQVhCZ05WQkFNVEVHWmhZbkpwWXkxallTMXpaWEoyWlhJd1dUQVRCZ2NxCmhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUlZNajcyR1dTeXk1UjRQN084ckpidXkrNHd6NWJWSE94dHBxRlUKamNadVE0Q2VSUGJoNDF3KzR1dFJsTlRTbFhLdTBMblBlVEZLSjlRT00xd0xwTGJtbzBVd1F6QU9CZ05WSFE4QgpBZjhFQkFNQ0FRWXdFZ1lEVlIwVEFRSC9CQWd3QmdFQi93SUJBREFkQmdOVkhRNEVGZ1FVNnU0VnVQVzgrR2ROCnZuOXQ4VkI1UWFPOHNXb3dDZ1lJS29aSXpqMEVBd0lEU0FBd1JRSWhBTzRod085UjB2Z3htMUphaGdTOWJnajQKZm9JNmc1QnRrUzRKcmgvc0ZpbzlBaUFRVVhnTUhXYzZSMVZhTHpXTkx0U0tkbHMvWTFuM3Z5MnlPZE1PL1Y4cApCZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

	responder, _ := httpmock.NewJsonResponder(200, res)
	httpmock.RegisterResponder("GET", `http://localhost:12345/identities/signer001`, responder)
	resolved, err := e.NormalizeSigningKey(context.Background(), "signer001")
	assert.NoError(t, err)
	assert.Equal(t, "org1MSP::x509::CN=admin,OU=client::CN=fabric-ca-server", resolved)
}

func TestResolveSignerFailedFabricCARequest(t *testing.T) {
	e, cancel := newTestFabric()
	e.idCache = make(map[string]*fabIdentity)
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	res := make(map[string]string)

	responder, _ := httpmock.NewJsonResponder(503, res)
	httpmock.RegisterResponder("GET", `http://localhost:12345/identities/signer001`, responder)
	_, err := e.NormalizeSigningKey(context.Background(), "signer001")
	assert.EqualError(t, err, "FF10284: Error from fabconnect: %!!(MISSING)s(<nil>)")
}

func TestResolveSignerBadECertReturned(t *testing.T) {
	e, cancel := newTestFabric()
	e.idCache = make(map[string]*fabIdentity)
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	res := make(map[string]string)
	res["name"] = "signer001"
	res["mspId"] = "org1MSP"
	res["enrollmentCert"] = base64.StdEncoding.EncodeToString([]byte(badCert))
	res["caCert"] = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJmVENDQVNPZ0F3SUJBZ0lVYndac0FnK2Zac0FmSUF2VWFlWXBpOXF3NG9jd0NnWUlLb1pJemowRUF3SXcKR3pFWk1CY0dBMVVFQXhNUVptRmljbWxqTFdOaExYTmxjblpsY2pBZUZ3MHlNVEEzTWpNd01URTRNREJhRncwegpOakEzTVRrd01URTRNREJhTUJzeEdUQVhCZ05WQkFNVEVHWmhZbkpwWXkxallTMXpaWEoyWlhJd1dUQVRCZ2NxCmhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUlZNajcyR1dTeXk1UjRQN084ckpidXkrNHd6NWJWSE94dHBxRlUKamNadVE0Q2VSUGJoNDF3KzR1dFJsTlRTbFhLdTBMblBlVEZLSjlRT00xd0xwTGJtbzBVd1F6QU9CZ05WSFE4QgpBZjhFQkFNQ0FRWXdFZ1lEVlIwVEFRSC9CQWd3QmdFQi93SUJBREFkQmdOVkhRNEVGZ1FVNnU0VnVQVzgrR2ROCnZuOXQ4VkI1UWFPOHNXb3dDZ1lJS29aSXpqMEVBd0lEU0FBd1JRSWhBTzRod085UjB2Z3htMUphaGdTOWJnajQKZm9JNmc1QnRrUzRKcmgvc0ZpbzlBaUFRVVhnTUhXYzZSMVZhTHpXTkx0U0tkbHMvWTFuM3Z5MnlPZE1PL1Y4cApCZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

	responder, _ := httpmock.NewJsonResponder(200, res)
	httpmock.RegisterResponder("GET", `http://localhost:12345/identities/signer001`, responder)
	_, err := e.NormalizeSigningKey(context.Background(), "signer001")
	assert.Contains(t, err.Error(), "FF10286: Failed to decode certificate:")
}

func TestResolveSignerBadCACertReturned(t *testing.T) {
	e, cancel := newTestFabric()
	e.idCache = make(map[string]*fabIdentity)
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	res := make(map[string]string)
	res["name"] = "signer001"
	res["mspId"] = "org1MSP"
	res["enrollmentCert"] = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJmVENDQVNPZ0F3SUJBZ0lVYndac0FnK2Zac0FmSUF2VWFlWXBpOXF3NG9jd0NnWUlLb1pJemowRUF3SXcKR3pFWk1CY0dBMVVFQXhNUVptRmljbWxqTFdOaExYTmxjblpsY2pBZUZ3MHlNVEEzTWpNd01URTRNREJhRncwegpOakEzTVRrd01URTRNREJhTUJzeEdUQVhCZ05WQkFNVEVHWmhZbkpwWXkxallTMXpaWEoyWlhJd1dUQVRCZ2NxCmhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUlZNajcyR1dTeXk1UjRQN084ckpidXkrNHd6NWJWSE94dHBxRlUKamNadVE0Q2VSUGJoNDF3KzR1dFJsTlRTbFhLdTBMblBlVEZLSjlRT00xd0xwTGJtbzBVd1F6QU9CZ05WSFE4QgpBZjhFQkFNQ0FRWXdFZ1lEVlIwVEFRSC9CQWd3QmdFQi93SUJBREFkQmdOVkhRNEVGZ1FVNnU0VnVQVzgrR2ROCnZuOXQ4VkI1UWFPOHNXb3dDZ1lJS29aSXpqMEVBd0lEU0FBd1JRSWhBTzRod085UjB2Z3htMUphaGdTOWJnajQKZm9JNmc1QnRrUzRKcmgvc0ZpbzlBaUFRVVhnTUhXYzZSMVZhTHpXTkx0U0tkbHMvWTFuM3Z5MnlPZE1PL1Y4cApCZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
	res["caCert"] = base64.StdEncoding.EncodeToString([]byte(badCert))

	responder, _ := httpmock.NewJsonResponder(200, res)
	httpmock.RegisterResponder("GET", `http://localhost:12345/identities/signer001`, responder)
	_, err := e.NormalizeSigningKey(context.Background(), "signer001")
	assert.Contains(t, err.Error(), "FF10286: Failed to decode certificate:")
}

func TestGetUserNameWithMatches(t *testing.T) {
	result := getUserName(signer)
	assert.Equal(t, result, "signer001")
}

func TestGetUserNameNoMatches(t *testing.T) {
	result := getUserName("orgMSP::x509::OU=client::OU=CA")
	assert.Equal(t, result, "")
}

func TestHandleMessageBatchPinOK(t *testing.T) {
	data := []byte(`
[
  {
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"transactionIndex": 2,
		"eventIndex": 50,
		"eventName": "BatchPin",
		"payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMxNjY3LCJuYW5vcyI6NzkxNDk5MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIjB4MTliODIwOTNkZTVjZTkyYTAxZTMzMzA0OGU4NzdlMjM3NDM1NGJmODQ2ZGQwMzQ4NjRlZjZmZmJkNjQzODc3MSJdfQ==",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  },
  {
		"chaincodeId": "firefly",
		"blockNumber": 77,
		"transactionIndex": 0,
		"eventIndex": 0,
		"transactionId": "a488800a70c8f765871611168d422fb29cc37da2d0a196a3200c8068ba1706fd",
		"eventName": "BatchPin",
		"payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMxNjY3LCJuYW5vcyI6NzkxNDk5MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIjB4MTliODIwOTNkZTVjZTkyYTAxZTMzMzA0OGU4NzdlMjM3NDM1NGJmODQ2ZGQwMzQ4NjRlZjZmZmJkNjQzODc3MSJdfQ==",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }	
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e", "firefly",
	)

	expectedSigningKeyRef := &core.VerifierRef{
		Type:  core.VerifierTypeMSPIdentity,
		Value: "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server",
	}

	em.On("BatchPinComplete", "ns1", mock.Anything, expectedSigningKeyRef).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	b := em.Calls[0].Arguments[1].(*blockchain.BatchPin)
	assert.Equal(t, "e19af8b3-9060-4051-812d-7597d19adfb9", b.TransactionID.String())
	assert.Equal(t, "847d3bfd-0742-49ef-b65d-3fed15f5b0a6", b.BatchID.String())
	assert.Equal(t, "d71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be", b.BatchHash.String())
	assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", b.BatchPayloadRef)
	assert.Equal(t, expectedSigningKeyRef, em.Calls[1].Arguments[2])
	assert.Len(t, b.Contexts, 2)
	assert.Equal(t, "68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a", b.Contexts[0].String())
	assert.Equal(t, "19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771", b.Contexts[1].String())

	em.AssertExpectations(t)

}

func TestHandleMessageBatchPinMissingChaincodeID(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte(fftypes.JSONObject{
		"signer":     "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server",
		"timestamp":  fftypes.JSONObject{"seconds": 1630031667, "nanos": 791499000},
		"namespace":  "ns1",
		"uuids":      "0xe19af8b390604051812d7597d19adfb9847d3bfd074249efb65d3fed15f5b0a6",
		"batchHash":  "0xd71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be",
		"payloadRef": "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		"contexts":   []string{"0x68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a", "0x19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771"},
	}.String()))
	data := []byte(`
[
  {
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"transactionIndex": 2,
		"eventIndex": 50,
		"eventName": "BatchPin",
		"payload": "` + payload + `",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e", "firefly",
	)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.Regexp(t, "FF10310", err)

}

func TestHandleMessageUnknownEventName(t *testing.T) {
	data := []byte(`
[
  {
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"eventName": "UnknownEvent",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e", "firefly",
	)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadPayloadEncoding(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e", "firefly",
	)

	data := []byte(`[{
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"eventName": "BatchPin",
		"payload": "--eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMzMzQ0LCJuYW5vcyI6OTY1NjE4MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIiFnb29kIl19",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
	}]`)
	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchBadJSON(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: common.NewBlockchainCallbacks(),
	}
	e.SetHandler("ns1", em)
	err := e.handleMessageBatch(context.Background(), []interface{}{10, 20})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestEventLoopContextCancelled(t *testing.T) {
	e, cancel := newTestFabric()
	cancel()
	r := make(<-chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	wsm.On("Receive").Return(r)
	wsm.On("Close").Return()
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopReceiveClosed(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	close(r)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Close").Return()
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopSendClosed(t *testing.T) {
	e, cancel := newTestFabric()
	s := make(chan []byte, 1)
	s <- []byte(`[]`)
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	wsm.On("Receive").Return((<-chan []byte)(s))
	wsm.On("Close").Return()
	wsm.On("Send", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Run(func(args mock.Arguments) {
		go cancel()
		close(r)
	})
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
	wsm.AssertExpectations(t)
}

func TestEventLoopUnexpectedMessage(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Close").Return()
	e.closed = make(chan struct{})
	operationID := fftypes.NewUUID()
	data := []byte(`{
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
	em := &blockchainmocks.Callbacks{}
	e.SetHandler("ns1", em)
	txsu := em.On("BlockchainOpUpdate",
		e,
		"ns1:"+operationID.String(),
		core.OpStatusFailed,
		"",
		"Packing arguments for method 'broadcastBatch': abi: cannot use [3]uint8 as type [32]uint8 as argument",
		mock.Anything).Return(fmt.Errorf("Shutdown"))
	done := make(chan struct{})
	txsu.RunFn = func(a mock.Arguments) {
		close(done)
	}

	go e.eventLoop()
	r <- []byte(`!badjson`)        // ignored bad json
	r <- []byte(`"not an object"`) // ignored wrong type
	r <- data
	e.ctx.Done()
}

func TestHandleReceiptTXSuccess(t *testing.T) {
	em := &coremocks.OperationCallbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
		wsconn:    wsm,
	}
	e.SetOperationHandler("ns1", em)

	var reply fftypes.JSONObject
	operationID := fftypes.NewUUID()
	data := []byte(`{
		"_id": "748e7587-9e72-4244-7351-808f69b88291",
		"headers": {
				"id": "0ef91fb6-09c5-4ca2-721c-74b4869097c2",
				"requestId": "ns1:` + operationID.String() + `",
				"requestOffset": "",
				"timeElapsed": 0.475721,
				"timeReceived": "2021-08-27T03:04:34.199742Z",
				"type": "TransactionSuccess"
		},
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"receivedAt": 1630033474675
  }`)

	em.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == "ns1:"+operationID.String() &&
			update.Status == core.OpStatusSucceeded &&
			update.BlockchainTXID == "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2" &&
			update.Plugin == "fabric"
	})).Return(nil)

	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	e.handleReceipt(context.Background(), reply)

	em.AssertExpectations(t)
}

func TestHandleReceiptNoRequestID(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
		wsconn:    wsm,
	}
	e.SetHandler("ns1", em)

	var reply fftypes.JSONObject
	data := []byte(`{}`)
	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	e.handleReceipt(context.Background(), reply)
}

func TestHandleReceiptFailedTx(t *testing.T) {
	em := &coremocks.OperationCallbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
		wsconn:    wsm,
	}
	e.SetOperationHandler("ns1", em)

	var reply fftypes.JSONObject
	operationID := fftypes.NewUUID()
	data := []byte(`{
		"_id": "748e7587-9e72-4244-7351-808f69b88291",
		"headers": {
				"id": "0ef91fb6-09c5-4ca2-721c-74b4869097c2",
				"requestId": "ns1:` + operationID.String() + `",
				"requestOffset": "",
				"timeElapsed": 0.475721,
				"timeReceived": "2021-08-27T03:04:34.199742Z",
				"type": "TransactionFailure"
		},
		"receivedAt": 1630033474675,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2"
  }`)

	em.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == "ns1:"+operationID.String() &&
			update.Status == core.OpStatusFailed &&
			update.BlockchainTXID == "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2" &&
			update.Plugin == "fabric"
	})).Return(nil)

	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	e.handleReceipt(context.Background(), reply)

	em.AssertExpectations(t)
}

func TestFormatNil(t *testing.T) {
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", hexFormatB32(nil))
}

func TestAddSubscription(t *testing.T) {
	e, cancel := newTestFabric()
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
				"channel":   "firefly",
				"chaincode": "mycode",
			}.String()),
			Event: &core.FFISerializedEvent{},
			Options: &core.ContractListenerOptions{
				FirstEvent: string(core.SubOptsFirstEventOldest),
			},
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/subscriptions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "0", body["fromBlock"])
			return httpmock.NewJsonResponderOrPanic(200, &subscription{})(req)
		})

	err := e.AddContractListener(context.Background(), sub)

	assert.NoError(t, err)
}

func TestAddSubscriptionNoChannel(t *testing.T) {
	e, cancel := newTestFabric()
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
				"chaincode": "mycode",
			}.String()),
			Event: &core.FFISerializedEvent{},
			Options: &core.ContractListenerOptions{
				FirstEvent: string(core.SubOptsFirstEventOldest),
			},
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/subscriptions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "0", body["fromBlock"])
			return httpmock.NewJsonResponderOrPanic(200, &subscription{})(req)
		})

	err := e.AddContractListener(context.Background(), sub)

	assert.Regexp(t, "FF10310.*channel", err)
}

func TestAddSubscriptionNoLocation(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.streamID = "es-1"
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &core.ContractListenerInput{
		ContractListener: core.ContractListener{
			Event: &core.FFISerializedEvent{},
			Options: &core.ContractListenerOptions{
				FirstEvent: string(core.SubOptsFirstEventOldest),
			},
		},
	}

	err := e.AddContractListener(context.Background(), sub)

	assert.Regexp(t, "FF10310.*channel", err)
}

func TestAddSubscriptionBadLocation(t *testing.T) {
	e, cancel := newTestFabric()
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
	e, cancel := newTestFabric()
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
				"channel":   "firefly",
				"chaincode": "mycode",
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

	assert.Regexp(t, "FF10284.*pop", err)
}

func TestDeleteSubscription(t *testing.T) {
	e, cancel := newTestFabric()
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
	e, cancel := newTestFabric()
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
		httpmock.NewStringResponder(500, "pop"))

	err := e.DeleteContractListener(context.Background(), sub)

	assert.Regexp(t, "FF10284.*pop", err)
}

func TestHandleMessageContractEventOldSubscription(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
		"blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"transactionIndex": 20,
		"eventIndex": 30,
		"eventName": "AssetCreated",
		"payload": "eyJBcHByYWlzZWRWYWx1ZSI6MTAsIkNvbG9yIjoicmVkIiwiSUQiOiIxMjM0IiwiT3duZXIiOiJtZSIsIlNpemUiOjN9",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	em := &blockchainmocks.Callbacks{}
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sb-cb37cc07-e873-4f58-44ab-55add6bba320", Stream: "es12345", Name: "old-sub-name",
		}))

	e.streams = newTestStreamManager(e.client, e.signer)
	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns1", em)

	em.On("BlockchainEvent", mock.MatchedBy(func(e *blockchain.EventWithSubscription) bool {
		assert.Equal(t, "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d", e.BlockchainTXID)
		assert.Equal(t, "000000000010/000020/000030", e.Event.ProtocolID)
		return true
	})).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	ev := em.Calls[0].Arguments[0].(*blockchain.EventWithSubscription)
	assert.Equal(t, "sb-cb37cc07-e873-4f58-44ab-55add6bba320", ev.Subscription)
	assert.Equal(t, "AssetCreated", ev.Event.Name)

	outputs := fftypes.JSONObject{
		"AppraisedValue": float64(10),
		"Color":          "red",
		"ID":             "1234",
		"Owner":          "me",
		"Size":           float64(3),
	}
	assert.Equal(t, outputs, ev.Event.Output)

	info := fftypes.JSONObject{
		"blockNumber":      float64(10),
		"chaincodeId":      "basic",
		"eventName":        "AssetCreated",
		"subId":            "sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		"transactionId":    "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"transactionIndex": float64(20),
		"eventIndex":       float64(30),
	}
	assert.Equal(t, info, ev.Event.Info)

	em.AssertExpectations(t)
}

func TestHandleMessageContractEventNamespacedHandlers(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
	  "blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"transactionIndex": 20,
		"eventIndex": 30,
		"eventName": "AssetCreated",
		"payload": "eyJBcHByYWlzZWRWYWx1ZSI6MTAsIkNvbG9yIjoicmVkIiwiSUQiOiIxMjM0IiwiT3duZXIiOiJtZSIsIlNpemUiOjN9",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	},
	{
		"chaincodeId": "basic",
	  "blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746f",
		"transactionIndex": 20,
		"eventIndex": 30,
		"eventName": "AssetCreated",
		"payload": "eyJBcHByYWlzZWRWYWx1ZSI6MTAsIkNvbG9yIjoicmVkIiwiSUQiOiIxMjM0IiwiT3duZXIiOiJtZSIsIlNpemUiOjN9",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	em := &blockchainmocks.Callbacks{}
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sb-cb37cc07-e873-4f58-44ab-55add6bba320", Stream: "es12345", Name: "ff-sub-ns1-11232312312",
		}))

	e.streams = newTestStreamManager(e.client, e.signer)
	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns1", em)

	em.On("BlockchainEvent", mock.MatchedBy(func(e *blockchain.EventWithSubscription) bool {
		assert.Equal(t, "000000000010/000020/000030", e.Event.ProtocolID)
		return true
	})).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	ev := em.Calls[0].Arguments[0].(*blockchain.EventWithSubscription)
	assert.Equal(t, "sb-cb37cc07-e873-4f58-44ab-55add6bba320", ev.Subscription)
	assert.Equal(t, "AssetCreated", ev.Event.Name)

	outputs := fftypes.JSONObject{
		"AppraisedValue": float64(10),
		"Color":          "red",
		"ID":             "1234",
		"Owner":          "me",
		"Size":           float64(3),
	}
	assert.Equal(t, outputs, ev.Event.Output)

	info := fftypes.JSONObject{
		"blockNumber":      float64(10),
		"chaincodeId":      "basic",
		"eventName":        "AssetCreated",
		"subId":            "sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		"transactionId":    "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"transactionIndex": float64(20),
		"eventIndex":       float64(30),
	}
	assert.Equal(t, info, ev.Event.Info)

	em.AssertExpectations(t)
}

func TestHandleMessageContractEventNoNamespacedHandlers(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
	  "blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"transactionIndex": 20,
		"eventIndex": 30,
		"eventName": "AssetCreated",
		"payload": "eyJBcHByYWlzZWRWYWx1ZSI6MTAsIkNvbG9yIjoicmVkIiwiSUQiOiIxMjM0IiwiT3duZXIiOiJtZSIsIlNpemUiOjN9",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	em := &blockchainmocks.Callbacks{}
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sb-cb37cc07-e873-4f58-44ab-55add6bba320", Stream: "es12345", Name: "ff-sub-ns1-11232312312",
		}))

	e.streams = newTestStreamManager(e.client, e.signer)
	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns2", em)

	em.On("BlockchainEvent", mock.MatchedBy(func(e *blockchain.EventWithSubscription) bool {
		assert.Equal(t, "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d", e.BlockchainTXID)
		assert.Equal(t, "000000000010/000020/000030", e.Event.ProtocolID)
		return true
	})).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageContractEventNoPayload(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
	  "blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"transactionIndex": 20,
		"eventIndex": 30,
		"eventName": "AssetCreated",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sb-cb37cc07-e873-4f58-44ab-55add6bba320", Stream: "es12345", Name: "ff-sub-ns1-11232312312",
		}))

	em := &blockchainmocks.Callbacks{}
	e.streams = newTestStreamManager(e.client, e.signer)
	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns1", em)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
}

func TestHandleMessageContractOldSubError(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
		"blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"eventName": "AssetCreated",
		"payload": "eyJBcHByYWlzZWRWYWx1ZSI6MTAsIkNvbG9yIjoicmVkIiwiSUQiOiIxMjM0IiwiT3duZXIiOiJtZSIsIlNpemUiOjN9",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sb-cb37cc07-e873-4f58-44ab-55add6bba320", Stream: "es12345", Name: "oldsubname",
		}))

	em := &blockchainmocks.Callbacks{}
	e.streams = newTestStreamManager(e.client, e.signer)
	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", "firefly",
	)

	em.On("BlockchainEvent", mock.Anything).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

	em.AssertExpectations(t)
}

func TestHandleMessageContractEventError(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
		"blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"eventName": "AssetCreated",
		"payload": "eyJBcHByYWlzZWRWYWx1ZSI6MTAsIkNvbG9yIjoicmVkIiwiSUQiOiIxMjM0IiwiT3duZXIiOiJtZSIsIlNpemUiOjN9",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		httpmock.NewJsonResponderOrPanic(200, subscription{
			ID: "sb-cb37cc07-e873-4f58-44ab-55add6bba320", Stream: "es12345", Name: "ff-sub-ns1-11232312312",
		}))

	em := &blockchainmocks.Callbacks{}
	e.streams = newTestStreamManager(e.client, e.signer)
	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns1", em)

	em.On("BlockchainEvent", mock.Anything).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

	em.AssertExpectations(t)
}

func TestHandleMessageContractGetSubError(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
		"blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"eventName": "AssetCreated",
		"payload": "eyJBcHByYWlzZWRWYWx1ZSI6MTAsIkNvbG9yIjoicmVkIiwiSUQiOiIxMjM0IiwiT3duZXIiOiJtZSIsIlNpemUiOjN9",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions/sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		httpmock.NewJsonResponderOrPanic(500, fabError{Error: "pop"}))

	em := &blockchainmocks.Callbacks{}
	e.streams = newTestStreamManager(e.client, e.signer)
	e.callbacks = common.NewBlockchainCallbacks()
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-b5b97a4e-a317-4053-6400-1474650efcb5", "firefly",
	)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.Regexp(t, "FF10284", err)

	em.AssertExpectations(t)
}

func TestInvokeContractOK(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x":           float64(1),
		"y":           float64(2),
		"description": "test",
	}
	options := map[string]interface{}{
		"customOption": "customValue",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signingKey, (body["headers"].(map[string]interface{}))["signer"])
			assert.Equal(t, "firefly", (body["headers"].(map[string]interface{}))["channel"])
			assert.Equal(t, "simplestorage", (body["headers"].(map[string]interface{}))["chaincode"])
			assert.Equal(t, "1", body["args"].(map[string]interface{})["x"])
			assert.Equal(t, "2", body["args"].(map[string]interface{})["y"])
			assert.Equal(t, "test", body["args"].(map[string]interface{})["description"])
			assert.Equal(t, "customValue", body["customOption"])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.NoError(t, err)
}

func TestDeployContractOK(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	input := []interface{}{
		float64(1),
		"1000000000000000000000000",
	}
	options := map[string]interface{}{
		"contract": "not really a contract",
	}
	definitionBytes, err := json.Marshal([]interface{}{})
	contractBytes, err := json.Marshal("0x123456")
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			params := body["params"].([]interface{})
			headers := body["headers"].(map[string]interface{})
			assert.Equal(t, "DeployContract", headers["type"])
			assert.Equal(t, float64(1), params[0])
			assert.Equal(t, "1000000000000000000000000", params[1])
			assert.Equal(t, body["customOption"].(string), "customValue")
			return httpmock.NewJsonResponderOrPanic(400, "pop")(req)
		})
	err = e.DeployContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(definitionBytes), fftypes.JSONAnyPtrBytes(contractBytes), input, options)
	assert.Regexp(t, "FF10429", err)
}

func TestInvokeContractBadSchema(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	method := &fftypes.FFIMethod{
		Name: "sum",
		Params: []*fftypes.FFIParam{
			{
				Name:   "x",
				Schema: fftypes.JSONAnyPtr(`{not json]`),
			},
		},
		Returns: []*fftypes.FFIParam{},
	}
	params := map[string]interface{}{
		"x":           float64(1),
		"y":           float64(2),
		"description": "test",
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF00127", err)
}

func TestInvokeContractInvalidOption(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{
		"func": "foobar",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF10398", err)
}

func TestInvokeContractChaincodeNotSet(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	signingKey := fftypes.NewRandB32().String()
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
	assert.Regexp(t, "FF10310", err)
}

func TestInvokeContractFabconnectError(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{
		Channel:   "fabric",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponderOrPanic(400, "")(req)
		})
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF10284", err)
}

func TestQueryContractOK(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	e.signer = signingKey
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{
		"customOption": "customValue",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/query`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signingKey, body["headers"].(map[string]interface{})["signer"])
			assert.Equal(t, "firefly", body["headers"].(map[string]interface{})["channel"])
			assert.Equal(t, "simplestorage", body["headers"].(map[string]interface{})["chaincode"])
			assert.Equal(t, "1", body["args"].(map[string]interface{})["x"])
			assert.Equal(t, "2", body["args"].(map[string]interface{})["y"])
			assert.Equal(t, "customValue", body["customOption"])
			return httpmock.NewJsonResponderOrPanic(200, &fabQueryNamedOutput{})(req)
		})
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.NoError(t, err)
}

func TestQueryContractInputNotJSON(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	e.signer = signingKey
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"bad": map[interface{}]interface{}{true: false},
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF00127", err)
}

func TestQueryContractBadLocation(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	e.signer = signingKey
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	_, err := e.QueryContract(context.Background(), fftypes.JSONAnyPtr(`{"validLocation": false}`), method, params, options)
	assert.Regexp(t, "FF10310", err)
}

func TestQueryContractFabconnectError(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := &Location{
		Channel:   "fabric",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/query`,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponderOrPanic(400, &fabQueryNamedOutput{})(req)
		})
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF10284", err)
}

func TestQueryContractUnmarshalResponseError(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/query`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "firefly", (body["headers"].(map[string]interface{}))["channel"])
			assert.Equal(t, "simplestorage", (body["headers"].(map[string]interface{}))["chaincode"])
			assert.Equal(t, "1", body["args"].(map[string]interface{})["x"])
			assert.Equal(t, "2", body["args"].(map[string]interface{})["y"])
			return httpmock.NewStringResponder(200, "[definitely not JSON}")(req)
		})
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "invalid character", err)
}

func TestNormalizeContractLocation(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.NormalizeContractLocation(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes))
	assert.NoError(t, err)
}

func TestNormalizeContractLocationNoChannel(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	location := &Location{
		Chaincode: "simplestorage",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.NormalizeContractLocation(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes))
	assert.Regexp(t, "FF10310.*channel", err)
}

func TestValidateNoContractLocationChaincode(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	location := &Location{
		Channel: "firefly",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.NormalizeContractLocation(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes))
	assert.Regexp(t, "FF10310", err)
}

func TestInvokeJSONEncodeParamsError(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{
		Channel:   "fabric",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": map[bool]interface{}{true: false},
		"y": float64(2),
	}
	options := map[string]interface{}{}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponderOrPanic(400, "")(req)
		})
	err = e.InvokeContract(context.Background(), "", signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params, options)
	assert.Regexp(t, "FF00127", err)
}

func TestGetFFIParamValidator(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	_, err := e.GetFFIParamValidator(context.Background())
	assert.NoError(t, err)
}

func TestGenerateFFI(t *testing.T) {
	e, _ := newTestFabric()
	_, err := e.GenerateFFI(context.Background(), &fftypes.FFIGenerationRequest{
		Name:        "Simple",
		Version:     "v0.0.1",
		Description: "desc",
		Input:       fftypes.JSONAnyPtr(`[]`),
	})
	assert.Regexp(t, "FF10347", err)
}

func TestGenerateEventSignature(t *testing.T) {
	e, _ := newTestFabric()
	signature := e.GenerateEventSignature(context.Background(), &fftypes.FFIEventDefinition{Name: "Changed"})
	assert.Equal(t, "Changed", signature)
}

func TestHandleNetworkAction(t *testing.T) {
	data := []byte(`
[
  {
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"transactionIndex": 2,
		"eventIndex": 50,
		"eventName": "BatchPin",
		"payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMxNjY3LCJuYW5vcyI6NzkxNDk5MDAwfSwibmFtZXNwYWNlIjoiZmlyZWZseTp0ZXJtaW5hdGUiLCJ1dWlkcyI6IjB4MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMCIsImJhdGNoSGFzaCI6IjB4MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMCIsInBheWxvYWRSZWYiOiIiLCJjb250ZXh0cyI6W119",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e", "firefly",
	)

	expectedSigningKeyRef := &core.VerifierRef{
		Type:  core.VerifierTypeMSPIdentity,
		Value: "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server",
	}

	em.On("BlockchainNetworkAction", "terminate", mock.AnythingOfType("*fftypes.JSONAny"), mock.AnythingOfType("*blockchain.Event"), expectedSigningKeyRef).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	em.AssertExpectations(t)

}

func TestHandleNetworkActionFail(t *testing.T) {
	data := []byte(`
[
  {
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"transactionIndex": 2,
		"eventIndex": 50,
		"eventName": "BatchPin",
		"payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMxNjY3LCJuYW5vcyI6NzkxNDk5MDAwfSwibmFtZXNwYWNlIjoiZmlyZWZseTp0ZXJtaW5hdGUiLCJ1dWlkcyI6IjB4MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMCIsImJhdGNoSGFzaCI6IjB4MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMCIsInBheWxvYWRSZWYiOiIiLCJjb250ZXh0cyI6W119",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: common.NewBlockchainCallbacks(),
		subs:      common.NewFireflySubscriptions(),
	}
	e.SetHandler("ns1", em)
	e.subs.AddSubscription(
		context.Background(),
		&core.Namespace{Name: "ns1", NetworkName: "ns1"},
		1, "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e", "firefly",
	)

	expectedSigningKeyRef := &core.VerifierRef{
		Type:  core.VerifierTypeMSPIdentity,
		Value: "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server",
	}

	em.On("BlockchainNetworkAction", "terminate", mock.AnythingOfType("*fftypes.JSONAny"), mock.AnythingOfType("*blockchain.Event"), expectedSigningKeyRef).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

	em.AssertExpectations(t)

}

func TestGetNetworkVersion(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

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
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		httpmock.NewJsonResponderOrPanic(200, fabQueryNamedOutput{Result: nil}))

	_, err := e.GetNetworkVersion(context.Background(), location)
	assert.Regexp(t, "FF10412", err)
}

func TestGetNetworkVersionFunctionNotFound(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", "http://localhost:12345/query",
		httpmock.NewJsonResponderOrPanic(500, fabError{Error: "Function NetworkVersion not found"}))

	version, err := e.GetNetworkVersion(context.Background(), location)

	assert.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestGetNetworkVersionBadLocation(t *testing.T) {
	e, _ := newTestFabric()

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"bad": "no good",
	}.String())

	version, err := e.GetNetworkVersion(context.Background(), location)

	assert.Regexp(t, "FF10310", err)
	assert.Equal(t, 0, version)
}

func TestGetNetworkVersionBadResponse(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", "http://localhost:12345/query",
		httpmock.NewJsonResponderOrPanic(200, ""))

	version, err := e.GetNetworkVersion(context.Background(), location)

	assert.Regexp(t, "json: cannot unmarshal", err)
	assert.Equal(t, 0, version)
}

func TestGetNetworkVersionFunctionNotFoundQueryFail(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	httpmock.RegisterResponder("POST", "http://localhost:12345/query",
		httpmock.NewJsonResponderOrPanic(500, fabError{Error: "pop"}))

	version, err := e.GetNetworkVersion(context.Background(), location)

	assert.Regexp(t, "pop", err)
	assert.Equal(t, 0, version)
}

func TestConvertDeprecatedContractConfig(t *testing.T) {
	e, _ := newTestFabric()
	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "Firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")

	locationBytes, fromBlock, err := e.GetAndConvertDeprecatedContractConfig(e.ctx)
	assert.Equal(t, "oldest", fromBlock)
	assert.NoError(t, err)

	location, err := parseContractLocation(e.ctx, locationBytes)
	assert.NoError(t, err)

	assert.Equal(t, "Firefly", location.Chaincode)
}

func TestConvertDeprecatedContractConfigNoChaincode(t *testing.T) {
	e, _ := newTestFabric()
	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")

	_, _, err := e.GetAndConvertDeprecatedContractConfig(e.ctx)
	assert.Regexp(t, "F10138", err)
}

func TestConvertDeprecatedContractConfigNoChannel(t *testing.T) {
	e, _ := newTestFabric()
	resetConf(e)
	utFabconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(FabconnectConfigChaincodeDeprecated, "Firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")

	e.defaultChannel = ""
	_, _, err := e.GetAndConvertDeprecatedContractConfig(e.ctx)
	assert.Regexp(t, "FF10310", err)
}

func TestSubmitNetworkAction(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, (body["headers"].(map[string]interface{}))["signer"])
			assert.Equal(t, "\"firefly:terminate\"", (body["args"].(map[string]interface{}))["action"])
			assert.Equal(t, "", (body["args"].(map[string]interface{}))["payload"])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 2))

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	err := e.SubmitNetworkAction(context.Background(), "", signer, core.NetworkActionTerminate, location)
	assert.NoError(t, err)
}

func TestSubmitNetworkActionV1(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, (body["headers"].(map[string]interface{}))["signer"])
			assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", (body["args"].(map[string]interface{}))["uuids"])
			assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", (body["args"].(map[string]interface{}))["batchHash"])
			assert.Equal(t, "", (body["args"].(map[string]interface{}))["payloadRef"])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		mockNetworkVersion(t, 1))

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	err := e.SubmitNetworkAction(context.Background(), "", signer, core.NetworkActionTerminate, location)
	assert.NoError(t, err)
}

func TestSubmitNetworkActionBadLocation(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, (body["headers"].(map[string]interface{}))["signer"])
			assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", (body["args"].(map[string]interface{}))["uuids"])
			assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", (body["args"].(map[string]interface{}))["batchHash"])
			assert.Equal(t, "", (body["args"].(map[string]interface{}))["payloadRef"])
			return httpmock.NewJsonResponderOrPanic(200, "")(req)
		})

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"bad": "location",
	}.String())

	err := e.SubmitNetworkAction(context.Background(), "", signer, core.NetworkActionTerminate, location)
	assert.Regexp(t, "FF10310", err)
}

func TestSubmitNetworkActionVersionError(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"

	httpmock.RegisterResponder("POST", fmt.Sprintf("http://localhost:12345/query"),
		httpmock.NewJsonResponderOrPanic(500, "pop"))

	location := fftypes.JSONAnyPtr(fftypes.JSONObject{
		"channel":   "firefly",
		"chaincode": "simplestorage",
	}.String())

	err := e.SubmitNetworkAction(context.Background(), "", signer, core.NetworkActionTerminate, location)
	assert.Regexp(t, "FF10284", err)
}

func TestGetContractListenerStatus(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	status, err := e.GetContractListenerStatus(context.Background(), "id")
	assert.Nil(t, status)
	assert.NoError(t, err)
}
