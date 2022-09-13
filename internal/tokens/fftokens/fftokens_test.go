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

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/mocks/coremocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/mocks/wsmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var ffTokensConfig = config.RootSection("fftokens")

func newTestFFTokens(t *testing.T) (h *FFTokens, toServer, fromServer chan string, httpURL string, done func()) {
	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)

	toServer, fromServer, wsURL, cancel := wsclient.NewTestWSServer(nil)

	u, _ := url.Parse(wsURL)
	u.Scheme = "http"
	httpURL = u.String()

	coreconfig.Reset()
	h = &FFTokens{}
	h.InitConfig(ffTokensConfig)

	ffTokensConfig.AddKnownKey(ffresty.HTTPConfigURL, httpURL)
	ffTokensConfig.AddKnownKey(ffresty.HTTPCustomClient, mockedClient)
	config.Set("tokens", []fftypes.JSONObject{{}})

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := h.Init(ctx, cancelCtx, "testtokens", ffTokensConfig)
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
	coreconfig.Reset()
	h := &FFTokens{}
	h.InitConfig(ffTokensConfig)

	ffTokensConfig.AddKnownKey(ffresty.HTTPConfigURL, "::::////")

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := h.Init(ctx, cancelCtx, "testtokens", ffTokensConfig)
	assert.Regexp(t, "FF00149", err)
}

func TestInitMissingURL(t *testing.T) {
	coreconfig.Reset()
	h := &FFTokens{}
	h.InitConfig(ffTokensConfig)

	ctx, cancelCtx := context.WithCancel(context.Background())
	err := h.Init(ctx, cancelCtx, "testtokens", ffTokensConfig)
	assert.Regexp(t, "FF10138", err)
}

func TestCreateTokenPool(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()
	pool := &core.TokenPool{
		ID: fftypes.NewUUID(),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenPool,
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
				"requestId": "ns1:" + opID.String(),
				"signer":    "0x123",
				"type":      "fungible",
				"config": map[string]interface{}{
					"foo": "bar",
				},
				"data": fftypes.JSONObject{
					"tx":     pool.TX.ID.String(),
					"txtype": core.TransactionTypeTokenPool.String(),
				}.String(),
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

	complete, err := h.CreateTokenPool(context.Background(), nsOpID, pool)
	assert.False(t, complete)
	assert.NoError(t, err)
}

func TestCreateTokenPoolError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	pool := &core.TokenPool{
		ID: fftypes.NewUUID(),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenPool,
		},
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/createpool", httpURL),
		httpmock.NewJsonResponderOrPanic(400, fftypes.JSONObject{
			"error":   "Bad Request",
			"message": "Missing required field",
		}))

	nsOpID := "ns1:" + fftypes.NewUUID().String()
	complete, err := h.CreateTokenPool(context.Background(), nsOpID, pool)
	assert.False(t, complete)
	assert.Regexp(t, "FF10274.*Bad Request: Missing required field", err)
}

func TestCreateTokenPoolErrorMessageOnly(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	pool := &core.TokenPool{
		ID: fftypes.NewUUID(),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenPool,
		},
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/createpool", httpURL),
		httpmock.NewJsonResponderOrPanic(400, fftypes.JSONObject{
			"message": "Missing required field",
		}))

	nsOpID := "ns1:" + fftypes.NewUUID().String()
	complete, err := h.CreateTokenPool(context.Background(), nsOpID, pool)
	assert.False(t, complete)
	assert.Regexp(t, "FF10274.*Missing required field", err)
}

func TestCreateTokenPoolUnexpectedError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	pool := &core.TokenPool{
		ID: fftypes.NewUUID(),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenPool,
		},
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/createpool", httpURL),
		httpmock.NewStringResponder(400, "Failed miserably"))

	nsOpID := "ns1:" + fftypes.NewUUID().String()
	complete, err := h.CreateTokenPool(context.Background(), nsOpID, pool)
	assert.False(t, complete)
	assert.Regexp(t, "FF10274.*Failed miserably", err)
}

