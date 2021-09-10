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

package https

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/restclient"
	"github.com/hyperledger-labs/firefly/internal/wsclient"
	"github.com/hyperledger-labs/firefly/mocks/tokenmocks"
	"github.com/hyperledger-labs/firefly/mocks/wsmocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/tokens"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("tokens").Array()

func newTestHTTPS(t *testing.T) (h *HTTPS, toServer, fromServer chan string, httpURL string, done func()) {
	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)

	toServer, fromServer, wsURL, cancel := wsclient.NewTestWSServer(nil)

	u, _ := url.Parse(wsURL)
	u.Scheme = "http"
	httpURL = u.String()

	config.Reset()
	h = &HTTPS{}
	h.InitPrefix(utConfPrefix)

	utConfPrefix.AddKnownKey(tokens.TokensConfigName, "test")
	utConfPrefix.AddKnownKey(tokens.TokensConfigConnector, "https")
	utConfPrefix.AddKnownKey(restclient.HTTPConfigURL, httpURL)
	utConfPrefix.AddKnownKey(restclient.HTTPCustomClient, mockedClient)
	config.Set("tokens", []fftypes.JSONObject{{}})

	err := h.Init(context.Background(), utConfPrefix.ArrayEntry(0), &tokenmocks.Callbacks{})
	assert.NoError(t, err)
	assert.Equal(t, "https", h.Name())
	assert.NotNil(t, h.Capabilities())
	return h, toServer, fromServer, httpURL, func() {
		cancel()
		httpmock.DeactivateAndReset()
	}
}

func TestInitBadURL(t *testing.T) {
	config.Reset()
	h := &HTTPS{}
	h.InitPrefix(utConfPrefix)

	utConfPrefix.AddKnownKey(tokens.TokensConfigName, "test")
	utConfPrefix.AddKnownKey(tokens.TokensConfigConnector, "https")
	utConfPrefix.AddKnownKey(restclient.HTTPConfigURL, "::::////")
	err := h.Init(context.Background(), utConfPrefix.ArrayEntry(0), &tokenmocks.Callbacks{})
	assert.Regexp(t, "FF10162", err)
}

func TestInitMissingURL(t *testing.T) {
	config.Reset()
	h := &HTTPS{}
	h.InitPrefix(utConfPrefix)

	utConfPrefix.AddKnownKey(tokens.TokensConfigName, "test")
	utConfPrefix.AddKnownKey(tokens.TokensConfigConnector, "https")
	utConfPrefix.AddKnownKey(restclient.HTTPConfigURL, "")
	err := h.Init(context.Background(), utConfPrefix.ArrayEntry(0), &tokenmocks.Callbacks{})
	assert.Regexp(t, "FF10138", err)
}

func TestCreateTokenPool(t *testing.T) {
	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
		Namespace: "ns1",
		Name:      "new-pool",
		Type:      "fungible",
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/pool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Contains(t, body, "requestId")
			delete(body, "requestId")
			assert.Equal(t, fftypes.JSONObject{
				"data": "{\"namespace\":\"ns1\",\"name\":\"new-pool\",\"id\":\"" + pool.ID.String() + "\",\"transactionId\":\"" + pool.TX.ID.String() + "\"}",
				"type": "fungible",
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

	err := h.CreateTokenPool(context.Background(), &fftypes.Identity{}, pool)
	assert.NoError(t, err)
}

func TestCreateTokenPoolError(t *testing.T) {
	h, _, _, httpURL, done := newTestHTTPS(t)
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

	err := h.CreateTokenPool(context.Background(), &fftypes.Identity{}, pool)
	assert.Regexp(t, "FF10274", err)
}

func TestEvents(t *testing.T) {
	h, toServer, fromServer, _, done := newTestHTTPS(t)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	fromServer <- `!}`         // ignored
	fromServer <- `{}`         // ignored
	fromServer <- `{"id":"1"}` // ignored but acked
	msg := <-toServer
	assert.Equal(t, `{"data":{"id":"1"},"event":"ack"}`, string(msg))

	mcb := h.callbacks.(*tokenmocks.Callbacks)

	fromServer <- `{"id":"2","event":"receipt","data":{}}`

	// receipt: success
	mcb.On("TokensTxUpdate", h, "abc", fftypes.OpStatusSucceeded, "", mock.Anything).Return(nil).Once()
	fromServer <- `{"id":"3","event":"receipt","data":{"id":"abc","success":true}}`

	// receipt: failure
	mcb.On("TokensTxUpdate", h, "abc", fftypes.OpStatusFailed, "", mock.Anything).Return(nil).Once()
	fromServer <- `{"id":"4","event":"receipt","data":{"id":"abc","success":false}}`

	// token-pool: missing data
	fromServer <- `{"id":"5","event":"token-pool"}`
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"5"},"event":"ack"}`, string(msg))

	// token-pool: unparseable packed data
	fromServer <- `{"id":"6","event":"token-pool","data":{"data":"!","type":"fungible","poolId":"F1","operator":"0x0","transaction":{"transactionHash":"abc"}}}`
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"6"},"event":"ack"}`, string(msg))

	// token-pool: missing packed data
	fromServer <- `{"id":"7","event":"token-pool","data":{"data":"{}","type":"fungible","poolId":"F1","operator":"0x0","transaction":{"transactionHash":"abc"}}}`
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"7"},"event":"ack"}`, string(msg))

	// token-pool: invalid uuids
	fromServer <- `{"id":"8","event":"token-pool","data":{"data":"{\"namespace\":\"ns1\",\"name\":\"new-pool\",\"id\":\"bad\",\"transactionId\":\"bad\"}","type":"fungible","poolId":"F1","operator":"0x0","transaction":{"transactionHash":"abc"}}}`
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"8"},"event":"ack"}`, string(msg))

	id1 := fftypes.NewUUID()
	id2 := fftypes.NewUUID()

	// token-pool: success
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(pool *fftypes.TokenPool) bool {
		assert.Equal(t, "ns1", pool.Namespace)
		assert.Equal(t, "new-pool", pool.Name)
		assert.Equal(t, fftypes.TokenTypeFungible, pool.Type)
		assert.Equal(t, "F1", pool.ProtocolID)
		assert.Equal(t, *id1, *pool.ID)
		assert.Equal(t, *id2, *pool.TX.ID)
		return true
	}), "0x0", "abc", fftypes.JSONObject{"transactionHash": "abc"},
	).Return(nil)
	fromServer <- `{"id":"9","event":"token-pool","data":{"data":"{\"namespace\":\"ns1\",\"name\":\"new-pool\",\"id\":\"` + id1.String() + `\",\"transactionId\":\"` + id2.String() + `\"}","type":"fungible","poolId":"F1","operator":"0x0","transaction":{"transactionHash":"abc"}}}`
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"9"},"event":"ack"}`, string(msg))

	mcb.AssertExpectations(t)
}

func TestEventLoopReceiveClosed(t *testing.T) {
	dxc := &tokenmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	h := &HTTPS{
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
	h := &HTTPS{
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
	h := &HTTPS{
		ctx:       ctx,
		callbacks: dxc,
		wsconn:    wsm,
	}
	r := make(chan []byte, 1)
	wsm.On("Close").Return()
	wsm.On("Receive").Return((<-chan []byte)(r))
	h.eventLoop() // we're simply looking for it exiting
}
