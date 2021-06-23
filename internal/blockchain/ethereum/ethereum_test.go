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

package ethereum

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
var utEthconnectConf = utConfPrefix.SubPrefix(EthconnectConfigKey)

func resetConf() {
	config.Reset()
	e := &Ethereum{}
	e.InitPrefix(utConfPrefix)
}

func newTestEthereum() (*Ethereum, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Ethereum{
		ctx:          ctx,
		client:       resty.New().SetHostURL("http://localhost:12345"),
		instancePath: "/instances/0x12345",
		topic:        "topic1",
		prefixShort:  defaultPrefixShort,
		prefixLong:   defaultPrefixLong,
		callbacks:    em,
		wsconn:       wsm,
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
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf()
	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10138.*url", err)
}

func TestInitMissingInstance(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf()
	utEthconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10138.*instance", err)
}

func TestInitMissingTopic(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	resetConf()
	utEthconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10138.*topic", err)
}

func TestInitAllNewStreamsAndWSEvent(t *testing.T) {

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
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/subscriptions", httpURL),
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/instances/0x12345/BatchPin", httpURL),
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "es12345", body["streamID"])
			return httpmock.NewJsonResponderOrPanic(200, subscription{ID: "sub12345"})(req)
		})

	resetConf()
	utEthconnectConf.Set(restclient.HTTPConfigURL, httpURL)
	utEthconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.NoError(t, err)

	assert.Equal(t, "ethereum", e.Name())
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

	e, cancel := newTestEthereum()
	defer cancel()

	resetConf()
	utEthconnectConf.Set(restclient.HTTPConfigURL, "!!!://")
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")
	utEthconnectConf.Set(EthconnectConfigSkipEventstreamInit, true)

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10162", err)

}

func TestWSConnectFail(t *testing.T) {

	wsm := &wsmocks.WSClient{}
	e := &Ethereum{
		ctx:    context.Background(),
		wsconn: wsm,
	}
	wsm.On("Connect").Return(fmt.Errorf("pop"))

	err := e.Start()
	assert.EqualError(t, err, "pop")
}

func TestInitAllExistingStreams(t *testing.T) {

	e, cancel := newTestEthereum()
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
	utEthconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Equal(t, 2, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.initInfo.stream.ID)
	assert.Equal(t, "sub12345", e.initInfo.subs[0].ID)

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

	resetConf()
	utEthconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10111", err)
	assert.Regexp(t, "pop", err)

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

	resetConf()
	utEthconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10111", err)
	assert.Regexp(t, "pop", err)

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

	resetConf()
	utEthconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10111", err)
	assert.Regexp(t, "pop", err)

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
	httpmock.RegisterResponder("POST", "http://localhost:12345/instances/0x12345/BatchPin",
		httpmock.NewStringResponder(500, `pop`))

	resetConf()
	utEthconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10111", err)
	assert.Regexp(t, "pop", err)

}

func TestSubmitBatchPinOK(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
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

	httpmock.RegisterResponder("POST", `http://localhost:12345/instances/0x12345/pinBatch`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, addr, req.FormValue(defaultPrefixShort+"-from"))
			assert.Equal(t, "false", req.FormValue(defaultPrefixShort+"-sync"))
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", body["uuids"])
			assert.Equal(t, ethHexFormatB32(batch.BatchHash), body["batchHash"])
			assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", body["payloadRef"])
			return httpmock.NewJsonResponderOrPanic(200, asyncTXSubmission{ID: "abcd1234"})(req)
		})

	txid, err := e.SubmitBatchPin(context.Background(), nil, &fftypes.Identity{OnChain: addr}, batch)

	assert.NoError(t, err)
	assert.Equal(t, "abcd1234", txid)

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

	httpmock.RegisterResponder("POST", `http://localhost:12345/instances/0x12345/pinBatch`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, addr, req.FormValue("fly-from"))
			assert.Equal(t, "false", req.FormValue("fly-sync"))
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", body["uuids"])
			assert.Equal(t, ethHexFormatB32(batch.BatchHash), body["batchHash"])
			assert.Equal(t, "", body["payloadRef"])
			return httpmock.NewJsonResponderOrPanic(200, asyncTXSubmission{ID: "abcd1234"})(req)
		})

	txid, err := e.SubmitBatchPin(context.Background(), nil, &fftypes.Identity{OnChain: addr}, batch)

	assert.NoError(t, err)
	assert.Equal(t, "abcd1234", txid)

}

