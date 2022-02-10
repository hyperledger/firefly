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

package ffdx

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/wsmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/wsclient"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("ffdx_unit_tests")

func newTestFFDX(t *testing.T) (h *FFDX, toServer, fromServer chan string, httpURL string, done func()) {
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

	h = &FFDX{}
	h.InitPrefix(utConfPrefix)
	err := h.Init(context.Background(), utConfPrefix, &dataexchangemocks.Callbacks{})
	assert.NoError(t, err)
	assert.Equal(t, "ffdx", h.Name())
	assert.NotNil(t, h.Capabilities())
	return h, toServer, fromServer, httpURL, func() {
		cancel()
		httpmock.DeactivateAndReset()
	}
}

func TestInitBadURL(t *testing.T) {
	config.Reset()
	h := &FFDX{}
	h.InitPrefix(utConfPrefix)
	utConfPrefix.Set(restclient.HTTPConfigURL, "::::////")
	err := h.Init(context.Background(), utConfPrefix, &dataexchangemocks.Callbacks{})
	assert.Regexp(t, "FF10162", err)
}

func TestInitMissingURL(t *testing.T) {
	config.Reset()
	h := &FFDX{}
	h.InitPrefix(utConfPrefix)
	err := h.Init(context.Background(), utConfPrefix, &dataexchangemocks.Callbacks{})
	assert.Regexp(t, "FF10138", err)
}

func TestGetEndpointInfo(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
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
	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/id", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	_, _, err := h.GetEndpointInfo(context.Background())
	assert.Regexp(t, "FF10229", err)
}

func TestAddPeer(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/peers/peer1", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{}))

	err := h.AddPeer(context.Background(), "peer1",
		fftypes.JSONObject{
			"id":       "peer1",
			"endpoint": "https://peer1.example.com",
			"cert":     "cert...",
		},
	)
	assert.NoError(t, err)
}

func TestAddPeerError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/peers/peer1", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.AddPeer(context.Background(), "peer1", fftypes.JSONObject{})
	assert.Regexp(t, "FF10229", err)
}