func TestCreateTokenPoolSynchronous(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()
	pool := &core.TokenPool{
		ID: fftypes.NewUUID(),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenPool,
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
					"type":        "fungible",
					"poolLocator": "F1",
					"signer":      "0x0",
					"decimals":    18,
					"data":        `{"tx":"` + pool.TX.ID.String() + `"}`,
				}.String()))),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	mcb := &tokenmocks.Callbacks{}
	h.SetHandler("ns1", mcb)

	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.PoolLocator == "F1" && p.Type == core.TokenTypeFungible && *p.TX.ID == *pool.TX.ID
	})).Return(nil)

	complete, err := h.CreateTokenPool(context.Background(), nsOpID, pool)
	assert.True(t, complete)
	assert.NoError(t, err)
}

func TestCreateTokenPoolSynchronousBadResponse(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()
	pool := &core.TokenPool{
		ID: fftypes.NewUUID(),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenPool,
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

	complete, err := h.CreateTokenPool(context.Background(), nsOpID, pool)
	assert.False(t, complete)
	assert.Regexp(t, "FF00127", err)
}

func TestActivateTokenPool(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()
	poolConfig := map[string]interface{}{
		"address": "0x12345",
	}
	pool := &core.TokenPool{
		Namespace: "ns1",
		Locator:   "N1",
		Config:    poolConfig,
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/activatepool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolData":    "ns1",
				"requestId":   "ns1:" + opID.String(),
				"poolLocator": "N1",
				"config":      poolConfig,
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

	complete, err := h.ActivateTokenPool(context.Background(), nsOpID, pool)
	assert.False(t, complete)
	assert.NoError(t, err)
}

func TestActivateTokenPoolError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	pool := &core.TokenPool{
		ID: fftypes.NewUUID(),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenPool,
		},
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/activatepool", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	nsOpID := "ns1:" + fftypes.NewUUID().String()
	complete, err := h.ActivateTokenPool(context.Background(), nsOpID, pool)
	assert.False(t, complete)
	assert.Regexp(t, "FF10274", err)
}

func TestActivateTokenPoolSynchronous(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()
	poolConfig := map[string]interface{}{
		"foo": "bar",
	}
	pool := &core.TokenPool{
		Namespace: "ns1",
		Locator:   "N1",
		Config:    poolConfig,
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/activatepool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolData":    "ns1",
				"requestId":   "ns1:" + opID.String(),
				"poolLocator": "N1",
				"config":      poolConfig,
			}, body)

			res := &http.Response{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(fftypes.JSONObject{
					"type":        "fungible",
					"poolLocator": "F1",
					"signer":      "0x0",
				}.String()))),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				StatusCode: 200,
			}
			return res, nil
		})

	mcb := &tokenmocks.Callbacks{}
	h.SetHandler("ns1", mcb)
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.PoolLocator == "F1" && p.Type == core.TokenTypeFungible && p.TX.ID == nil && p.Event == nil
	})).Return(nil)

	complete, err := h.ActivateTokenPool(context.Background(), nsOpID, pool)
	assert.True(t, complete)
	assert.NoError(t, err)
}

func TestActivateTokenPoolSynchronousBadResponse(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()
	poolConfig := map[string]interface{}{
		"foo": "bar",
	}
	pool := &core.TokenPool{
		Namespace: "ns1",
		Locator:   "N1",
		Config:    poolConfig,
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/activatepool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolData":    "ns1",
				"requestId":   "ns1:" + opID.String(),
				"poolLocator": "N1",
				"config":      poolConfig,
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

	mcb := &tokenmocks.Callbacks{}
	h.SetHandler("ns1", mcb)
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.PoolLocator == "F1" && p.Type == core.TokenTypeFungible && p.TX.ID == nil
	})).Return(nil)

	complete, err := h.ActivateTokenPool(context.Background(), nsOpID, pool)
	assert.False(t, complete)
	assert.Regexp(t, "FF00127", err)
}

func TestActivateTokenPoolNoContent(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()
	poolConfig := map[string]interface{}{
		"foo": "bar",
	}
	pool := &core.TokenPool{
		Namespace: "ns1",
		Locator:   "N1",
		Config:    poolConfig,
	}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/activatepool", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolData":    "ns1",
				"requestId":   "ns1:" + opID.String(),
				"poolLocator": "N1",
				"config":      poolConfig,
			}, body)

			res := &http.Response{
				StatusCode: 204,
			}
			return res, nil
		})

	complete, err := h.ActivateTokenPool(context.Background(), nsOpID, pool)
	assert.True(t, complete)
	assert.NoError(t, err)
}

