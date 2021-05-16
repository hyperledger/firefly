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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/ffresty"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/wsserver"
	"github.com/kaleido-io/firefly/mocks/blockchainmocks"
	"github.com/kaleido-io/firefly/mocks/wsmocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("eth_unit_tests")
var utEthconnectConf = utConfPrefix.SubPrefix(EthconnectConfigKey)

func resetConf() {
	config.Reset()
	e := &Ethereum{}
	e.InitConfigPrefix(utConfPrefix)
}

func TestInitMissingURL(t *testing.T) {
	e := &Ethereum{}
	resetConf()
	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})
	assert.Regexp(t, "FF10138.*url", err.Error())
}

func TestInitMissingInstance(t *testing.T) {
	e := &Ethereum{}
	resetConf()
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})
	assert.Regexp(t, "FF10138.*instance", err.Error())
}

func TestInitMissingTopic(t *testing.T) {
	e := &Ethereum{}
	resetConf()
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")

	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})
	assert.Regexp(t, "FF10138.*topic", err.Error())
}

func TestInitAllNewStreamsAndWSEvent(t *testing.T) {

	log.SetLevel("debug")
	e := &Ethereum{}

	wsServer := wsserver.NewWebSocketServer(context.Background())
	svr := httptest.NewServer(wsServer.Handler())
	defer svr.Close()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", fmt.Sprintf("http://%s/eventstreams", svr.Listener.Addr()),
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://%s/eventstreams", svr.Listener.Addr()),
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", fmt.Sprintf("http://%s/subscriptions", svr.Listener.Addr()),
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("http://%s/instances/0x12345/BroadcastBatch", svr.Listener.Addr()),
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "es12345", body["streamId"])
			return httpmock.NewJsonResponderOrPanic(200, subscription{ID: "sub12345"})(req)
		})

	resetConf()
	utEthconnectConf.Set(ffresty.HTTPConfigURL, fmt.Sprintf("http://%s", svr.Listener.Addr()))
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})
	assert.NoError(t, err)

	assert.Equal(t, "ethereum", e.Name())
	assert.Equal(t, 4, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.initInfo.stream.ID)
	assert.Equal(t, "sub12345", e.initInfo.subs[0].ID)
	assert.True(t, e.Capabilities().GlobalSequencer)

	err = e.Start()
	assert.NoError(t, err)

	sender, receiver, _ := wsServer.GetChannels("topic1")
	sender <- []fftypes.JSONObject{} // empty batch, will be ignored, but acked
	err = <-receiver
	assert.NoError(t, err) // should be ack, not error

}

func TestWSInitFail(t *testing.T) {

	e := &Ethereum{}

	resetConf()
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "!!!://")
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")
	utEthconnectConf.Set(EthconnectConfigSkipEventstreamInit, true)

	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})
	assert.Regexp(t, "FF10162", err.Error())

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

	e := &Ethereum{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", WebSocket: eventStreamWebsocket{Topic: "topic1"}}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Name: "BroadcastBatch"},
		}))

	resetConf()
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})

	assert.Equal(t, 2, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.initInfo.stream.ID)
	assert.Equal(t, "sub12345", e.initInfo.subs[0].ID)

	assert.NoError(t, err)

}

func TestStreamQueryError(t *testing.T) {

	e := &Ethereum{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewStringResponder(500, `pop`))

	resetConf()
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})

	assert.Regexp(t, "FF10111", err.Error())
	assert.Regexp(t, "pop", err.Error())

}

func TestStreamCreateError(t *testing.T) {

	e := &Ethereum{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewStringResponder(500, `pop`))

	resetConf()
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})

	assert.Regexp(t, "FF10111", err.Error())
	assert.Regexp(t, "pop", err.Error())

}

func TestSubQueryError(t *testing.T) {

	e := &Ethereum{}

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
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})

	assert.Regexp(t, "FF10111", err.Error())
	assert.Regexp(t, "pop", err.Error())

}

