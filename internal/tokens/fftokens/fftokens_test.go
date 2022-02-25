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
		Key:       "0x123",
		Config: fftypes.JSONObject{
			"foo": "bar",
		},
		Symbol: "symbol",
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/createpool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"requestId": opID.String(),
				"signer":    "0x123",
				"type":      "fungible",
				"config": map[string]interface{}{
					"foo": "bar",
				},
				"data":   `{"tx":"` + pool.TX.ID.String() + `"}`,
				"name":   "new-pool",
				"symbol": "symbol",
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

	complete, err := h.CreateTokenPool(context.Background(), opID, pool)
	assert.False(t, complete)
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

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/createpool", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	complete, err := h.CreateTokenPool(context.Background(), fftypes.NewUUID(), pool)
	assert.False(t, complete)
	assert.Regexp(t, "FF10274", err)
}

func TestCreateTokenPoolSynchronous(t *testing.T) {
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
		Key:       "0x123",
		Config: fftypes.JSONObject{
			"foo": "bar",
		},
		Symbol: "symbol",
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/createpool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)

			res := &http.Response{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(fftypes.JSONObject{
					"id":     "000000000010/000020/000030/000040",
					"type":   "fungible",
					"poolId": "F1",
					"signer": "0x0",
					"data":   `{"tx":"` + pool.TX.ID.String() + `"}`,
				}.String()))),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	mcb := h.callbacks.(*tokenmocks.Callbacks)
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.ProtocolID == "F1" && p.Type == fftypes.TokenTypeFungible && *p.TransactionID == *pool.TX.ID && p.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil)

	complete, err := h.CreateTokenPool(context.Background(), opID, pool)
	assert.True(t, complete)
	assert.NoError(t, err)
}

func TestCreateTokenPoolSynchronousBadResponse(t *testing.T) {
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
		Key:       "0x123",
		Config: fftypes.JSONObject{
			"foo": "bar",
		},
		Symbol: "symbol",
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/createpool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)

			res := &http.Response{
				Body: ioutil.NopCloser(bytes.NewReader([]byte("bad"))),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	complete, err := h.CreateTokenPool(context.Background(), opID, pool)
	assert.False(t, complete)
	assert.Regexp(t, "FF10151", err)
}

func TestActivateTokenPool(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	pool := &fftypes.TokenPool{
		ProtocolID: "N1",
	}
	txInfo := map[string]interface{}{
		"foo": "bar",
	}
	ev := &fftypes.BlockchainEvent{
		Info: txInfo,
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/activatepool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"requestId":   opID.String(),
				"poolId":      "N1",
				"transaction": txInfo,
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

	complete, err := h.ActivateTokenPool(context.Background(), opID, pool, ev)
	assert.False(t, complete)
	assert.NoError(t, err)
}

func TestActivateTokenPoolError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	pool := &fftypes.TokenPool{
		ID: fftypes.NewUUID(),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenPool,
		},
	}
	ev := &fftypes.BlockchainEvent{}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/activatepool", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	complete, err := h.ActivateTokenPool(context.Background(), fftypes.NewUUID(), pool, ev)
	assert.False(t, complete)
	assert.Regexp(t, "FF10274", err)
}

func TestActivateTokenPoolSynchronous(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	pool := &fftypes.TokenPool{
		ProtocolID: "N1",
	}
	txInfo := map[string]interface{}{
		"foo": "bar",
	}
	ev := &fftypes.BlockchainEvent{
		Info: txInfo,
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/activatepool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"requestId":   opID.String(),
				"poolId":      "N1",
				"transaction": txInfo,
			}, body)

			res := &http.Response{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(fftypes.JSONObject{
					"type":   "fungible",
					"poolId": "F1",
					"signer": "0x0",
				}.String()))),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	mcb := h.callbacks.(*tokenmocks.Callbacks)
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.ProtocolID == "F1" && p.Type == fftypes.TokenTypeFungible && p.TransactionID == nil && p.Event.ProtocolID == ""
	})).Return(nil)

	complete, err := h.ActivateTokenPool(context.Background(), opID, pool, ev)
	assert.True(t, complete)
	assert.NoError(t, err)
}