func TestMintTokens(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	mint := &core.TokenTransfer{
		LocalID: fftypes.NewUUID(),
		To:      "user1",
		Key:     "0x123",
		Amount:  *fftypes.NewFFBigInt(10),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenTransfer,
		},
		URI: "FLAPFLIP",
		Config: fftypes.JSONObject{
			"foo": "bar",
		},
	}
	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/mint", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolLocator": "123",
				"to":          "user1",
				"amount":      "10",
				"signer":      "0x123",
				"config": map[string]interface{}{
					"foo": "bar",
				},
				"requestId": "ns1:" + opID.String(),
				"data": fftypes.JSONObject{
					"tx":     mint.TX.ID.String(),
					"txtype": core.TransactionTypeTokenTransfer.String(),
				}.String(),
				"uri": "FLAPFLIP",
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

	err := h.MintTokens(context.Background(), nsOpID, "123", mint)
	assert.NoError(t, err)
}

func TestTokenApproval(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	approval := &core.TokenApproval{
		LocalID:  fftypes.NewUUID(),
		Operator: "0x02",
		Key:      "0x123",
		Approved: true,
		Config: fftypes.JSONObject{
			"foo": "bar",
		},
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenApproval,
		},
	}
	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/approval", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolLocator": "123",
				"operator":    "0x02",
				"approved":    true,
				"signer":      "0x123",
				"config": map[string]interface{}{
					"foo": "bar",
				},
				"requestId": "ns1:" + opID.String(),
				"data": fftypes.JSONObject{
					"tx":     approval.TX.ID.String(),
					"txtype": core.TransactionTypeTokenApproval.String(),
				}.String(),
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

	err := h.TokensApproval(context.Background(), nsOpID, "123", approval)
	assert.NoError(t, err)
}

func TestTokenApprovalError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	approval := &core.TokenApproval{}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/approval", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	nsOpID := "ns1:" + fftypes.NewUUID().String()
	err := h.TokensApproval(context.Background(), nsOpID, "F1", approval)
	assert.Regexp(t, "FF10274", err)
}

func TestMintTokensError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	mint := &core.TokenTransfer{}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/mint", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	nsOpID := "ns1:" + fftypes.NewUUID().String()
	err := h.MintTokens(context.Background(), nsOpID, "F1", mint)
	assert.Regexp(t, "FF10274", err)
}

func TestBurnTokens(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	burn := &core.TokenTransfer{
		LocalID:    fftypes.NewUUID(),
		TokenIndex: "1",
		From:       "user1",
		Key:        "0x123",
		Amount:     *fftypes.NewFFBigInt(10),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenTransfer,
		},
		Config: fftypes.JSONObject{
			"foo": "bar",
		},
	}
	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/burn", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolLocator": "123",
				"tokenIndex":  "1",
				"from":        "user1",
				"amount":      "10",
				"signer":      "0x123",
				"config": map[string]interface{}{
					"foo": "bar",
				},
				"requestId": "ns1:" + opID.String(),
				"data": fftypes.JSONObject{
					"tx":     burn.TX.ID.String(),
					"txtype": core.TransactionTypeTokenTransfer.String(),
				}.String(),
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

	err := h.BurnTokens(context.Background(), nsOpID, "123", burn)
	assert.NoError(t, err)
}

func TestBurnTokensError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	burn := &core.TokenTransfer{}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/burn", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	nsOpID := "ns1:" + fftypes.NewUUID().String()
	err := h.BurnTokens(context.Background(), nsOpID, "F1", burn)
	assert.Regexp(t, "FF10274", err)
}