func TestSubmitBatchPinFail(t *testing.T) {

	e, cancel := newTestEthereum()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
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

	httpmock.RegisterResponder("POST", `http://localhost:12345/instances/0x12345/pinBatch`,
		httpmock.NewStringResponder(500, "pop"))

	_, err := e.SubmitBatchPin(context.Background(), nil, &fftypes.Identity{OnChain: addr}, batch)

	assert.Regexp(t, "FF10111", err)
	assert.Regexp(t, "pop", err)

}

func TestVerifyEthAddress(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	id := &fftypes.Identity{OnChain: "0x12345"}
	err := e.VerifyIdentitySyntax(context.Background(), id)
	assert.Regexp(t, "FF10141", err)

	id = &fftypes.Identity{OnChain: "0x2a7c9D5248681CE6c393117E641aD037F5C079F6"}
	err = e.VerifyIdentitySyntax(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "0x2a7c9d5248681ce6c393117e641ad037f5c079f6", id.OnChain)

}

func TestHandleMessageBatchPinOK(t *testing.T) {
	data := []byte(`
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
			],
      "timestamp": "1620576488"
    },
    "subID": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
    "signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
    "logIndex": "50"
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
      "timestamp": "1620576488",
			"contexts": [
				"0x8a63eb509713b0cf9250a8eee24ee2dfc4b37225e3ad5c29c95127699d382f85"
			]
    },
    "subID": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
    "signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
    "logIndex": "51"
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
      "timestamp": "1620576488",
			"contexts": [
				"0xdab67320f1a0d0f1da572975e3a9ab6ef0fed315771c99fea0bfb54886c1aa94"
			]
    },
    "subID": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
    "signature": "Random(address,uint256,bytes32,bytes32,bytes32)",
    "logIndex": "51"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{
		callbacks: em,
	}

	em.On("BatchPinComplete", mock.Anything, "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635", mock.Anything, mock.Anything).Return(nil)

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
	assert.Equal(t, "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635", em.Calls[0].Arguments[1])
	assert.Equal(t, "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628", em.Calls[0].Arguments[2])
	assert.Len(t, b.Contexts, 2)
	assert.Equal(t, "68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a", b.Contexts[0].String())
	assert.Equal(t, "19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771", b.Contexts[1].String())

	em.AssertExpectations(t)

}

func TestHandleMessageEmptyPayloadRef(t *testing.T) {
	data := []byte(`
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
      "payloadRef": "",
			"contexts": [
				"0x68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a",
				"0x19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771"
			],
      "timestamp": "1620576488"
    },
    "subID": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
    "signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
    "logIndex": "50"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{
		callbacks: em,
	}

	em.On("BatchPinComplete", mock.Anything, "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635", mock.Anything, mock.Anything).Return(nil)

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
	assert.Equal(t, "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635", em.Calls[0].Arguments[1])
	assert.Equal(t, "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628", em.Calls[0].Arguments[2])
	assert.Len(t, b.Contexts, 2)
	assert.Equal(t, "68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a", b.Contexts[0].String())
	assert.Equal(t, "19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771", b.Contexts[1].String())

	em.AssertExpectations(t)

}