func TestActivateTokenPoolSynchronousBadResponse(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	pool := &fftypes.TokenPool{
		ProtocolID: "N1",
	}
	txInfo := map[string]interface{}{
		"foo": "bar",
	}
	ev := &fftypes.BlockchainEvent{
		Info: txInfo,
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/activatepool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"requestId":   opID.String(),
				"poolId":      "N1",
				"transaction": txInfo,
			}, body)

			res := &http.Response{
				Body: ioutil.NopCloser(bytes.NewReader([]byte("bad"))),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	mcb := h.callbacks.(*tokenmocks.Callbacks)
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.ProtocolID == "F1" && p.Type == fftypes.TokenTypeFungible && p.TransactionID == nil && p.Event.ProtocolID == ""
	})).Return(nil)

	complete, err := h.ActivateTokenPool(context.Background(), opID, pool, ev)
	assert.False(t, complete)
	assert.Regexp(t, "FF10151", err)
}

func TestMintTokens(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	mint := &fftypes.TokenTransfer{
		LocalID: fftypes.NewUUID(),
		To:      "user1",
		Key:     "0x123",
		Amount:  *fftypes.NewFFBigInt(10),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenTransfer,
		},
	}
	opID := fftypes.NewUUID()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/mint", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolId":    "123",
				"to":        "user1",
				"amount":    "10",
				"signer":    "0x123",
				"requestId": opID.String(),
				"data":      `{"tx":"` + mint.TX.ID.String() + `"}`,
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

	err := h.MintTokens(context.Background(), opID, "123", mint)
	assert.NoError(t, err)
}

func TestTokenApproval(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	approval := &fftypes.TokenApproval{
		LocalID:  fftypes.NewUUID(),
		Operator: "0x02",
		Key:      "0x123",
		Approved: true,
		Config: fftypes.JSONObject{
			"foo": "bar",
		},
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenApproval,
		},
	}
	opID := fftypes.NewUUID()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/approval", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolId":   "123",
				"operator": "0x02",
				"approved": true,
				"signer":   "0x123",
				"config": map[string]interface{}{
					"foo": "bar",
				},
				"requestId": opID.String(),
				"data":      `{"tx":"` + approval.TX.ID.String() + `"}`,
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

	err := h.TokensApproval(context.Background(), opID, "123", approval)
	assert.NoError(t, err)
}

func TestTokenApprovalError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	approval := &fftypes.TokenApproval{}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/approval", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.TokensApproval(context.Background(), fftypes.NewUUID(), "F1", approval)
	assert.Regexp(t, "FF10274", err)
}

func TestMintTokensError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	mint := &fftypes.TokenTransfer{}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/mint", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.MintTokens(context.Background(), fftypes.NewUUID(), "F1", mint)
	assert.Regexp(t, "FF10274", err)
}

func TestBurnTokens(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	burn := &fftypes.TokenTransfer{
		LocalID:    fftypes.NewUUID(),
		TokenIndex: "1",
		From:       "user1",
		Key:        "0x123",
		Amount:     *fftypes.NewFFBigInt(10),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenTransfer,
		},
	}
	opID := fftypes.NewUUID()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/burn", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolId":     "123",
				"tokenIndex": "1",
				"from":       "user1",
				"amount":     "10",
				"signer":     "0x123",
				"requestId":  opID.String(),
				"data":       `{"tx":"` + burn.TX.ID.String() + `"}`,
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

	err := h.BurnTokens(context.Background(), opID, "123", burn)
	assert.NoError(t, err)
}

func TestBurnTokensError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	burn := &fftypes.TokenTransfer{}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/burn", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.BurnTokens(context.Background(), fftypes.NewUUID(), "F1", burn)
	assert.Regexp(t, "FF10274", err)
}

func TestTransferTokens(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	transfer := &fftypes.TokenTransfer{
		LocalID:    fftypes.NewUUID(),
		TokenIndex: "1",
		From:       "user1",
		To:         "user2",
		Key:        "0x123",
		Amount:     *fftypes.NewFFBigInt(10),
		TX: fftypes.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: fftypes.TransactionTypeTokenTransfer,
		},
	}
	opID := fftypes.NewUUID()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfer", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolId":     "123",
				"tokenIndex": "1",
				"from":       "user1",
				"to":         "user2",
				"amount":     "10",
				"signer":     "0x123",
				"requestId":  opID.String(),
				"data":       `{"tx":"` + transfer.TX.ID.String() + `"}`,
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

	err := h.TransferTokens(context.Background(), opID, "123", transfer)
	assert.NoError(t, err)
}

