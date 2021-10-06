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

package fftokens

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/mocks/wsmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/hyperledger/firefly/pkg/wsclient"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("tokens").Array()

func newTestFFTokens(t *testing.T) (h *FFTokens, toServer, fromServer chan string, httpURL string, done func()) {
	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)

	toServer, fromServer, wsURL, cancel := wsclient.NewTestWSServer(nil)

	u, _ := url.Parse(wsURL)
	u.Scheme = "http"
	httpURL = u.String()

	config.Reset()
	h = &FFTokens{}
	h.InitPrefix(utConfPrefix)

	utConfPrefix.AddKnownKey(tokens.TokensConfigName, "test")
	utConfPrefix.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	utConfPrefix.AddKnownKey(restclient.HTTPConfigURL, httpURL)
	utConfPrefix.AddKnownKey(restclient.HTTPCustomClient, mockedClient)
	config.Set("tokens", []fftypes.JSONObject{{}})

	err := h.Init(context.Background(), "testtokens", utConfPrefix.ArrayEntry(0), &tokenmocks.Callbacks{})
	assert.NoError(t, err)
	assert.Equal(t, "fftokens", h.Name())
	assert.Equal(t, "testtokens", h.configuredName)
	assert.NotNil(t, h.Capabilities())
	return h, toServer, fromServer, httpURL, func() {
		cancel()
		httpmock.DeactivateAndReset()
	}
}

func TestInitBadURL(t *testing.T) {
	config.Reset()
	h := &FFTokens{}
	h.InitPrefix(utConfPrefix)

	utConfPrefix.AddKnownKey(tokens.TokensConfigName, "test")
	utConfPrefix.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	utConfPrefix.AddKnownKey(restclient.HTTPConfigURL, "::::////")
	err := h.Init(context.Background(), "testtokens", utConfPrefix.ArrayEntry(0), &tokenmocks.Callbacks{})
	assert.Regexp(t, "FF10162", err)
}

func TestInitMissingURL(t *testing.T) {
	config.Reset()
	h := &FFTokens{}
	h.InitPrefix(utConfPrefix)

	utConfPrefix.AddKnownKey(tokens.TokensConfigName, "test")
	utConfPrefix.AddKnownKey(tokens.TokensConfigPlugin, "fftokens")
	utConfPrefix.AddKnownKey(restclient.HTTPConfigURL, "")
	err := h.Init(context.Background(), "testtokens", utConfPrefix.ArrayEntry(0), &tokenmocks.Callbacks{})
	assert.Regexp(t, "FF10138", err)
}

func TestCreateTokenPool(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "ns1",
		Name:      "new-pool",
		Type:      "fungible",
		Config: fftypes.JSONObject{
			"foo": "bar",
		},
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/pool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"requestId":  opID.String(),
				"trackingId": pool.TX.ID.String(),
				"type":       "fungible",
				"config": map[string]interface{}{
					"foo": "bar",
				},
			}, body)

			res := &http.Response{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"id":"1"}`))),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				StatusCode: 202,
			}
			return res, nil
		})

	err := h.CreateTokenPool(context.Background(), opID, pool)
	assert.NoError(t, err)
}

func TestCreateTokenPoolError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/pool", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.CreateTokenPool(context.Background(), fftypes.NewUUID(), pool)
	assert.Regexp(t, "FF10274", err)
}

func TestEvents(t *testing.T) {
	h, toServer, fromServer, _, done := newTestFFTokens(t)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	fromServer <- `!}`         // ignored
	fromServer <- `{}`         // ignored
	fromServer <- `{"id":"1"}` // ignored but acked
	msg := <-toServer
	assert.Equal(t, `{"data":{"id":"1"},"event":"ack"}`, string(msg))

	mcb := h.callbacks.(*tokenmocks.Callbacks)
	opID := fftypes.NewUUID()

	fromServer <- `{"id":"2","event":"receipt","data":{}}`
	fromServer <- `{"id":"3","event":"receipt","data":{"id":"abc"}}`

	// receipt: success
	mcb.On("TokensOpUpdate", h, opID, fftypes.OpStatusSucceeded, "", mock.Anything).Return(nil).Once()
	fromServer <- `{"id":"4","event":"receipt","data":{"id":"` + opID.String() + `","success":true}}`

	// receipt: failure
	mcb.On("TokensOpUpdate", h, opID, fftypes.OpStatusFailed, "", mock.Anything).Return(nil).Once()
	fromServer <- `{"id":"5","event":"receipt","data":{"id":"` + opID.String() + `","success":false}}`

	// token-pool: missing data
	fromServer <- `{"id":"6","event":"token-pool"}`
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"6"},"event":"ack"}`, string(msg))

	// token-pool: invalid uuid
	fromServer <- `{"id":"7","event":"token-pool","data":{"trackingId":"bad","type":"fungible","poolId":"F1","operator":"0x0","transaction":{"transactionHash":"abc"}}}`
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"7"},"event":"ack"}`, string(msg))

	txID := fftypes.NewUUID()

	// token-pool: success
	mcb.On("TokenPoolCreated", h, fftypes.TokenTypeFungible, txID, "F1", "0x0", "abc", fftypes.JSONObject{"transactionHash": "abc"}).Return(nil)
	fromServer <- `{"id":"8","event":"token-pool","data":{"trackingId":"` + txID.String() + `","type":"fungible","poolId":"F1","operator":"0x0","transaction":{"transactionHash":"abc"}}}`
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"8"},"event":"ack"}`, string(msg))

	mcb.AssertExpectations(t)
}

func TestEventLoopReceiveClosed(t *testing.T) {
	dxc := &tokenmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	h := &FFTokens{
		ctx:       context.Background(),
		callbacks: dxc,
		wsconn:    wsm,
	}
	r := make(chan []byte)
	close(r)
	wsm.On("Close").Return()
	wsm.On("Receive").Return((<-chan []byte)(r))
	h.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopSendClosed(t *testing.T) {
	dxc := &tokenmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	h := &FFTokens{
		ctx:       context.Background(),
		callbacks: dxc,
		wsconn:    wsm,
	}
	r := make(chan []byte, 1)
	r <- []byte(`{"id":"1"}`) // ignored but acked
	wsm.On("Close").Return()
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Send", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	h.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopClosedContext(t *testing.T) {
	dxc := &tokenmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := &FFTokens{
		ctx:       ctx,
		callbacks: dxc,
		wsconn:    wsm,
	}
	r := make(chan []byte, 1)
	wsm.On("Close").Return()
	wsm.On("Receive").Return((<-chan []byte)(r))
	h.eventLoop() // we're simply looking for it exiting
}
