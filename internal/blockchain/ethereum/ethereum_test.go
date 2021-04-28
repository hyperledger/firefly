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

	"github.com/jarcoal/httpmock"
	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/ffresty"
	"github.com/stretchr/testify/assert"
)

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
