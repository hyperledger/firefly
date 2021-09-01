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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/restclient"
	"github.com/hyperledger-labs/firefly/internal/wsclient"
	"github.com/hyperledger-labs/firefly/mocks/blockchainmocks"
	"github.com/hyperledger-labs/firefly/mocks/wsmocks"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("eth_unit_tests")
var utFabconnectConf = utConfPrefix.SubPrefix(FabconnectConfigKey)

func resetConf() {
	config.Reset()
	e := &Fabric{}
	e.InitPrefix(utConfPrefix)
}

func newTestFabric() (*Fabric, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:            ctx,
		client:         resty.New().SetHostURL("http://localhost:12345"),
		defaultChannel: "firefly",
		chaincode:      "firefly",
		topic:          "topic1",
		prefixShort:    defaultPrefixShort,
		prefixLong:     defaultPrefixLong,
		callbacks:      em,
		wsconn:         wsm,
	}
	return e, func() {
		cancel()
		if e.closed != nil {
			// We've init'd, wait to close
			<-e.closed
		}
	}
}

func TestInitMissingURL(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf()
	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10138.*url", err)
}

func TestInitMissingChaincode(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10138.*chaincode", err)
}

func TestInitMissingTopic(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(FabconnectConfigChaincode, "Firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
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
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/subscriptions", httpURL),
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/subscriptions", httpURL),
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "es12345", body["stream"])
			return httpmock.NewJsonResponderOrPanic(200, subscription{ID: "sub12345"})(req)
		})

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, httpURL)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.NoError(t, err)

	assert.Equal(t, "fabric", e.Name())
	assert.Equal(t, 4, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.initInfo.stream.ID)
	assert.Equal(t, "sub12345", e.initInfo.subs[0].ID)
	assert.True(t, e.Capabilities().GlobalSequencer)

	err = e.Start()
	assert.NoError(t, err)

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

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "!!!://")
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")
	utFabconnectConf.Set(FabconnectConfigSkipEventstreamInit, true)

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10162", err)

}

func TestWSConnectFail(t *testing.T) {

	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:    context.Background(),
		wsconn: wsm,
	}
	wsm.On("Connect").Return(fmt.Errorf("pop"))

	err := e.Start()
	assert.EqualError(t, err, "pop")
}

func TestInitAllExistingStreams(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", WebSocket: eventStreamWebsocket{Topic: "topic1"}}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Name: "BatchPin"},
		}))

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Equal(t, 2, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.initInfo.stream.ID)
	assert.Equal(t, "sub12345", e.initInfo.subs[0].ID)

	assert.NoError(t, err)

}

func TestStreamQueryError(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewStringResponder(500, `pop`))

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10277", err)
	assert.Regexp(t, "pop", err)

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

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10277", err)
	assert.Regexp(t, "pop", err)

}

func TestSubQueryError(t *testing.T) {

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
		httpmock.NewStringResponder(500, `pop`))

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10277", err)
	assert.Regexp(t, "pop", err)

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

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10277", err)
	assert.Regexp(t, "pop", err)

}

func TestSubmitBatchPinOK(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
	batch := &blockchain.BatchPin{
		TransactionID:  fftypes.MustParseUUID("9ffc50ff-6bfe-4502-adc7-93aea54cc059"),
		BatchID:        fftypes.MustParseUUID("c5df767c-fe44-4e03-8eb5-1c5523097db5"),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, req.FormValue(defaultPrefixShort+"-signer"))
			assert.Equal(t, "false", req.FormValue(defaultPrefixShort+"-sync"))
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", (body["args"].([]interface{}))[1])
			assert.Equal(t, hexFormatB32(batch.BatchHash), (body["args"].([]interface{}))[2])
			assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", (body["args"].([]interface{}))[3])
			return httpmock.NewJsonResponderOrPanic(200, asyncTXSubmission{})(req)
		})

	err := e.SubmitBatchPin(context.Background(), nil, &fftypes.Identity{OnChain: signer}, batch)

	assert.NoError(t, err)

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

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, req.FormValue("fly-signer"))
			assert.Equal(t, "false", req.FormValue("fly-sync"))
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", (body["args"].([]interface{}))[1])
			assert.Equal(t, hexFormatB32(batch.BatchHash), (body["args"].([]interface{}))[2])
			assert.Equal(t, "", (body["args"].([]interface{}))[3])
			return httpmock.NewJsonResponderOrPanic(200, asyncTXSubmission{})(req)
		})

	err := e.SubmitBatchPin(context.Background(), nil, &fftypes.Identity{OnChain: signer}, batch)

	assert.NoError(t, err)

}