func TestSubQueryCreateError(t *testing.T) {

	e := &Ethereum{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/instances/0x12345/BroadcastBatch",
		httpmock.NewStringResponder(500, `pop`))

	resetConf()
	utEthconnectConf.Set(ffresty.HTTPConfigURL, "http://localhost:12345")
	utEthconnectConf.Set(ffresty.HTTPConfigRetryEnabled, false)
	utEthconnectConf.Set(ffresty.HTTPCustomClient, mockedClient)
	utEthconnectConf.Set(EthconnectConfigInstancePath, "/instances/0x12345")
	utEthconnectConf.Set(EthconnectConfigTopic, "topic1")

	err := e.Init(context.Background(), utConfPrefix, &blockchainmocks.Events{})

	assert.Regexp(t, "FF10111", err.Error())
	assert.Regexp(t, "pop", err.Error())

}

func newTestEthereum() *Ethereum {
	return &Ethereum{
		ctx:          context.Background(),
		client:       resty.New().SetHostURL("http://localhost:12345"),
		instancePath: "/instances/0x12345",
		topic:        "topic1",
	}
}

func TestSubmitBroadcastBatchOK(t *testing.T) {

	e := newTestEthereum()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BroadcastBatch{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchPaylodRef: fftypes.NewRandB32(),
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/instances/0x12345/broadcastBatch`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, addr, req.FormValue("kld-from"))
			assert.Equal(t, "false", req.FormValue("kld-sync"))
			assert.Equal(t, ethHexFormatB32(fftypes.UUIDBytes(batch.TransactionID)), body["txnId"])
			assert.Equal(t, ethHexFormatB32(fftypes.UUIDBytes(batch.BatchID)), body["batchId"])
			assert.Equal(t, ethHexFormatB32(batch.BatchPaylodRef), body["payloadRef"])
			return httpmock.NewJsonResponderOrPanic(200, asyncTXSubmission{ID: "abcd1234"})(req)
		})

	txid, err := e.SubmitBroadcastBatch(context.Background(), addr, batch)

	assert.NoError(t, err)
	assert.Equal(t, "abcd1234", txid)

}

func TestSubmitBroadcastBatchFail(t *testing.T) {

	e := newTestEthereum()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BroadcastBatch{
		TransactionID:  fftypes.NewUUID(),
		BatchID:        fftypes.NewUUID(),
		BatchPaylodRef: fftypes.NewRandB32(),
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/instances/0x12345/broadcastBatch`,
		httpmock.NewStringResponder(500, "pop"))

	_, err := e.SubmitBroadcastBatch(context.Background(), addr, batch)

	assert.Regexp(t, "FF10111", err.Error())
	assert.Regexp(t, "pop", err.Error())

}

func TestVerifyEthAddress(t *testing.T) {
	e := &Ethereum{}

	_, err := e.VerifyIdentitySyntax(context.Background(), "0x12345")
	assert.Regexp(t, "FF10141", err.Error())

	addr, err := e.VerifyIdentitySyntax(context.Background(), "0x2a7c9D5248681CE6c393117E641aD037F5C079F6")
	assert.NoError(t, err)
	assert.Equal(t, "0x2a7c9d5248681ce6c393117e641ad037f5c079f6", addr)

}

