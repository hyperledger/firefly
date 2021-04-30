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
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/ffresty"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestConfigInterfaceCorrect(t *testing.T) {
	e := &Ethereum{}
	_, ok := e.ConfigInterface().(*Config)
	assert.True(t, ok)
}

func TestInitAllNewStreams(t *testing.T) {

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
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "es12345", body["streamId"])
			return httpmock.NewJsonResponderOrPanic(200, subscription{ID: "sub12345"})(req)
		})

	_, err := e.Init(context.Background(), &Config{
		Ethconnect: EthconnectConfig{
			HTTPConfig: ffresty.HTTPConfig{
				URL:        "http://localhost:12345",
				HttpClient: mockedClient,
			},
		},
	}, &blockchain.MockEvents{})

	assert.Equal(t, 4, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.initInfo.stream.ID)
	assert.Equal(t, "sub12345", e.initInfo.subs[0].ID)

	assert.NoError(t, err)

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
			{ID: "sub12345", Name: "AssetInstanceBatchCreated"},
		},
		))

	_, err := e.Init(context.Background(), &Config{
		Ethconnect: EthconnectConfig{
			HTTPConfig: ffresty.HTTPConfig{
				URL:        "http://localhost:12345",
				HttpClient: mockedClient,
			},
			Topic: "topic1",
		},
	}, &blockchain.MockEvents{})

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

	var no bool = false
	_, err := e.Init(context.Background(), &Config{
		Ethconnect: EthconnectConfig{
			HTTPConfig: ffresty.HTTPConfig{
				URL:        "http://localhost:12345",
				HttpClient: mockedClient,
				Retry: &ffresty.HTTPRetryConfig{
					Enabled: &no,
				},
			},
			Topic: "topic1",
		},
	}, &blockchain.MockEvents{})

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

	var no bool = false
	_, err := e.Init(context.Background(), &Config{
		Ethconnect: EthconnectConfig{
			HTTPConfig: ffresty.HTTPConfig{
				URL:        "http://localhost:12345",
				HttpClient: mockedClient,
				Retry: &ffresty.HTTPRetryConfig{
					Enabled: &no,
				},
			},
			Topic: "topic1",
		},
	}, &blockchain.MockEvents{})

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

	var no bool = false
	_, err := e.Init(context.Background(), &Config{
		Ethconnect: EthconnectConfig{
			HTTPConfig: ffresty.HTTPConfig{
				URL:        "http://localhost:12345",
				HttpClient: mockedClient,
				Retry: &ffresty.HTTPRetryConfig{
					Enabled: &no,
				},
			},
			Topic: "topic1",
		},
	}, &blockchain.MockEvents{})

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
	httpmock.RegisterResponder("POST", "http://localhost:12345/subscriptions",
		httpmock.NewStringResponder(500, `pop`))

	var no bool = false
	_, err := e.Init(context.Background(), &Config{
		Ethconnect: EthconnectConfig{
			HTTPConfig: ffresty.HTTPConfig{
				URL:        "http://localhost:12345",
				HttpClient: mockedClient,
				Retry: &ffresty.HTTPRetryConfig{
					Enabled: &no,
				},
			},
			Topic: "topic1",
		},
	}, &blockchain.MockEvents{})

	assert.Regexp(t, "FF10111", err.Error())
	assert.Regexp(t, "pop", err.Error())

}

func newTestEthereum() *Ethereum {
	return &Ethereum{
		ctx:    context.Background(),
		client: resty.New().SetHostURL("http://localhost:12345"),
		conf: &Config{
			Ethconnect: EthconnectConfig{
				InstancePath: "instances/0x12345",
			},
		},
	}
}

func TestSubmitBroadcastBatchOK(t *testing.T) {

	e := newTestEthereum()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	addr := ethHexFormatB32(fftypes.NewRandB32())
	batch := &blockchain.BroadcastBatch{
		Timestamp:      time.Now().UnixNano(),
		BatchID:        fftypes.NewRandB32(),
		BatchPaylodRef: fftypes.NewRandB32(),
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/instances/0x12345/broadcastBatch`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, addr, req.FormValue("kld-from"))
			assert.Equal(t, "false", req.FormValue("kld-sync"))
			assert.Equal(t, ethHexFormatB32(batch.BatchID), body["batchId"])
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
		Timestamp:      time.Now().UnixNano(),
		BatchID:        fftypes.NewRandB32(),
		BatchPaylodRef: fftypes.NewRandB32(),
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/instances/0x12345/broadcastBatch`,
		httpmock.NewStringResponder(500, "pop"))

	_, err := e.SubmitBroadcastBatch(context.Background(), addr, batch)

	assert.Regexp(t, "FF10111", err.Error())
	assert.Regexp(t, "pop", err.Error())

}