func TestSubmitBatchPinFail(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
	batch := &blockchain.BatchPin{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchHash:      fftypes.NewRandB32(),
		BatchPaylodRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		httpmock.NewStringResponder(500, "pop"))

	err := e.SubmitBatchPin(context.Background(), nil, &fftypes.Identity{OnChain: signer}, batch)

	assert.Regexp(t, "FF10277", err)
	assert.Regexp(t, "pop", err)

}

func TestVerifySigner(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()

	id := &fftypes.Identity{OnChain: "signer001"}
	err := e.VerifyIdentitySyntax(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "signer001", id.OnChain)

}

func TestHandleMessageBatchPinOK(t *testing.T) {
	data := []byte(`
[
  {
    "chaincodeId": "firefly",
    "blockNumber": 91,
    "transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
    "eventName": "BatchPin",
    "payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMxNjY3LCJuYW5vcyI6NzkxNDk5MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIjB4MTliODIwOTNkZTVjZTkyYTAxZTMzMzA0OGU4NzdlMjM3NDM1NGJmODQ2ZGQwMzQ4NjRlZjZmZmJkNjQzODc3MSJdfQ==",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  },
  {
    "chaincodeId": "firefly",
    "blockNumber": 77,
    "transactionId": "a488800a70c8f765871611168d422fb29cc37da2d0a196a3200c8068ba1706fd",
    "eventName": "BatchPin",
    "payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMxNjY3LCJuYW5vcyI6NzkxNDk5MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIjB4MTliODIwOTNkZTVjZTkyYTAxZTMzMzA0OGU4NzdlMjM3NDM1NGJmODQ2ZGQwMzQ4NjRlZjZmZmJkNjQzODc3MSJdfQ==",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }	
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: em,
	}

	em.On("BatchPinComplete", mock.Anything, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", mock.Anything, mock.Anything).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	b := em.Calls[0].Arguments[0].(*blockchain.BatchPin)
	assert.Equal(t, "ns1", b.Namespace)
	assert.Equal(t, "e19af8b3-9060-4051-812d-7597d19adfb9", b.TransactionID.String())
	assert.Equal(t, "847d3bfd-0742-49ef-b65d-3fed15f5b0a6", b.BatchID.String())
	assert.Equal(t, "d71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be", b.BatchHash.String())
	assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", b.BatchPaylodRef)
	assert.Equal(t, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", em.Calls[0].Arguments[1])
	assert.Equal(t, "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2", em.Calls[0].Arguments[2])
	assert.Len(t, b.Contexts, 2)
	assert.Equal(t, "68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a", b.Contexts[0].String())
	assert.Equal(t, "19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771", b.Contexts[1].String())

	em.AssertExpectations(t)

}

func TestHandleMessageEmptyPayloadRef(t *testing.T) {
	data := []byte(`
[
  {
    "chaincodeId": "firefly",
    "blockNumber": 91,
    "transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
    "eventName": "BatchPin",
    "payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMyMDQwLCJuYW5vcyI6MjI5MjM1MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiIiwiY29udGV4dHMiOlsiMHg2OGU0ZGE3OWY4MDViY2E1YjkxMmJjZGE5YzYzZDAzZTZlODY3MTA4ZGFiYjliOTQ0MTA5YWVhNTQxZWY1MjJhIiwiMHgxOWI4MjA5M2RlNWNlOTJhMDFlMzMzMDQ4ZTg3N2UyMzc0MzU0YmY4NDZkZDAzNDg2NGVmNmZmYmQ2NDM4NzcxIl19",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: em,
	}

	em.On("BatchPinComplete", mock.Anything, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", mock.Anything, mock.Anything).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	b := em.Calls[0].Arguments[0].(*blockchain.BatchPin)
	assert.Equal(t, "ns1", b.Namespace)
	assert.Equal(t, "e19af8b3-9060-4051-812d-7597d19adfb9", b.TransactionID.String())
	assert.Equal(t, "847d3bfd-0742-49ef-b65d-3fed15f5b0a6", b.BatchID.String())
	assert.Equal(t, "d71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be", b.BatchHash.String())
	assert.Empty(t, b.BatchPaylodRef)
	assert.Equal(t, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", em.Calls[0].Arguments[1])
	assert.Equal(t, "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2", em.Calls[0].Arguments[2])
	assert.Len(t, b.Contexts, 2)
	assert.Equal(t, "68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a", b.Contexts[0].String())
	assert.Equal(t, "19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771", b.Contexts[1].String())

	em.AssertExpectations(t)

}

func TestHandleMessageBatchPinExit(t *testing.T) {
	data := []byte(`
[
  {
    "chaincodeId": "firefly",
    "blockNumber": 91,
    "transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
    "eventName": "BatchPin",
    "payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMyMDQwLCJuYW5vcyI6MjI5MjM1MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiIiwiY29udGV4dHMiOlsiMHg2OGU0ZGE3OWY4MDViY2E1YjkxMmJjZGE5YzYzZDAzZTZlODY3MTA4ZGFiYjliOTQ0MTA5YWVhNTQxZWY1MjJhIiwiMHgxOWI4MjA5M2RlNWNlOTJhMDFlMzMzMDQ4ZTg3N2UyMzc0MzU0YmY4NDZkZDAzNDg2NGVmNmZmYmQ2NDM4NzcxIl19",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: em,
	}

	em.On("BatchPinComplete", mock.Anything, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

}

func TestHandleMessageBatchPinEmpty(t *testing.T) {
	data := []byte(`
[
  {
    "chaincodeId": "firefly",
    "blockNumber": 91,
    "transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
    "eventName": "BatchPin",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadBatchHash(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	data := []byte(`[{
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"eventName": "BatchPin",
		"payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMzMjMzLCJuYW5vcyI6NTAwMDc3MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIhZ29vZCIsInBheWxvYWRSZWYiOiJRbWY0MTJqUVppdVZVdGRnbkIzNkZYRlg3eGc1VjZLRWJTSjRkcFF1aGtMeWZEIiwiY29udGV4dHMiOltdfQ==",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
	}]`)
	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadPin(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	data := []byte(`[{
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"eventName": "BatchPin",
		"payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMzMzQ0LCJuYW5vcyI6OTY1NjE4MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIiFnb29kIl19",
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
	e := &Fabric{callbacks: em}
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
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopSendClosed(t *testing.T) {
	e, cancel := newTestFabric()
	cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	close(r)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Send", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
}

func TestHandleReceiptTXSuccess(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: em,
		wsconn:    wsm,
	}

	var reply fftypes.JSONObject
	data := []byte(`{
		"_id": "748e7587-9e72-4244-7351-808f69b88291",
    "headers": {
        "id": "0ef91fb6-09c5-4ca2-721c-74b4869097c2",
        "requestId": "748e7587-9e72-4244-7351-808f69b88291",
        "requestOffset": "",
        "timeElapsed": 0.475721,
        "timeReceived": "2021-08-27T03:04:34.199742Z",
        "type": "TransactionSuccess"
    },
    "receivedAt": 1630033474675
  }`)

	em.On("BlockchainTxUpdate",
		"748e7587-9e72-4244-7351-808f69b88291",
		fftypes.OpStatusSucceeded,
		"",
		mock.Anything).Return(nil)

	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	err = e.handleReceipt(context.Background(), reply)
	assert.NoError(t, err)

}

func TestHandleReceiptNoRequestID(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: em,
		wsconn:    wsm,
	}

	var reply fftypes.JSONObject
	data := []byte(`{}`)
	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	err = e.handleReceipt(context.Background(), reply)
	assert.NoError(t, err)
}

func TestFormatNil(t *testing.T) {
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", hexFormatB32(nil))
}