func TestTransferTokensError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	transfer := &fftypes.TokenTransfer{}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfer", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	err := h.TransferTokens(context.Background(), fftypes.NewUUID(), "F1", transfer)
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
	txID := fftypes.NewUUID()

	fromServer <- fftypes.JSONObject{
		"id":    "2",
		"event": "receipt",
		"data":  fftypes.JSONObject{},
	}.String()

	fromServer <- fftypes.JSONObject{
		"id":    "3",
		"event": "receipt",
		"data":  fftypes.JSONObject{"id": "abc"},
	}.String()

	// receipt: success
	mcb.On("TokenOpUpdate", h, opID, fftypes.OpStatusSucceeded, "0xffffeeee", "", mock.Anything).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "4",
		"event": "receipt",
		"data": fftypes.JSONObject{
			"id":              opID.String(),
			"success":         true,
			"transactionHash": "0xffffeeee",
		},
	}.String()

	// receipt: failure
	mcb.On("TokenOpUpdate", h, opID, fftypes.OpStatusFailed, "0xffffeeee", "", mock.Anything).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "5",
		"event": "receipt",
		"data": fftypes.JSONObject{
			"id":              opID.String(),
			"success":         false,
			"transactionHash": "0xffffeeee",
		},
	}.String()

	// token-pool: missing data
	fromServer <- fftypes.JSONObject{
		"id":    "6",
		"event": "token-pool",
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"6"},"event":"ack"}`, string(msg))

	// token-pool: invalid uuid (success)
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.ProtocolID == "F1" && p.Type == fftypes.TokenTypeFungible && p.TransactionID == nil && p.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "7",
		"event": "token-pool",
		"data": fftypes.JSONObject{
			"id":     "000000000010/000020/000030/000040",
			"type":   "fungible",
			"poolId": "F1",
			"signer": "0x0",
			"data":   fftypes.JSONObject{"tx": "bad"}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"7"},"event":"ack"}`, string(msg))

	// token-pool: success
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.ProtocolID == "F1" && p.Type == fftypes.TokenTypeFungible && txID.Equals(p.TransactionID) && p.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "8",
		"event": "token-pool",
		"data": fftypes.JSONObject{
			"id":     "000000000010/000020/000030/000040",
			"type":   "fungible",
			"poolId": "F1",
			"signer": "0x0",
			"data":   fftypes.JSONObject{"tx": txID.String()}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"8"},"event":"ack"}`, string(msg))

	// token-mint: missing data
	fromServer <- fftypes.JSONObject{
		"id":    "9",
		"event": "token-mint",
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"9"},"event":"ack"}`, string(msg))

	// token-mint: invalid amount
	fromServer <- fftypes.JSONObject{
		"id":    "10",
		"event": "token-mint",
		"data": fftypes.JSONObject{
			"id":         "1.0.0",
			"poolId":     "F1",
			"tokenIndex": "0",
			"signer":     "0x0",
			"to":         "0x0",
			"amount":     "bad",
			"data":       fftypes.JSONObject{"tx": txID.String()}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"10"},"event":"ack"}`, string(msg))

	// token-mint: success
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 2 && t.To == "0x0" && t.TokenIndex == "" && *t.TX.ID == *txID && t.PoolProtocolID == "F1" && t.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "11",
		"event": "token-mint",
		"data": fftypes.JSONObject{
			"id":     "000000000010/000020/000030/000040",
			"poolId": "F1",
			"signer": "0x0",
			"to":     "0x0",
			"amount": "2",
			"data":   fftypes.JSONObject{"tx": txID.String()}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"11"},"event":"ack"}`, string(msg))

	// token-mint: invalid uuid (success)
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 1 && t.To == "0x0" && t.TokenIndex == "1" && t.PoolProtocolID == "N1" && t.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "12",
		"event": "token-mint",
		"data": fftypes.JSONObject{
			"id":         "000000000010/000020/000030/000040",
			"poolId":     "N1",
			"tokenIndex": "1",
			"signer":     "0x0",
			"to":         "0x0",
			"amount":     "1",
			"data":       fftypes.JSONObject{"tx": "bad"}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"12"},"event":"ack"}`, string(msg))

	// token-transfer: missing from
	fromServer <- fftypes.JSONObject{
		"id":    "13",
		"event": "token-transfer",
		"data": fftypes.JSONObject{
			"id":         "000000000010/000020/000030/000040",
			"poolId":     "F1",
			"tokenIndex": "0",
			"signer":     "0x0",
			"to":         "0x0",
			"amount":     "2",
			"data":       fftypes.JSONObject{"tx": txID.String()}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"13"},"event":"ack"}`, string(msg))

	// token-transfer: bad message hash (success)
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 2 && t.From == "0x0" && t.To == "0x1" && t.TokenIndex == "" && t.PoolProtocolID == "F1" && t.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "14",
		"event": "token-transfer",
		"data": fftypes.JSONObject{
			"id":     "000000000010/000020/000030/000040",
			"poolId": "F1",
			"signer": "0x0",
			"from":   "0x0",
			"to":     "0x1",
			"amount": "2",
			"data":   fftypes.JSONObject{"tx": txID.String(), "messageHash": "bad"}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"14"},"event":"ack"}`, string(msg))

	// token-transfer: success
	messageID := fftypes.NewUUID()
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 2 && t.From == "0x0" && t.To == "0x1" && t.TokenIndex == "" && messageID.Equals(t.Message) && t.PoolProtocolID == "F1" && t.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "15",
		"event": "token-transfer",
		"data": fftypes.JSONObject{
			"id":     "000000000010/000020/000030/000040",
			"poolId": "F1",
			"signer": "0x0",
			"from":   "0x0",
			"to":     "0x1",
			"amount": "2",
			"data":   fftypes.JSONObject{"tx": txID.String(), "message": messageID.String()}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"15"},"event":"ack"}`, string(msg))

	// token-burn: success
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 2 && t.From == "0x0" && t.TokenIndex == "0" && t.PoolProtocolID == "F1" && t.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "16",
		"event": "token-burn",
		"data": fftypes.JSONObject{
			"id":         "000000000010/000020/000030/000040",
			"poolId":     "F1",
			"tokenIndex": "0",
			"signer":     "0x0",
			"from":       "0x0",
			"amount":     "2",
			"data":       fftypes.JSONObject{"tx": txID.String()}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"16"},"event":"ack"}`, string(msg))

	// token-approval: success
	mcb.On("TokensApproved", h, mock.MatchedBy(func(t *tokens.TokenApproval) bool {
		return t.Approved == true && t.Operator == "0x0" && t.PoolProtocolID == "F1" && t.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "17",
		"event": "token-approval",
		"data": fftypes.JSONObject{
			"id":       "000000000010/000020/000030/000040",
			"poolId":   "F1",
			"signer":   "0x0",
			"operator": "0x0",
			"approved": true,
			"data":     fftypes.JSONObject{"tx": txID.String()}.String(),
			"transaction": fftypes.JSONObject{
				"transactionHash": "0xffffeeee",
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"17"},"event":"ack"}`, string(msg))

	// token-approval: success (no data)
	mcb.On("TokensApproved", h, mock.MatchedBy(func(t *tokens.TokenApproval) bool {
		return t.Approved == true && t.Operator == "0x0" && t.PoolProtocolID == "F1" && t.Event.ProtocolID == "000000000010/000020/000030/000040"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "18",
		"event": "token-approval",
		"data": fftypes.JSONObject{
			"id":       "000000000010/000020/000030/000040",
			"poolId":   "F1",
			"signer":   "0x0",
			"operator": "0x0",
			"approved": true,
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"18"},"event":"ack"}`, string(msg))

	// token-approval: missing data
	fromServer <- fftypes.JSONObject{
		"id":    "9",
		"event": "token-approval",
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"9"},"event":"ack"}`, string(msg))

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