func TestTransferTokens(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	transfer := &core.TokenTransfer{
		LocalID:    fftypes.NewUUID(),
		TokenIndex: "1",
		From:       "user1",
		To:         "user2",
		Key:        "0x123",
		Amount:     *fftypes.NewFFBigInt(10),
		TX: core.TransactionRef{
			ID:   fftypes.NewUUID(),
			Type: core.TransactionTypeTokenTransfer,
		},
		Config: fftypes.JSONObject{
			"foo": "bar",
		},
	}
	opID := fftypes.NewUUID()
	nsOpID := "ns1:" + opID.String()

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfer", httpURL),
		func(req *http.Request) (*http.Response, error) {
			body := make(fftypes.JSONObject)
			err := json.NewDecoder(req.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, fftypes.JSONObject{
				"poolLocator": "123",
				"tokenIndex":  "1",
				"from":        "user1",
				"to":          "user2",
				"amount":      "10",
				"signer":      "0x123",
				"config": map[string]interface{}{
					"foo": "bar",
				},
				"requestId": "ns1:" + opID.String(),
				"data": fftypes.JSONObject{
					"tx":     transfer.TX.ID.String(),
					"txtype": core.TransactionTypeTokenTransfer.String(),
				}.String(),
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

	err := h.TransferTokens(context.Background(), nsOpID, "123", transfer)
	assert.NoError(t, err)
}

func TestTransferTokensError(t *testing.T) {
	h, _, _, httpURL, done := newTestFFTokens(t)
	defer done()

	transfer := &core.TokenTransfer{}

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/api/v1/transfer", httpURL),
		httpmock.NewJsonResponderOrPanic(500, fftypes.JSONObject{}))

	nsOpID := "ns1:" + fftypes.NewUUID().String()
	err := h.TransferTokens(context.Background(), nsOpID, "F1", transfer)
	assert.Regexp(t, "FF10274", err)
}

func TestIgnoredEvents(t *testing.T) {
	h, toServer, fromServer, _, done := newTestFFTokens(t)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	fromServer <- `!}`         // ignored
	fromServer <- `{}`         // ignored
	fromServer <- `{"id":"1"}` // ignored but acked
	msg := <-toServer
	assert.Equal(t, `{"data":{"id":"1"},"event":"ack"}`, string(msg))

	fromServer <- fftypes.JSONObject{
		"id":    "2",
		"event": "receipt",
		"data":  fftypes.JSONObject{},
	}.String()
}

func TestReceiptEvents(t *testing.T) {
	h, _, fromServer, _, done := newTestFFTokens(t)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	mcb := &coremocks.OperationCallbacks{}
	h.SetOperationHandler("ns1", mcb)
	opID := fftypes.NewUUID()
	mockCalled := make(chan bool)

	// receipt: bad ID - passed through
	mcb.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == "ns1:wrong" &&
			update.Status == core.OpStatusPending &&
			update.Plugin == "fftokens"
	})).Return(nil).Once().Run(func(args mock.Arguments) { mockCalled <- true })
	fromServer <- fftypes.JSONObject{
		"id":    "3",
		"event": "receipt",
		"data": fftypes.JSONObject{
			"headers": fftypes.JSONObject{
				"requestId": "ns1:wrong", // passed through to OperationUpdate to ignore
				"type":      "TransactionUpdate",
			},
		},
	}.String()
	<-mockCalled

	// receipt: success
	mcb.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == "ns1:"+opID.String() &&
			update.Status == core.OpStatusSucceeded &&
			update.BlockchainTXID == "0xffffeeee" &&
			update.Plugin == "fftokens"
	})).Return(nil).Once().Run(func(args mock.Arguments) { mockCalled <- true })
	fromServer <- fftypes.JSONObject{
		"id":    "4",
		"event": "receipt",
		"data": fftypes.JSONObject{
			"headers": fftypes.JSONObject{
				"requestId": "ns1:" + opID.String(),
				"type":      "TransactionSuccess",
			},
			"transactionHash": "0xffffeeee",
		},
	}.String()
	<-mockCalled

	// receipt: update
	mcb.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == "ns1:"+opID.String() &&
			update.Status == core.OpStatusPending &&
			update.BlockchainTXID == "0xffffeeee"
	})).Return(nil).Once().Run(func(args mock.Arguments) { mockCalled <- true })
	fromServer <- fftypes.JSONObject{
		"id":    "5",
		"event": "receipt",
		"data": fftypes.JSONObject{
			"headers": fftypes.JSONObject{
				"requestId": "ns1:" + opID.String(),
				"type":      "TransactionUpdate",
			},
			"transactionHash": "0xffffeeee",
		},
	}.String()
	<-mockCalled

	// receipt: failure
	mcb.On("OperationUpdate", mock.MatchedBy(func(update *core.OperationUpdate) bool {
		return update.NamespacedOpID == "ns1:"+opID.String() &&
			update.Status == core.OpStatusFailed &&
			update.BlockchainTXID == "0xffffeeee" &&
			update.Plugin == "fftokens"
	})).Return(nil).Once().Run(func(args mock.Arguments) { mockCalled <- true })
	fromServer <- fftypes.JSONObject{
		"id":    "5",
		"event": "receipt",
		"data": fftypes.JSONObject{
			"headers": fftypes.JSONObject{
				"requestId": "ns1:" + opID.String(),
				"type":      "TransactionFailed",
			},
			"transactionHash": "0xffffeeee",
		},
	}.String()
	<-mockCalled

	mcb.AssertExpectations(t)
}