func TestHandleMessageBatchPinExit(t *testing.T) {
	data := []byte(`
[
  {
    "address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
    "blockNumber": "38011",
    "transactionIndex": "0x1",
    "transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
    "data": {
      "author": "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
			"namespace": "ns1",
			"uuids": "0xe19af8b390604051812d7597d19adfb9a04c7cc37d444c2ba3b054e21326697e",
			"batchHash": "0x9c19a93b6e85fee041f60f097121829e54cd4aa97ed070d1bc76147caf911fed",
      "payloadRef": "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
      "timestamp": "1620576488"
    },
    "subID": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
    "signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
    "logIndex": "51"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{
		callbacks: em,
	}

	em.On("BatchPinComplete", mock.Anything, "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

}

func TestHandleMessageBatchPinEmpty(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{callbacks: em}
	var events []interface{}
	err := json.Unmarshal([]byte(`[{"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])"}]`), &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadTransactionID(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{callbacks: em}
	data := []byte(`[{
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
			],
			"timestamp": "!1620576488"
		}
	}]`)
	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadIDentity(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{callbacks: em}
	data := []byte(`[{
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
			],
			"timestamp": "1620576488"
		}
	}]`)
	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadBatchHash(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Ethereum{callbacks: em}
	data := []byte(`[{
		"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
    "blockNumber": "38011",
    "transactionIndex": "0x1",
    "transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
      "author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"namespace": "ns1",
			"uuids": "0xe19af8b390604051812d7597d19adfb9847d3bfd074249efb65d3fed15f5b0a6",
			"batchHash": "!good",
      "payloadRef": "0xeda586bd8f3c4bc1db5c4b5755113b9a9b4174abe28679fdbc219129400dd7ae",
			"contexts": [
				"0xb41753f11522d4ef5c4a467972cf54744c04628ff84a1c994f1b288b2f6ec836",
				"0xc6c683a0fbe15e452e1ecc3751657446e2f645a8231e3ef9f3b4a8eae03c4136"
			],
			"timestamp": "1620576488"
		}
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
	e := &Ethereum{callbacks: em}
	data := []byte(`[{
		"signature": "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])",
    "blockNumber": "38011",
    "transactionIndex": "0x1",
    "transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
      "author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"namespace": "ns1",
			"uuids": "0xe19af8b390604051812d7597d19adfb9847d3bfd074249efb65d3fed15f5b0a6",
			"batchHash": "0xd71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be",
      "payloadRef": "0xeda586bd8f3c4bc1db5c4b5755113b9a9b4174abe28679fdbc219129400dd7ae",
			"contexts": [
				"0xb41753f11522d4ef5c4a467972cf54744c04628ff84a1c994f1b288b2f6ec836",
				"!good"
			],
			"timestamp": "1620576488"
		}
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
	e := &Ethereum{callbacks: em}
	err := e.handleMessageBatch(context.Background(), []interface{}{10, 20})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestEventLoopContextCancelled(t *testing.T) {
	e, cancel := newTestEthereum()
	cancel()
	r := make(<-chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	wsm.On("Receive").Return(r)
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopReceiveClosed(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	close(r)
	wsm.On("Receive").Return((<-chan []byte)(r))
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopSendClosed(t *testing.T) {
	e, cancel := newTestEthereum()
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
	e := &Ethereum{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: em,
		wsconn:    wsm,
	}

	var reply fftypes.JSONObject
	data := []byte(`{
    "_id": "4373614c-e0f7-47b0-640e-7eacec417a9e",
    "blockHash": "0xad269b2b43481e44500f583108e8d24bd841fb767c7f526772959d195b9c72d5",
    "blockNumber": "209696",
    "cumulativeGasUsed": "24655",
    "from": "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
    "gasUsed": "24655",
    "headers": {
      "id": "4603a151-f212-446e-5c15-0f36b57cecc7",
      "requestId": "4373614c-e0f7-47b0-640e-7eacec417a9e",
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

	em.On("TxSubmissionUpdate",
		"4373614c-e0f7-47b0-640e-7eacec417a9e",
		fftypes.OpStatusSucceeded,
		"0x71a38acb7a5d4a970854f6d638ceb1fa10a4b59cbf4ed7674273a1a8dc8b36b8",
		"",
		mock.Anything).Return(nil)

	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	err = e.handleReceipt(context.Background(), reply)
	assert.NoError(t, err)

}

func TestHandleBadPayloadsAndThenReceiptFailure(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	e.closed = make(chan struct{})

	wsm.On("Receive").Return((<-chan []byte)(r))
	data := []byte(`{
		"_id": "6fb94fff-81d3-4094-567d-e031b1871694",
		"errorMessage": "Packing arguments for method 'broadcastBatch': abi: cannot use [3]uint8 as type [32]uint8 as argument",
		"headers": {
			"id": "3a37b17b-13b6-4dc5-647a-07c11eae0be3",
			"requestId": "6fb94fff-81d3-4094-567d-e031b1871694",
			"requestOffset": "zzn4y4v4si-zzjjepe9x4-requests:0:0",
			"timeElapsed": 0.020969053,
			"timeReceived": "2021-05-31T02:35:11.458880504Z",
			"type": "Error"
		},
		"receivedAt": 1622428511616,
		"requestPayload": "{\"from\":\"0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635\",\"gas\":0,\"gasPrice\":0,\"headers\":{\"id\":\"6fb94fff-81d3-4094-567d-e031b1871694\",\"type\":\"SendTransaction\"},\"method\":{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"txnId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"batchId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"payloadRef\",\"type\":\"bytes32\"}],\"name\":\"broadcastBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},\"params\":[\"12345\",\"!\",\"!\"],\"to\":\"0xd3266a857285fb75eb7df37353b4a15c8bb828f5\",\"value\":0}"
	}`)

	em := e.callbacks.(*blockchainmocks.Callbacks)
	txsu := em.On("TxSubmissionUpdate",
		"6fb94fff-81d3-4094-567d-e031b1871694",
		fftypes.OpStatusFailed,
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
	<-done
}

func TestHandleReceiptNoRequestID(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Ethereum{
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
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", ethHexFormatB32(nil))
}
