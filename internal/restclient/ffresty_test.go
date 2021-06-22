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

package restclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

var utConfPrefix = config.NewPluginConfig("http_unit_tests")

func resetConf() {
	config.Reset()
	InitPrefix(utConfPrefix)
}

func TestRequestOK(t *testing.T) {

	customClient := &http.Client{}

	resetConf()
	utConfPrefix.Set(HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.Set(HTTPConfigHeaders, map[string]interface{}{
		"someheader": "headervalue",
	})
	utConfPrefix.Set(HTTPConfigAuthUsername, "user")
	utConfPrefix.Set(HTTPConfigAuthPassword, "pass")
	utConfPrefix.Set(HTTPConfigRetryEnabled, true)
	utConfPrefix.Set(HTTPCustomClient, customClient)

	c := New(context.Background(), utConfPrefix)
	httpmock.ActivateNonDefault(customClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/test",
		func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "headervalue", req.Header.Get("someheader"))
			assert.Equal(t, "Basic dXNlcjpwYXNz", req.Header.Get("Authorization"))
			return httpmock.NewStringResponder(200, `{"some": "data"}`)(req)
		})

	resp, err := c.R().Get("/test")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode())
	assert.Equal(t, `{"some": "data"}`, resp.String())

	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

func TestRequestRetry(t *testing.T) {

	ctx := context.Background()

	resetConf()
	utConfPrefix.Set(HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.Set(HTTPConfigRetryEnabled, true)
	utConfPrefix.Set(HTTPConfigRetryInitDelay, 1)

	c := New(ctx, utConfPrefix)
	httpmock.ActivateNonDefault(c.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/test",
		httpmock.NewStringResponder(500, `{"message": "pop"}`))

	resp, err := c.R().Get("/test")
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode())
	assert.Equal(t, 6, httpmock.GetTotalCallCount())

	err = WrapRestErr(ctx, resp, err, i18n.MsgEthconnectRESTErr)
	assert.Error(t, err)

}

func TestConfWithProxy(t *testing.T) {

	ctx := context.Background()

	resetConf()
	utConfPrefix.Set(HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.Set(HTTPConfigProxyURL, "http://myproxy.example.com:12345")
	utConfPrefix.Set(HTTPConfigRetryEnabled, false)

	c := New(ctx, utConfPrefix)
	assert.True(t, c.IsProxySet())
}

func TestLongResponse(t *testing.T) {

	ctx := context.Background()

	resetConf()
	utConfPrefix.Set(HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.Set(HTTPConfigRetryEnabled, false)

	c := New(ctx, utConfPrefix)
	httpmock.ActivateNonDefault(c.GetClient())
	defer httpmock.DeactivateAndReset()

	resText := strings.Builder{}
	for i := 0; i < 512; i++ {
		resText.WriteByte(byte('a' + (i % 26)))
	}
	httpmock.RegisterResponder("GET", "http://localhost:12345/test",
		httpmock.NewStringResponder(500, resText.String()))

	resp, err := c.R().Get("/test")
	err = WrapRestErr(ctx, resp, err, i18n.MsgEthconnectRESTErr)
	assert.Error(t, err)
}

func TestErrResponse(t *testing.T) {

	ctx := context.Background()

	resetConf()
	utConfPrefix.Set(HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.Set(HTTPConfigRetryEnabled, false)

	c := New(ctx, utConfPrefix)
	httpmock.ActivateNonDefault(c.GetClient())
	defer httpmock.DeactivateAndReset()

	resText := strings.Builder{}
	for i := 0; i < 512; i++ {
		resText.WriteByte(byte('a' + (i % 26)))
	}
	httpmock.RegisterResponder("GET", "http://localhost:12345/test",
		httpmock.NewErrorResponder(fmt.Errorf("pop")))

	resp, err := c.R().Get("/test")
	err = WrapRestErr(ctx, resp, err, i18n.MsgEthconnectRESTErr)
	assert.Error(t, err)
}

func TestOnAfterResponseNil(t *testing.T) {
	OnAfterResponse(nil, nil)
}