func TestHandleMessageBatchBroadcastOK(t *testing.T) {
	data := []byte(`
[
  {
    "address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
    "blockNumber": "38011",
    "transactionIndex": "0x0",
    "transactionHash": "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628",
    "data": {
      "author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"txnId": "0xe19af8b390604051812d7597d19adfb900000000000000000000000000000000",
      "batchId": "0x847d3bfd074249efb65d3fed15f5b0a600000000000000000000000000000000",
      "payloadRef": "0xeda586bd8f3c4bc1db5c4b5755113b9a9b4174abe28679fdbc219129400dd7ae",
      "timestamp": "1620576488"
    },
    "subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
    "signature": "BroadcastBatch(address,uint256,bytes32,bytes32,bytes32)",
    "logIndex": "50"
  },
  {
    "address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
    "blockNumber": "38011",
    "transactionIndex": "0x1",
    "transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
    "data": {
      "author": "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
			"txnId": "0x8a578549e56b49f9bd78d731f22b08d700000000000000000000000000000000",
      "batchId": "0xa04c7cc37d444c2ba3b054e21326697e00000000000000000000000000000000",
      "payloadRef": "0x23ad1bc340ac7516f0cbf1be677122303ffce81f32400c440295c44d7963d185",
      "timestamp": "1620576488"
    },
    "subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
    "signature": "BroadcastBatch(address,uint256,bytes32,bytes32,bytes32)",
    "logIndex": "51"
  }
]`)

	em := &blockchainmocks.Events{}
	e := &Ethereum{
		events: em,
	}

	em.On("SequencedBroadcastBatch", mock.Anything, "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635", mock.Anything, mock.Anything).Return(nil)

	e.handleMessageBatch(context.Background(), data)

	b := em.Calls[0].Arguments[0].(*blockchain.BroadcastBatch)
	assert.Equal(t, "e19af8b3-9060-4051-812d-7597d19adfb9", b.TransactionID.String())
	assert.Equal(t, "847d3bfd-0742-49ef-b65d-3fed15f5b0a6", b.BatchID.String())
	assert.Equal(t, "eda586bd8f3c4bc1db5c4b5755113b9a9b4174abe28679fdbc219129400dd7ae", b.BatchPaylodRef.String())
	assert.Equal(t, "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635", em.Calls[0].Arguments[1])
	assert.Equal(t, "0xc26df2bf1a733e9249372d61eb11bd8662d26c8129df76890b1beb2f6fa72628", em.Calls[0].Arguments[2])

}

