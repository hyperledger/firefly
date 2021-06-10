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

package dxhttps

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/restclient"
	"github.com/hyperledger-labs/firefly/internal/wsclient"
	"github.com/hyperledger-labs/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger-labs/firefly/mocks/wsmocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("dxhttps_unit_tests")

func newTestHTTPS(t *testing.T) (h *HTTPS, toServer, fromServer chan string, httpURL string, done func()) {
	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)

	toServer, fromServer, wsURL, cancel := wsclient.NewTestWSServer(nil)

	u, _ := url.Parse(wsURL)
	u.Scheme = "http"
	httpURL = u.String()

	config.Reset()
	h.InitPrefix(utConfPrefix)
	utConfPrefix.Set(restclient.HTTPConfigURL, httpURL)
	utConfPrefix.Set(restclient.HTTPCustomClient, mockedClient)

	h = &HTTPS{}
	h.InitPrefix(utConfPrefix)
	err := h.Init(context.Background(), utConfPrefix, &dataexchangemocks.Callbacks{})
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
	utConfPrefix.Set(restclient.HTTPConfigURL, "::::////")
	err := h.Init(context.Background(), utConfPrefix, &dataexchangemocks.Callbacks{})
	assert.Regexp(t, "FF10162", err)
}

func TestInitMissingURL(t *testing.T) {
	config.Reset()
	h := &HTTPS{}
	h.InitPrefix(utConfPrefix)
	err := h.Init(context.Background(), utConfPrefix, &dataexchangemocks.Callbacks{})
	assert.Regexp(t, "FF10138", err)
}

func TestGetEndpointInfo(t *testing.T) {

	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/id", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{
			"id":       "peer1",
			"endpoint": "https://peer1.example.com",
			"cert":     "cert data...",
		}))

	peerID, endpoint, err := h.GetEndpointInfo(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "peer1", peerID)
	assert.Equal(t, fftypes.JSONObject{
		"id":       "peer1",
		"endpoint": "https://peer1.example.com",
		"cert":     "cert data...",
	}, endpoint)
}

func TestGetEndpointInfoError(t *testing.T) {
	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/id", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	_, _, err := h.GetEndpointInfo(context.Background())
	assert.Regexp(t, "FF10229", err)
}

func TestAddPeer(t *testing.T) {

	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/peers/peer1", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{}))

	err := h.AddPeer(context.Background(), &fftypes.Node{
		DX: fftypes.DXInfo{
			Peer: "peer1",
			Endpoint: fftypes.JSONObject{
				"id":       "peer1",
				"endpoint": "https://peer1.example.com",
				"cert":     "cert...",
			},
		},
	})
	assert.NoError(t, err)
}

func TestAddPeerError(t *testing.T) {
	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/peers/peer1", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.AddPeer(context.Background(), &fftypes.Node{})
	assert.Regexp(t, "FF10229", err)
}

func TestUploadBLOB(t *testing.T) {

	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{}))

	err := h.UploadBLOB(context.Background(), "ns1", *u, bytes.NewReader([]byte(`{}`)))
	assert.NoError(t, err)
}

func TestUploadBLOBError(t *testing.T) {
	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.UploadBLOB(context.Background(), "ns1", *u, bytes.NewReader([]byte(`{}`)))
	assert.Regexp(t, "FF10229", err)
}

func TestDownloadBLOB(t *testing.T) {

	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		httpmock.NewBytesResponder(200, []byte(`some data`)))

	rc, err := h.DownloadBLOB(context.Background(), "ns1", *u)
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(rc)
	rc.Close()
	assert.Equal(t, `some data`, string(b))
}

func TestDownloadBLOBError(t *testing.T) {
	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	_, err := h.DownloadBLOB(context.Background(), "ns1", *u)
	assert.Regexp(t, "FF10229", err)
}

func TestSendMessage(t *testing.T) {

	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/messages", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{
			"requestID": "abcd1234",
		}))

	trackingID, err := h.SendMessage(context.Background(), &fftypes.Node{}, []byte(`some data`))
	assert.NoError(t, err)
	assert.Equal(t, "abcd1234", trackingID)
}

