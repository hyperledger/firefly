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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/wsmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfig = config.RootSection("ffdx_unit_tests")

func newTestFFDX(t *testing.T, manifestEnabled bool) (h *FFDX, toServer, fromServer chan string, httpURL string, done func()) {
	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)

	toServer, fromServer, wsURL, cancel := wsclient.NewTestWSServer(nil)

	u, _ := url.Parse(wsURL)
	u.Scheme = "http"
	httpURL = u.String()

	coreconfig.Reset()
	h.InitConfig(utConfig)
	utConfig.Set(ffresty.HTTPConfigURL, httpURL)
	utConfig.Set(ffresty.HTTPCustomClient, mockedClient)
	utConfig.Set(DataExchangeManifestEnabled, manifestEnabled)

	h = &FFDX{initialized: true}
	h.InitConfig(utConfig)

	dxCtx, dxCancel := context.WithCancel(context.Background())
	err := h.Init(dxCtx, utConfig)
	assert.NoError(t, err)
	assert.Equal(t, "ffdx", h.Name())
	assert.NotNil(t, h.Capabilities())
	return h, toServer, fromServer, httpURL, func() {
		cancel()
		dxCancel()
		httpmock.DeactivateAndReset()
	}
}

func TestInitBadURL(t *testing.T) {
	coreconfig.Reset()
	h := &FFDX{}
	h.InitConfig(utConfig)
	utConfig.Set(ffresty.HTTPConfigURL, "::::////")
	err := h.Init(context.Background(), utConfig)
	assert.Regexp(t, "FF00149", err)
}

func TestInitMissingURL(t *testing.T) {
	coreconfig.Reset()
	h := &FFDX{}
	h.InitConfig(utConfig)
	err := h.Init(context.Background(), utConfig)
	assert.Regexp(t, "FF10138", err)
}

func acker() func(args mock.Arguments) {
	return func(args mock.Arguments) {
		args[0].(dataexchange.DXEvent).Ack()
	}
}

func manifestAcker(manifest string) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		args[0].(dataexchange.DXEvent).AckWithManifest(manifest)
	}
}

func TestGetEndpointInfo(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/id", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{
			"id":       "peer1",
			"endpoint": "https://peer1.example.com",
			"cert":     "cert data...",
		}))

	peer, err := h.GetEndpointInfo(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "peer1", peer.GetString("id"))
	assert.Equal(t, fftypes.JSONObject{
		"id":       "peer1",
		"endpoint": "https://peer1.example.com",
		"cert":     "cert data...",
	}, peer)
}

func TestGetEndpointMissingID(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/id", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{
			"endpoint": "https://peer1.example.com",
			"cert":     "cert data...",
		}))

	_, err := h.GetEndpointInfo(context.Background())
	assert.Regexp(t, "FF10367", err)
}

func TestGetEndpointInfoError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/id", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	_, err := h.GetEndpointInfo(context.Background())
	assert.Regexp(t, "FF10229", err)
}

func TestAckClosed(t *testing.T) {
	h, _, _, _, done := newTestFFDX(t, false)
	done()

	dxe := &dxEvent{
		ffdx: h,
	}
	dxe.AckWithManifest("")
}

func TestAddPeer(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/peers/peer1", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{}))

	err := h.AddPeer(context.Background(), fftypes.JSONObject{
		"id":       "peer1",
		"endpoint": "https://peer1.example.com",
		"cert":     "cert...",
	})
	assert.NoError(t, err)
}

func TestAddPeerError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/peers/peer1", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.AddPeer(context.Background(), fftypes.JSONObject{
		"id": "peer1",
	})
	assert.Regexp(t, "FF10229", err)
}

func TestUploadBlob(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
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

	payloadRef, hashReturned, sizeReturned, err := h.UploadBlob(context.Background(), "ns1", *u, bytes.NewReader([]byte(`{}`)))
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("ns1/%s", u.String()), payloadRef)
	assert.Equal(t, *hash, *hashReturned)
	assert.Equal(t, int64(12345), sizeReturned)
}