func TestHandleMessageBatchBroadcastExit(t *testing.T) {
	data := []byte(`
[
  {
    "address": "0x1C197604587F046FD40684A8f21f4609FB811A7b",
    "blockNumber": "38011",
    "transactionIndex": "0x1",
    "transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
    "data": {
      "author": "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635",
			"txnId": "0xe19af8b390604051812d7597d19adfb900000000000000000000000000000000",
      "batchId": "0xa04c7cc37d444c2ba3b054e21326697e00000000000000000000000000000000",
      "payloadRef": "0x23ad1bc340ac7516f0cbf1be677122303ffce81f32400c440295c44d7963d185",
      "timestamp": "1620576488"
    },
    "subId": "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
    "signature": "BroadcastBatch(address,uint256,bytes32,bytes32,bytes32)",
    "logIndex": "51"
  }
]`)

	em := &blockchainmocks.Events{}
	e := &Ethereum{
		events: em,
	}

	em.On("SequencedBroadcastBatch", mock.Anything, "0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	err := e.handleMessageBatch(context.Background(), data)
	assert.EqualError(t, err, "pop")

}

func TestHandleMessageBatchBroadcastEmpty(t *testing.T) {
	em := &blockchainmocks.Events{}
	e := &Ethereum{events: em}
	e.handleMessageBatch(context.Background(), []byte(`[{"signature": "BroadcastBatch(address,uint256,bytes32,bytes32,bytes32)"}]`))
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchBroadcastBadTransactionID(t *testing.T) {
	em := &blockchainmocks.Events{}
	e := &Ethereum{events: em}
	e.handleMessageBatch(context.Background(), []byte(`[{
		"signature": "BroadcastBatch(address,uint256,bytes32,bytes32,bytes32)",
    "blockNumber": "38011",
    "transactionIndex": "0x1",
    "transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
      "author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"txnId": "!good",
      "batchId": "0x847d3bfd074249efb65d3fed15f5b0a600000000000000000000000000000000",
      "payloadRef": "0xeda586bd8f3c4bc1db5c4b5755113b9a9b4174abe28679fdbc219129400dd7ae",
			"timestamp": "!1620576488"
		}
	}]`))
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchBroadcastBadIdentity(t *testing.T) {
	em := &blockchainmocks.Events{}
	e := &Ethereum{events: em}
	e.handleMessageBatch(context.Background(), []byte(`[{
		"signature": "BroadcastBatch(address,uint256,bytes32,bytes32,bytes32)",
    "blockNumber": "38011",
    "transactionIndex": "0x1",
    "transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
      "author": "!good",
			"txnId": "0xe19af8b390604051812d7597d19adfb900000000000000000000000000000000",
      "batchId": "0x847d3bfd074249efb65d3fed15f5b0a600000000000000000000000000000000",
      "payloadRef": "0xeda586bd8f3c4bc1db5c4b5755113b9a9b4174abe28679fdbc219129400dd7ae",
			"timestamp": "1620576488"
		}
	}]`))
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchBroadcastBadBatchID(t *testing.T) {
	em := &blockchainmocks.Events{}
	e := &Ethereum{events: em}
	e.handleMessageBatch(context.Background(), []byte(`[{
		"signature": "BroadcastBatch(address,uint256,bytes32,bytes32,bytes32)",
    "blockNumber": "38011",
    "transactionIndex": "0x1",
    "transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
      "author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
			"txnId": "0xe19af8b390604051812d7597d19adfb900000000000000000000000000000000",
      "batchId": "!good",
      "payloadRef": "0xeda586bd8f3c4bc1db5c4b5755113b9a9b4174abe28679fdbc219129400dd7ae",
			"timestamp": "1620576488"
		}
	}]`))
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchBroadcastBadPayloadRef(t *testing.T) {
	em := &blockchainmocks.Events{}
	e := &Ethereum{events: em}
	e.handleMessageBatch(context.Background(), []byte(`[{
		"signature": "BroadcastBatch(address,uint256,bytes32,bytes32,bytes32)",
    "blockNumber": "38011",
    "transactionIndex": "0x1",
    "transactionHash": "0x0c50dff0893e795293189d9cc5ba0d63c4020d8758ace4a69d02c9d6d43cb695",
		"data": {
      "author": "0X91D2B4381A4CD5C7C0F27565A7D4B829844C8635",
      "batchId": "0x847d3bfd074249efb65d3fed15f5b0a600000000000000000000000000000000",
			"txnId": "0xe19af8b390604051812d7597d19adfb900000000000000000000000000000000",
      "payloadRef": "!good",
			"timestamp": "1620576488"
		}
	}]`))
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchBadJSON(t *testing.T) {
	em := &blockchainmocks.Events{}
	e := &Ethereum{events: em}
	e.handleMessageBatch(context.Background(), []byte(`!good`))
	assert.Equal(t, 0, len(em.Calls))
}

func TestEventLoopContextCancelled(t *testing.T) {
	em := &blockchainmocks.Events{}
	wsm := &wsmocks.WSClient{}
	ctxCancelled, cancel := context.WithCancel(context.Background())
	cancel()
	e := &Ethereum{
		ctx:    ctxCancelled,
		topic:  "topic1",
		events: em,
		wsconn: wsm,
	}
	r := make(<-chan []byte)
	wsm.On("Receive").Return(r)
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopReceiveClosed(t *testing.T) {
	em := &blockchainmocks.Events{}
	wsm := &wsmocks.WSClient{}
	e := &Ethereum{
		ctx:    context.Background(),
		topic:  "topic1",
		events: em,
		wsconn: wsm,
	}
	r := make(chan []byte)
	close(r)
	wsm.On("Receive").Return((<-chan []byte)(r))
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopSendClosed(t *testing.T) {
	em := &blockchainmocks.Events{}
	wsm := &wsmocks.WSClient{}
	e := &Ethereum{
		ctx:    context.Background(),
		topic:  "topic1",
		events: em,
		wsconn: wsm,
	}
	r := make(chan []byte, 1)
	r <- []byte(`[]`)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Send", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	e.eventLoop() // we're simply looking for it exiting
}