func TestPoolEvents(t *testing.T) {
	h, toServer, fromServer, _, done := newTestFFTokens(t)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	mcb := &tokenmocks.Callbacks{}
	h.SetHandler("ns1", mcb)
	txID := fftypes.NewUUID()

	// token-pool: missing data
	fromServer <- fftypes.JSONObject{
		"id":    "6",
		"event": "token-pool",
	}.String()
	msg := <-toServer
	assert.Equal(t, `{"data":{"id":"6"},"event":"ack"}`, string(msg))

	// token-pool: invalid uuid (success)
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.PoolLocator == "F1" && p.Type == core.TokenTypeFungible && p.TX.ID == nil && p.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "7",
		"event": "token-pool",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"type":        "fungible",
			"poolLocator": "F1",
			"signer":      "0x0",
			"data":        fftypes.JSONObject{"tx": "bad"}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"7"},"event":"ack"}`, string(msg))

	// token-pool: success
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.PoolLocator == "F1" && p.Type == core.TokenTypeFungible && txID.Equals(p.TX.ID) && p.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "8",
		"event": "token-pool",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolData":    "ns1",
			"type":        "fungible",
			"poolLocator": "F1",
			"signer":      "0x0",
			"data":        fftypes.JSONObject{"tx": txID.String()}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"8"},"event":"ack"}`, string(msg))

	// token-pool: batch + callback fail
	mcb.On("TokenPoolCreated", h, mock.MatchedBy(func(p *tokens.TokenPool) bool {
		return p.PoolLocator == "F1" && p.Type == core.TokenTypeFungible && txID.Equals(p.TX.ID) && p.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(fmt.Errorf("pop")).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "9",
		"event": "batch",
		"data": fftypes.JSONObject{
			"events": fftypes.JSONObjectArray{{
				"event": "token-pool",
				"data": fftypes.JSONObject{
					"id":          "000000000010/000020/000030/000040",
					"type":        "fungible",
					"poolLocator": "F1",
					"signer":      "0x0",
					"data":        fftypes.JSONObject{"tx": txID.String()}.String(),
					"blockchain": fftypes.JSONObject{
						"id": "000000000010/000020/000030",
						"info": fftypes.JSONObject{
							"transactionHash": "0xffffeeee",
						},
					},
				},
			}},
		},
	}.String()
}

func TestTransferEvents(t *testing.T) {
	h, toServer, fromServer, _, done := newTestFFTokens(t)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	mcb := &tokenmocks.Callbacks{}
	h.SetHandler("ns1", mcb)
	txID := fftypes.NewUUID()

	// token-mint: missing data
	fromServer <- fftypes.JSONObject{
		"id":    "9",
		"event": "token-mint",
	}.String()
	msg := <-toServer
	assert.Equal(t, `{"data":{"id":"9"},"event":"ack"}`, string(msg))

	// token-mint: invalid amount
	fromServer <- fftypes.JSONObject{
		"id":    "10",
		"event": "token-mint",
		"data": fftypes.JSONObject{
			"id":          "1.0.0",
			"poolLocator": "F1",
			"tokenIndex":  "0",
			"signer":      "0x0",
			"to":          "0x0",
			"amount":      "bad",
			"data":        fftypes.JSONObject{"tx": txID.String()}.String(),
			"blockchain": fftypes.JSONObject{
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"10"},"event":"ack"}`, string(msg))

	// token-mint: success
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 2 && t.To == "0x0" && t.TokenIndex == "" && *t.TX.ID == *txID && t.PoolLocator == "F1" && t.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "11",
		"event": "token-mint",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolData":    "ns1",
			"poolLocator": "F1",
			"signer":      "0x0",
			"to":          "0x0",
			"amount":      "2",
			"data":        fftypes.JSONObject{"tx": txID.String()}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"11"},"event":"ack"}`, string(msg))

	// token-mint: invalid uuid (success)
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 1 && t.To == "0x0" && t.TokenIndex == "1" && t.PoolLocator == "N1" && t.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "12",
		"event": "token-mint",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolLocator": "N1",
			"tokenIndex":  "1",
			"signer":      "0x0",
			"to":          "0x0",
			"amount":      "1",
			"data":        fftypes.JSONObject{"tx": "bad"}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
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
			"id":          "000000000010/000020/000030/000040",
			"poolLocator": "F1",
			"tokenIndex":  "0",
			"signer":      "0x0",
			"to":          "0x0",
			"amount":      "2",
			"data":        fftypes.JSONObject{"tx": txID.String()}.String(),
			"blockchain": fftypes.JSONObject{
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"13"},"event":"ack"}`, string(msg))

	// token-transfer: bad message hash (success)
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 2 && t.From == "0x0" && t.To == "0x1" && t.TokenIndex == "" && t.PoolLocator == "F1" && t.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "14",
		"event": "token-transfer",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolLocator": "F1",
			"signer":      "0x0",
			"from":        "0x0",
			"to":          "0x1",
			"amount":      "2",
			"data":        fftypes.JSONObject{"tx": txID.String(), "messageHash": "bad"}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"14"},"event":"ack"}`, string(msg))

	// token-transfer: success
	messageID := fftypes.NewUUID()
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 2 && t.From == "0x0" && t.To == "0x1" && t.TokenIndex == "" && messageID.Equals(t.Message) && t.PoolLocator == "F1" && t.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "15",
		"event": "token-transfer",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolLocator": "F1",
			"signer":      "0x0",
			"from":        "0x0",
			"to":          "0x1",
			"amount":      "2",
			"data":        fftypes.JSONObject{"tx": txID.String(), "message": messageID.String()}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"15"},"event":"ack"}`, string(msg))

	// token-burn: success
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 2 && t.From == "0x0" && t.TokenIndex == "0" && t.PoolLocator == "F1" && t.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "16",
		"event": "token-burn",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolLocator": "F1",
			"tokenIndex":  "0",
			"signer":      "0x0",
			"from":        "0x0",
			"amount":      "2",
			"data":        fftypes.JSONObject{"tx": txID.String()}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"16"},"event":"ack"}`, string(msg))

	// token-transfer: callback fail
	mcb.On("TokensTransferred", h, mock.MatchedBy(func(t *tokens.TokenTransfer) bool {
		return t.Amount.Int().Int64() == 2 && t.From == "0x0" && t.To == "0x1" && t.TokenIndex == "" && messageID.Equals(t.Message) && t.PoolLocator == "F1" && t.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(fmt.Errorf("pop")).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "17",
		"event": "token-transfer",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolLocator": "F1",
			"signer":      "0x0",
			"from":        "0x0",
			"to":          "0x1",
			"amount":      "2",
			"data":        fftypes.JSONObject{"tx": txID.String(), "message": messageID.String()}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
}

func TestApprovalEvents(t *testing.T) {
	h, toServer, fromServer, _, done := newTestFFTokens(t)
	defer done()

	err := h.Start()
	assert.NoError(t, err)

	mcb := &tokenmocks.Callbacks{}
	h.SetHandler("ns1", mcb)
	txID := fftypes.NewUUID()

	// token-approval: success
	mcb.On("TokensApproved", h, mock.MatchedBy(func(t *tokens.TokenApproval) bool {
		return t.Approved == true &&
			t.Operator == "0x0" &&
			t.PoolLocator == "F1" &&
			t.Event != nil &&
			t.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "17",
		"event": "token-approval",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolData":    "ns1",
			"subject":     "a:b",
			"poolLocator": "F1",
			"signer":      "0x0",
			"operator":    "0x0",
			"approved":    true,
			"data":        fftypes.JSONObject{"tx": txID.String()}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg := <-toServer
	assert.Equal(t, `{"data":{"id":"17"},"event":"ack"}`, string(msg))

	// token-approval: success (no data)
	mcb.On("TokensApproved", h, mock.MatchedBy(func(t *tokens.TokenApproval) bool {
		return t.Approved == true && t.Operator == "0x0" && t.PoolLocator == "F1"
	})).Return(nil).Once()
	fromServer <- fftypes.JSONObject{
		"id":    "18",
		"event": "token-approval",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolData":    "ns1",
			"subject":     "a:b",
			"poolLocator": "F1",
			"signer":      "0x0",
			"operator":    "0x0",
			"approved":    true,
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"18"},"event":"ack"}`, string(msg))

	// token-approval: missing data
	fromServer <- fftypes.JSONObject{
		"id":    "19",
		"event": "token-approval",
	}.String()
	msg = <-toServer
	assert.Equal(t, `{"data":{"id":"19"},"event":"ack"}`, string(msg))

	// token-approval: callback fail
	errProcessed := make(chan struct{})
	mcb.On("TokensApproved", h, mock.MatchedBy(func(t *tokens.TokenApproval) bool {
		return t.Approved == true && t.Operator == "0x0" && t.PoolLocator == "F1" && t.Event.ProtocolID == "000000000010/000020/000030"
	})).Return(fmt.Errorf("pop")).Once().Run(func(args mock.Arguments) {
		// We do not ack in the case of an error
		close(errProcessed)
	})
	fromServer <- fftypes.JSONObject{
		"id":    "20",
		"event": "token-approval",
		"data": fftypes.JSONObject{
			"id":          "000000000010/000020/000030/000040",
			"poolData":    "", // deliberately to drive to all namespaces
			"subject":     "a:b",
			"poolLocator": "F1",
			"signer":      "0x0",
			"operator":    "0x0",
			"approved":    true,
			"data":        fftypes.JSONObject{"tx": txID.String()}.String(),
			"blockchain": fftypes.JSONObject{
				"id": "000000000010/000020/000030",
				"info": fftypes.JSONObject{
					"transactionHash": "0xffffeeee",
				},
			},
		},
	}.String()
	<-errProcessed

	mcb.AssertExpectations(t)
}

func TestEventLoopReceiveClosed(t *testing.T) {
	wsm := &wsmocks.WSClient{}
	called := false
	h := &FFTokens{
		ctx:       context.Background(),
		cancelCtx: func() { called = true },
		wsconn:    wsm,
	}
	r := make(chan []byte)
	close(r)
	wsm.On("Close").Return()
	wsm.On("Receive").Return((<-chan []byte)(r))
	h.eventLoop()
	assert.True(t, called)
}

func TestEventLoopSendClosed(t *testing.T) {
	wsm := &wsmocks.WSClient{}
	called := false
	h := &FFTokens{
		ctx:       context.Background(),
		cancelCtx: func() { called = true },
		wsconn:    wsm,
	}
	r := make(chan []byte, 1)
	r <- []byte(`{"id":"1"}`) // ignored but acked
	wsm.On("Close").Return()
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Send", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	h.eventLoop()
	assert.True(t, called)
}

func TestEventLoopClosedContext(t *testing.T) {
	wsm := &wsmocks.WSClient{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := &FFTokens{
		ctx:    ctx,
		wsconn: wsm,
	}
	r := make(chan []byte, 1)
	wsm.On("Close").Return()
	wsm.On("Receive").Return((<-chan []byte)(r))
	h.eventLoop() // we're simply looking for it exiting
}

func TestCallbacksWrongNamespace(t *testing.T) {
	h, _, _, _, done := newTestFFTokens(t)
	defer done()
	nsOpID := "ns1:" + fftypes.NewUUID().String()
	h.callbacks.OperationUpdate(context.Background(), nsOpID, core.OpStatusSucceeded, "tx123", "", nil)
	h.callbacks.TokensTransferred(context.Background(), "ns1", nil)
	h.callbacks.TokensApproved(context.Background(), "ns1", nil)
}