func TestUploadBlobBadHash(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
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

	_, _, _, err := h.UploadBlob(context.Background(), "ns1", *u, bytes.NewReader([]byte(`{}`)))
	assert.Regexp(t, "FF10237", err)
}

func TestUploadBlobError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	_, _, _, err := h.UploadBlob(context.Background(), "ns1", *u, bytes.NewReader([]byte(`{}`)))
	assert.Regexp(t, "FF10229", err)
}

func TestCheckBlobReceivedOk(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
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

	hashReturned, size, err := h.CheckBlobReceived(context.Background(), "peer1", "ns1", *u)
	assert.NoError(t, err)
	assert.Equal(t, *hash, *hashReturned)
	assert.Equal(t, int64(size), size)
}

func TestCheckBlobReceivedBadHash(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
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

	_, _, err := h.CheckBlobReceived(context.Background(), "peer1", "ns1", *u)
	assert.Regexp(t, "FF10237", err)
}

func TestCheckBlobReceivedBadSize(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
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

	_, _, err := h.CheckBlobReceived(context.Background(), "peer1", "ns1", *u)
	assert.Regexp(t, "FF10237", err)
}

func TestCheckBlobReceivedNotFound(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("HEAD", fmt.Sprintf("%s/api/v1/blobs/peer1/ns1/%s", httpURL, u),
		func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				StatusCode: 404,
			}
			return res, nil
		})

	hashReturned, _, err := h.CheckBlobReceived(context.Background(), "peer1", "ns1", *u)
	assert.NoError(t, err)
	assert.Nil(t, hashReturned)
}

func TestCheckBlobReceivedError(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("HEAD", fmt.Sprintf("%s/api/v1/blobs/peer1/ns1/%s", httpURL, u),
		func(r *http.Request) (*http.Response, error) {
			res := &http.Response{
				StatusCode: 500,
			}
			return res, nil
		})

	_, _, err := h.CheckBlobReceived(context.Background(), "peer1", "ns1", *u)
	assert.Regexp(t, "FF10229", err)
}

func TestDownloadBlob(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	u := fftypes.NewUUID()
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/blobs/ns1/%s", httpURL, u),
		httpmock.NewBytesResponder(200, []byte(`some data`)))

	rc, err := h.DownloadBlob(context.Background(), fmt.Sprintf("ns1/%s", u))
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(rc)
	rc.Close()
	assert.Equal(t, `some data`, string(b))
}

func TestDownloadBlobError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/blobs/bad", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	_, err := h.DownloadBlob(context.Background(), "bad")
	assert.Regexp(t, "FF10229", err)
}

func TestSendMessage(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/messages", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{}))

	err := h.SendMessage(context.Background(), "ns1:"+fftypes.NewUUID().String(), "peer1", []byte(`some data`))
	assert.NoError(t, err)
}

func TestSendMessageError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/message", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.SendMessage(context.Background(), "ns1:"+fftypes.NewUUID().String(), "peer1", []byte(`some data`))
	assert.Regexp(t, "FF10229", err)
}

func TestTransferBlob(t *testing.T) {

	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfers", httpURL),
		httpmock.NewJsonResponderOrPanic(200, fftypes.JSONObject{}))

	err := h.TransferBlob(context.Background(), "ns1:"+fftypes.NewUUID().String(), "peer1", "ns1/id1")
	assert.NoError(t, err)
}

func TestTransferBlobError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFDX(t, false)
	defer done()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfers", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.TransferBlob(context.Background(), "ns1:"+fftypes.NewUUID().String(), "peer1", "ns1/id1")
	assert.Regexp(t, "FF10229", err)
}