func TestUploadBLOB(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	u := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{
					"hash": "%s",
					"size": 12345
				}`, hash)))),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"Dx-Hash":      []string{hash.String()},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	payloadRef, hashReturned, sizeReturned, err := h.UploadBLOB(context.Background(), "ns1", *u, bytes.NewReader([]byte(`{}`)))
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("ns1/%s", u.String()), payloadRef)
	assert.Equal(t, *hash, *hashReturned)
	assert.Equal(t, int64(12345), sizeReturned)
}

func TestUploadBLOBBadHash(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
				Header: http.Header{
					"Dx-Hash": []string{"!hash"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	_, _, _, err := h.UploadBLOB(context.Background(), "ns1", *u, bytes.NewReader([]byte(`{}`)))
	assert.Regexp(t, "FF10237", err)
}

func TestUploadBLOBError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	_, _, _, err := h.UploadBLOB(context.Background(), "ns1", *u, bytes.NewReader([]byte(`{}`)))
	assert.Regexp(t, "FF10229", err)
}

func TestCheckBLOBReceivedOk(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	u := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	httpmock.RegisterResponder("HEAD", fmt.Sprintf("%s/api/v1/blobs/peer1/ns1/%s", httpURL, u),
		func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				Header: http.Header{
					"Dx-Hash": []string{hash.String()},
					"Dx-Size": []string{"12345"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	hashReturned, size, err := h.CheckBLOBReceived(context.Background(), "peer1", "ns1", *u)
	assert.NoError(t, err)
	assert.Equal(t, *hash, *hashReturned)
	assert.Equal(t, int64(size), size)
}

func TestCheckBLOBReceivedBadHash(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("HEAD", fmt.Sprintf("%s/api/v1/blobs/peer1/ns1/%s", httpURL, u),
		func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				Header: http.Header{
					"Dx-Hash": []string{"!wrong"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	_, _, err := h.CheckBLOBReceived(context.Background(), "peer1", "ns1", *u)
	assert.Regexp(t, "FF10237", err)
}

func TestCheckBLOBReceivedBadSize(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	u := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	httpmock.RegisterResponder("HEAD", fmt.Sprintf("%s/api/v1/blobs/peer1/ns1/%s", httpURL, u),
		func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				Header: http.Header{
					"Dx-Hash": []string{hash.String()},
					"Dx-Size": []string{"bob"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	_, _, err := h.CheckBLOBReceived(context.Background(), "peer1", "ns1", *u)
	assert.Regexp(t, "FF10237", err)
}

func TestCheckBLOBReceivedNotFound(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("HEAD", fmt.Sprintf("%s/api/v1/blobs/peer1/ns1/%s", httpURL, u),
		func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				StatusCode: 404,
			}
			return res, nil
		})

	hashReturned, _, err := h.CheckBLOBReceived(context.Background(), "peer1", "ns1", *u)
	assert.NoError(t, err)
	assert.Nil(t, hashReturned)
}

func TestCheckBLOBReceivedError(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("HEAD", fmt.Sprintf("%s/api/v1/blobs/peer1/ns1/%s", httpURL, u),
		func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				StatusCode: 500,
			}
			return res, nil
		})

	_, _, err := h.CheckBLOBReceived(context.Background(), "peer1", "ns1", *u)
	assert.Regexp(t, "FF10229", err)
}

func TestDownloadBLOB(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		httpmock.NewBytesResponder(200, []byte(`some data`)))

	rc, err := h.DownloadBLOB(context.Background(), fmt.Sprintf("ns1/%s", u))
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(rc)
	rc.Close()
	assert.Equal(t, `some data`, string(b))
}

func TestDownloadBLOBError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/blobs/bad", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	_, err := h.DownloadBLOB(context.Background(), "bad")
	assert.Regexp(t, "FF10229", err)
}

func TestSendMessage(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/messages", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{}))

	err := h.SendMessage(context.Background(), fftypes.NewUUID(), "peer1", []byte(`some data`))
	assert.NoError(t, err)
}

func TestSendMessageError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/message", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.SendMessage(context.Background(), fftypes.NewUUID(), "peer1", []byte(`some data`))
	assert.Regexp(t, "FF10229", err)
}

func TestTransferBLOB(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfers", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{}))

	err := h.TransferBLOB(context.Background(), fftypes.NewUUID(), "peer1", "ns1/id1")
	assert.NoError(t, err)
}

func TestTransferBLOBError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfers", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.TransferBLOB(context.Background(), fftypes.NewUUID(), "peer1", "ns1/id1")
	assert.Regexp(t, "FF10229", err)
}

func TestEvents(t *testing.T) {

	h, toServer, fromServer, _, done := newTestFFDX(t)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	fromServer <- `!}` // ignored
	fromServer <- `{}` // ignored
	msg := <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb := h.callbacks.(*dataexchangemocks.Callbacks)

	mcb.On("TransferResult", "tx12345", fftypes.OpStatusFailed, mock.MatchedBy(func(ts fftypes.TransportStatusUpdate) bool {
		return "pop" == ts.Error
	})).Return(nil)
	fromServer <- `{"type":"message-failed","requestID":"tx12345","error":"pop"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.On("TransferResult", "tx12345", fftypes.OpStatusSucceeded, mock.MatchedBy(func(ts fftypes.TransportStatusUpdate) bool {
		return ts.Manifest == `{"manifest":true}` && ts.Info.String() == `{"signatures":"and stuff"}`
	})).Return(nil)
	fromServer <- `{"type":"message-delivered","requestID":"tx12345","info":{"signatures":"and stuff"},"manifest":"{\"manifest\":true}"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.On("MessageReceived", "peer1", []byte("message1")).Return(`{"manifest":true}`, nil)
	fromServer <- `{"type":"message-received","sender":"peer1","message":"message1"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit","manifest":"{\"manifest\":true}"}`, string(msg))

	mcb.On("TransferResult", "tx12345", fftypes.OpStatusFailed, mock.MatchedBy(func(ts fftypes.TransportStatusUpdate) bool {
		return "pop" == ts.Error
	})).Return(nil)
	fromServer <- `{"type":"blob-failed","requestID":"tx12345","error":"pop"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.On("TransferResult", "tx12345", fftypes.OpStatusSucceeded, mock.Anything).Return(nil)
	fromServer <- `{"type":"blob-delivered","requestID":"tx12345"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	fromServer <- `{"type":"blob-received","sender":"peer1","path":"ns1/! not a UUID"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	u := fftypes.NewUUID()
	fromServer <- fmt.Sprintf(`{"type":"blob-received","sender":"peer1","path":"ns1/%s","hash":"!wrong","size":-1}`, u.String())
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	hash := fftypes.NewRandB32()
	mcb.On("BLOBReceived", mock.Anything, mock.MatchedBy(func(b32 fftypes.Bytes32) bool {
		return b32 == *hash
	}), int64(12345), fmt.Sprintf("ns1/%s", u.String())).Return(nil)
	fromServer <- fmt.Sprintf(`{"type":"blob-received","sender":"peer1","path":"ns1/%s","hash":"%s","size":12345}`, u.String(), hash.String())
	msg = <-toServer
	assert.Equal(t, `{"action":"commit"}`, string(msg))

	mcb.AssertExpectations(t)
}

func TestEventLoopReceiveClosed(t *testing.T) {
	dxc := &dataexchangemocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	h := &FFDX{
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
	dxc := &dataexchangemocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	h := &FFDX{
		ctx:       context.Background(),
		callbacks: dxc,
		wsconn:    wsm,
	}
	r := make(chan []byte, 1)
	r <- []byte(`{}`)
	wsm.On("Close").Return()
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Send", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	h.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopClosedContext(t *testing.T) {
	dxc := &dataexchangemocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := &FFDX{
		ctx:       ctx,
		callbacks: dxc,
		wsconn:    wsm,
	}
	r := make(chan []byte, 1)
	r <- []byte(`{}`)
	wsm.On("Close").Return()
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Send", mock.Anything, mock.Anything).Return(nil)
	h.eventLoop() // we're simply looking for it exiting
}