func TestSendMessageError(t *testing.T) {
	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/message", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	_, err := h.SendMessage(context.Background(), &fftypes.Node{}, []byte(`some data`))
	assert.Regexp(t, "FF10229", err)
}

func TestTransferBLOB(t *testing.T) {

	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfers", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{
			"requestID": "abcd1234",
		}))

	u := fftypes.NewUUID()
	trackingID, err := h.TransferBLOB(context.Background(), &fftypes.Node{}, "ns1", *u)
	assert.NoError(t, err)
	assert.Equal(t, "abcd1234", trackingID)
}

func TestTransferBLOBError(t *testing.T) {
	h, _, _, httpURL, done := newTestHTTPS(t)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfers", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	u := fftypes.NewUUID()
	_, err := h.TransferBLOB(context.Background(), &fftypes.Node{}, "ns1", *u)
	assert.Regexp(t, "FF10229", err)
}

func TestExtractBlobPath(t *testing.T) {
	h := &HTTPS{}

	ns, id := h.extractBlobPath(context.Background(), "")
	assert.Empty(t, ns)
	assert.Nil(t, id)

	ns, id = h.extractBlobPath(context.Background(), "/")
	assert.Empty(t, ns)
	assert.Nil(t, id)

	ns, id = h.extractBlobPath(context.Background(), "abc/!UUID")
	assert.Empty(t, ns)
	assert.Nil(t, id)

	ns, id = h.extractBlobPath(context.Background(), fmt.Sprintf("abc/%s", fftypes.NewUUID()))
	assert.Equal(t, "abc", ns)
	assert.NotNil(t, id)
}

func TestEvents(t *testing.T) {

	h, toServer, fromServer, _, done := newTestHTTPS(t)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	fromServer <- `!}` // ignored
	fromServer <- `{}` // ignored
	msg := <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb := h.callbacks.(*dataexchangemocks.Callbacks)

	mcb.On("TransferResult", "tx12345", fftypes.OpStatusFailed, "pop", mock.Anything).Return()
	fromServer <- `{"type":"message-failed","requestID":"tx12345","error":"pop"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.On("TransferResult", "tx12345", fftypes.OpStatusSucceeded, "", mock.Anything).Return()
	fromServer <- `{"type":"message-delivered","requestID":"tx12345"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.On("MessageReceived", "peer1", []byte("message1")).Return()
	fromServer <- `{"type":"message-received","sender":"peer1","message":"message1"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.On("TransferResult", "tx12345", fftypes.OpStatusFailed, "pop", mock.Anything).Return()
	fromServer <- `{"type":"blob-failed","requestID":"tx12345","error":"pop"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.On("TransferResult", "tx12345", fftypes.OpStatusSucceeded, "", mock.Anything).Return()
	fromServer <- `{"type":"blob-delivered","requestID":"tx12345"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	fromServer <- `{"type":"blob-received","sender":"peer1","path":"ns1/! not a UUID"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.On("BLOBReceived", "peer1", "ns1", mock.Anything).Return()
	fromServer <- fmt.Sprintf(`{"type":"blob-received","sender":"peer1","path":"ns1/%s"}`, fftypes.NewUUID())
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.AssertExpectations(t)

}
func TestEventLoopReceiveClosed(t *testing.T) {
	dxc := &dataexchangemocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	h := &HTTPS{
		ctx:       context.Background(),
		callbacks: dxc,
		wsconn:    wsm,
	}
	r := make(chan []byte)
	close(r)
	wsm.On("Receive").Return((<-chan []byte)(r))
	h.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopSendClosed(t *testing.T) {
	dxc := &dataexchangemocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	h := &HTTPS{
		ctx:       context.Background(),
		callbacks: dxc,
		wsconn:    wsm,
	}
	r := make(chan []byte, 1)
	r <- []byte(`{}`)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Send", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	h.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopClosedContext(t *testing.T) {
	dxc := &dataexchangemocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := &HTTPS{
		ctx:       ctx,
		callbacks: dxc,
		wsconn:    wsm,
	}
	r := make(chan []byte, 1)
	r <- []byte(`{}`)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Send", mock.Anything, mock.Anything).Return(nil)
	h.eventLoop() // we're simply looking for it exiting
}