func TestEvents(t *testing.T) {

	h, toServer, fromServer, _, done := newTestFFDX(t, false)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	fromServer <- `!}`         // ignored without ack
	fromServer <- `{"id":"0"}` // ignored with ack
	msg := <-toServer
	assert.Equal(t, `{"action":"ack","id":"0"}`, string(msg))

	mcb := &dataexchangemocks.Callbacks{}
	h.callbacks = mcb

	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "1" &&
			ev.Type() == dataexchange.DXEventTypeTransferResult &&
			ev.TransferResult().TrackingID == "tx12345" &&
			ev.TransferResult().Status == core.OpStatusFailed &&
			ev.TransferResult().Error == "pop"
	})).Run(acker()).Return(nil)
	fromServer <- `{"id":"1","type":"message-failed","requestID":"tx12345","error":"pop"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"1"}`, string(msg))

	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "2" &&
			ev.Type() == dataexchange.DXEventTypeTransferResult &&
			ev.TransferResult().TrackingID == "tx12345" &&
			ev.TransferResult().Status == core.OpStatusSucceeded
	})).Run(acker()).Return(nil)
	fromServer <- `{"id":"2","type":"message-delivered","requestID":"tx12345"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"2"}`, string(msg))

	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "3" &&
			ev.Type() == dataexchange.DXEventTypeTransferResult &&
			ev.TransferResult().TrackingID == "tx12345" &&
			ev.TransferResult().Status == core.OpStatusSucceeded &&
			ev.TransferResult().Manifest == `{"manifest":true}` &&
			ev.TransferResult().Info.String() == `{"signatures":"and stuff"}`
	})).Run(acker()).Return(nil)
	fromServer <- `{"id":"3","type":"message-acknowledged","requestID":"tx12345","info":{"signatures":"and stuff"},"manifest":"{\"manifest\":true}"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"3"}`, string(msg))

	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "4" &&
			ev.Type() == dataexchange.DXEventTypeMessageReceived &&
			ev.MessageReceived().PeerID == "peer1" &&
			string(ev.MessageReceived().Data) == "message1"
	})).Run(manifestAcker(`{"manifest":true}`)).Return(nil)
	fromServer <- `{"id":"4","type":"message-received","sender":"peer1","message":"message1"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"4","manifest":"{\"manifest\":true}"}`, string(msg))

	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "5" &&
			ev.Type() == dataexchange.DXEventTypeTransferResult &&
			ev.TransferResult().TrackingID == "tx12345" &&
			ev.TransferResult().Status == core.OpStatusFailed &&
			ev.TransferResult().Error == "pop"
	})).Run(acker()).Return(nil)
	fromServer <- `{"id":"5","type":"blob-failed","requestID":"tx12345","error":"pop"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"5"}`, string(msg))

	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "6" &&
			ev.Type() == dataexchange.DXEventTypeTransferResult &&
			ev.TransferResult().TrackingID == "tx12345" &&
			ev.TransferResult().Status == core.OpStatusSucceeded &&
			ev.TransferResult().Info.String() == `{"some":"details"}`
	})).Run(acker()).Return(nil)
	fromServer <- `{"id":"6","type":"blob-delivered","requestID":"tx12345","info":{"some":"details"}}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"6"}`, string(msg))

	fromServer <- `{"id":"7","type":"blob-received","sender":"peer1","path":"ns1/! not a UUID"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"7"}`, string(msg))

	u := fftypes.NewUUID()
	fromServer <- fmt.Sprintf(`{"id":"8","type":"blob-received","sender":"peer1","path":"ns1/%s","hash":"!wrong","size":-1}`, u.String())
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"8"}`, string(msg))

	hash := fftypes.NewRandB32()
	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "9" &&
			ev.Type() == dataexchange.DXEventTypePrivateBlobReceived &&
			ev.PrivateBlobReceived().Hash.Equals(hash)
	})).Run(acker()).Return(nil)
	fromServer <- fmt.Sprintf(`{"id":"9","type":"blob-received","sender":"peer1","path":"ns1/%s","hash":"%s","size":12345}`, u.String(), hash.String())
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"9"}`, string(msg))

	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "10" &&
			ev.Type() == dataexchange.DXEventTypeTransferResult &&
			ev.TransferResult().TrackingID == "tx12345" &&
			ev.TransferResult().Status == core.OpStatusSucceeded &&
			ev.TransferResult().Info.String() == `{"signatures":"and stuff"}`
	})).Run(acker()).Return(nil)
	fromServer <- `{"id":"10","type":"blob-acknowledged","requestID":"tx12345","info":{"signatures":"and stuff"},"manifest":"{\"manifest\":true}"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"10"}`, string(msg))

	mcb.AssertExpectations(t)
}

func TestEventsWithManifest(t *testing.T) {

	h, toServer, fromServer, _, done := newTestFFDX(t, true)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	fromServer <- `!}`         // ignored without ack
	fromServer <- `{"id":"0"}` // ignored with ack
	msg := <-toServer
	assert.Equal(t, `{"action":"ack","id":"0"}`, string(msg))

	mcb := &dataexchangemocks.Callbacks{}
	h.callbacks = mcb

	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "1" &&
			ev.Type() == dataexchange.DXEventTypeTransferResult &&
			ev.TransferResult().Status == core.OpStatusPending
	})).Run(acker()).Return(nil)
	fromServer <- `{"id":"1","type":"message-delivered","requestID":"tx12345"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"1"}`, string(msg))

	mcb.On("DXEvent", mock.MatchedBy(func(ev dataexchange.DXEvent) bool {
		return ev.NamespacedID() == "2" &&
			ev.Type() == dataexchange.DXEventTypeTransferResult &&
			ev.TransferResult().Status == core.OpStatusPending
	})).Run(acker()).Return(nil)
	fromServer <- `{"id":"2","type":"blob-delivered","requestID":"tx12345"}`
	msg = <-toServer
	assert.Equal(t, `{"action":"ack","id":"2"}`, string(msg))

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
	ctx, cancelCtx := context.WithCancel(context.Background())
	h := &FFDX{
		ctx:        ctx,
		callbacks:  dxc,
		wsconn:     wsm,
		ackChannel: make(chan *ack, 1),
	}
	h.ackChannel <- &ack{
		eventID: "12345",
	}
	wsm.On("Close").Return()
	wsm.On("Send", mock.Anything, mock.Anything).Return(fmt.Errorf("pop")).Run(func(args mock.Arguments) {
		cancelCtx()
	})
	h.ackLoop() // we're simply looking for it exiting
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

func TestWebsocketWithReinit(t *testing.T) {
	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	_, _, wsURL, cancel := wsclient.NewTestWSServer(nil)
	defer cancel()

	u, _ := url.Parse(wsURL)
	u.Scheme = "http"
	httpURL := u.String()
	h := &FFDX{}
	nodes := []fftypes.JSONObject{{}}

	coreconfig.Reset()
	h.InitConfig(utConfig)
	utConfig.Set(ffresty.HTTPConfigURL, httpURL)
	utConfig.Set(ffresty.HTTPCustomClient, mockedClient)
	utConfig.Set(DataExchangeInitEnabled, true)

	count := 0
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/init", httpURL),
		func(req *http.Request) (*http.Response, error) {
			var reqNodes []fftypes.JSONObject
			err := json.NewDecoder(req.Body).Decode(&reqNodes)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(reqNodes))

			assert.False(t, h.initialized)

			count++
			if count == 1 {
				return httpmock.NewJsonResponse(200, fftypes.JSONObject{
					"status": "notready",
				})
			}
			if count == 2 {
				return nil, fmt.Errorf("pop")
			}
			return httpmock.NewJsonResponse(200, fftypes.JSONObject{
				"status": "ready",
			})
		})

	h.InitConfig(utConfig)
	err := h.Init(context.Background(), utConfig)
	assert.NoError(t, err)
	h.SetNodes(nodes)

	err = h.Start()
	assert.NoError(t, err)

	assert.Equal(t, 3, httpmock.GetTotalCallCount())
	assert.True(t, h.initialized)
}

func TestDXUninitialized(t *testing.T) {
	h, _, _, _, done := newTestFFDX(t, false)
	defer done()

	h.initialized = false

	_, err := h.GetEndpointInfo(context.Background())
	assert.Regexp(t, "FF10342", err)

	err = h.AddPeer(context.Background(), fftypes.JSONObject{})
	assert.Regexp(t, "FF10342", err)

	err = h.TransferBlob(context.Background(), "ns1:"+fftypes.NewUUID().String(), "peer1", "ns1/id1")
	assert.Regexp(t, "FF10342", err)

	err = h.SendMessage(context.Background(), "ns1:"+fftypes.NewUUID().String(), "peer1", []byte(`some data`))
	assert.Regexp(t, "FF10342", err)
}
