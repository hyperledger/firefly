// Copyright © 2022 Kaleido, Inc.
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

package fabric

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/mocks/blockchainmocks"
	"github.com/hyperledger/firefly/mocks/wsmocks"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/wsclient"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var utConfPrefix = config.NewPluginConfig("fab_unit_tests")
var utFabconnectConf = utConfPrefix.SubPrefix(FabconnectConfigKey)
var signer = "orgMSP::x509::CN=signer001,OU=client::CN=fabric-ca"

func resetConf() {
	config.Reset()
	e := &Fabric{}
	e.InitPrefix(utConfPrefix)
}

func newTestFabric() (*Fabric, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:            ctx,
		client:         resty.New().SetBaseURL("http://localhost:12345"),
		defaultChannel: "firefly",
		chaincode:      "firefly",
		topic:          "topic1",
		prefixShort:    defaultPrefixShort,
		prefixLong:     defaultPrefixLong,
		callbacks:      em,
		wsconn:         wsm,
	}
	return e, func() {
		cancel()
		if e.closed != nil {
			// We've init'd, wait to close
			<-e.closed
		}
	}
}

func testFFIMethod() *fftypes.FFIMethod {
	return &fftypes.FFIMethod{
		Name: "sum",
		Params: []*fftypes.FFIParam{
			{
				Name:    "x",
				Type:    "integer",
				Details: fftypes.JSONAnyPtr(`{"type": "int"}`),
			},
			{
				Name:    "y",
				Type:    "integer",
				Details: fftypes.JSONAnyPtr(`{"type": "int"}`),
			},
		},
		Returns: []*fftypes.FFIParam{
			{
				Name:    "z",
				Type:    "integer",
				Details: fftypes.JSONAnyPtr(`{"type": "int"}`),
			},
		},
	}
}

func TestInitMissingURL(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf()
	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10138.*url", err)
}

func TestInitMissingChaincode(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10138.*chaincode", err)
}

func TestInitMissingTopic(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(FabconnectConfigChaincode, "Firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10138.*topic", err)
}

func TestInitAllNewStreamsAndWSEvent(t *testing.T) {

	log.SetLevel("trace")
	e, cancel := newTestFabric()
	defer cancel()

	toServer, fromServer, _, wsURL, done := wsclient.NewTestWSServer(nil)
	defer done()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	u, _ := url.Parse(wsURL)
	u.Scheme = "http"
	httpURL := u.String()

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/eventstreams", httpURL),
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/eventstreams", httpURL),
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/subscriptions", httpURL),
		httpmock.NewJsonResponderOrPanic(200, []subscription{}))
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/subscriptions", httpURL),
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, "es12345", body["stream"])
			assert.Equal(t, "0", body["fromBlock"])
			return httpmock.NewJsonResponderOrPanic(200, subscription{ID: "sub12345"})(req)
		})

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, httpURL)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.NoError(t, err)

	assert.Equal(t, "fabric", e.Name())
	assert.Equal(t, 4, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.initInfo.stream.ID)
	assert.Equal(t, "sub12345", e.initInfo.sub.ID)
	assert.True(t, e.Capabilities().GlobalSequencer)

	err = e.Start()
	assert.NoError(t, err)

	startupMessage := <-toServer
	assert.Equal(t, `{"type":"listen","topic":"topic1"}`, startupMessage)
	startupMessage = <-toServer
	assert.Equal(t, `{"type":"listenreplies"}`, startupMessage)
	fromServer <- `[]` // empty batch, will be ignored, but acked
	reply := <-toServer
	assert.Equal(t, `{"topic":"topic1","type":"ack"}`, reply)

	// Bad data will be ignored
	fromServer <- `!json`
	fromServer <- `{"not": "a reply"}`
	fromServer <- `42`

}

func TestWSInitFail(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "!!!://")
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})
	assert.Regexp(t, "FF10162", err)

}

func TestWSConnectFail(t *testing.T) {

	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:    context.Background(),
		wsconn: wsm,
	}
	wsm.On("Connect").Return(fmt.Errorf("pop"))

	err := e.Start()
	assert.EqualError(t, err, "pop")
}

func TestInitAllExistingStreams(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{{ID: "es12345", WebSocket: eventStreamWebsocket{Topic: "topic1"}}}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewJsonResponderOrPanic(200, []subscription{
			{ID: "sub12345", Name: "BatchPin"},
		}))

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Equal(t, 2, httpmock.GetTotalCallCount())
	assert.Equal(t, "es12345", e.initInfo.stream.ID)
	assert.Equal(t, "sub12345", e.initInfo.sub.ID)

	assert.NoError(t, err)

}

func TestStreamQueryError(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewStringResponder(500, `pop`))

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10284", err)
	assert.Regexp(t, "pop", err)

}

func TestStreamCreateError(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewStringResponder(500, `pop`))

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10284", err)
	assert.Regexp(t, "pop", err)

}

func TestSubQueryError(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, []eventStream{}))
	httpmock.RegisterResponder("POST", "http://localhost:12345/eventstreams",
		httpmock.NewJsonResponderOrPanic(200, eventStream{ID: "es12345"}))
	httpmock.RegisterResponder("GET", "http://localhost:12345/subscriptions",
		httpmock.NewStringResponder(500, `pop`))

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10284", err)
	assert.Regexp(t, "pop", err)

}

func TestSubQueryCreateError(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()

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

	resetConf()
	utFabconnectConf.Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utFabconnectConf.Set(restclient.HTTPConfigRetryEnabled, false)
	utFabconnectConf.Set(restclient.HTTPCustomClient, mockedClient)
	utFabconnectConf.Set(FabconnectConfigChaincode, "firefly")
	utFabconnectConf.Set(FabconnectConfigSigner, "signer001")
	utFabconnectConf.Set(FabconnectConfigTopic, "topic1")

	err := e.Init(e.ctx, utConfPrefix, &blockchainmocks.Callbacks{})

	assert.Regexp(t, "FF10284", err)
	assert.Regexp(t, "pop", err)

}

func TestSubmitBatchPinOK(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
	batch := &blockchain.BatchPin{
		TransactionID:   fftypes.MustParseUUID("9ffc50ff-6bfe-4502-adc7-93aea54cc059"),
		BatchID:         fftypes.MustParseUUID("c5df767c-fe44-4e03-8eb5-1c5523097db5"),
		BatchHash:       fftypes.NewRandB32(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, req.FormValue(defaultPrefixShort+"-signer"))
			assert.Equal(t, "false", req.FormValue(defaultPrefixShort+"-sync"))
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", (body["args"].([]interface{}))[1])
			assert.Equal(t, hexFormatB32(batch.BatchHash), (body["args"].([]interface{}))[2])
			assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", (body["args"].([]interface{}))[3])
			return httpmock.NewJsonResponderOrPanic(200, asyncTXSubmission{})(req)
		})

	err := e.SubmitBatchPin(context.Background(), nil, nil, signer, batch)

	assert.NoError(t, err)

}

func TestSubmitBatchEmptyPayloadRef(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
	batch := &blockchain.BatchPin{
		TransactionID: fftypes.MustParseUUID("9ffc50ff-6bfe-4502-adc7-93aea54cc059"),
		BatchID:       fftypes.MustParseUUID("c5df767c-fe44-4e03-8eb5-1c5523097db5"),
		BatchHash:     fftypes.NewRandB32(),
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signer, req.FormValue("fly-signer"))
			assert.Equal(t, "false", req.FormValue("fly-sync"))
			assert.Equal(t, "0x9ffc50ff6bfe4502adc793aea54cc059c5df767cfe444e038eb51c5523097db5", (body["args"].([]interface{}))[1])
			assert.Equal(t, hexFormatB32(batch.BatchHash), (body["args"].([]interface{}))[2])
			assert.Equal(t, "", (body["args"].([]interface{}))[3])
			return httpmock.NewJsonResponderOrPanic(200, asyncTXSubmission{})(req)
		})

	err := e.SubmitBatchPin(context.Background(), nil, nil, signer, batch)

	assert.NoError(t, err)

}

func TestSubmitBatchPinFail(t *testing.T) {

	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	signer := "signer001"
	batch := &blockchain.BatchPin{
		TransactionID:   fftypes.NewUUID(),
		BatchID:         fftypes.NewUUID(),
		BatchHash:       fftypes.NewRandB32(),
		BatchPayloadRef: "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		Contexts: []*fftypes.Bytes32{
			fftypes.NewRandB32(),
			fftypes.NewRandB32(),
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		httpmock.NewStringResponder(500, "pop"))

	err := e.SubmitBatchPin(context.Background(), nil, nil, signer, batch)

	assert.Regexp(t, "FF10284", err)
	assert.Regexp(t, "pop", err)

}

func TestResolveFullIDSigner(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()

	id := "org1MSP::x509::CN=admin,OU=client::CN=fabric-ca-server"
	signKey, err := e.ResolveSigningKey(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "org1MSP::x509::CN=admin,OU=client::CN=fabric-ca-server", signKey)

}

func TestResolveSigner(t *testing.T) {
	e, cancel := newTestFabric()
	e.idCache = make(map[string]*fabIdentity)
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	res := make(map[string]string)
	res["name"] = "signer001"
	res["mspId"] = "org1MSP"
	res["enrollmentCert"] = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJ4ekNDQVcyZ0F3SUJBZ0lVTUpXdUJBcHl0eXdNVU13cC82b3o3Qk0wS3dJd0NnWUlLb1pJemowRUF3SXcKR3pFWk1CY0dBMVVFQXhNUVptRmljbWxqTFdOaExYTmxjblpsY2pBZUZ3MHlNVEEzTWpreE5UUXdNREJhRncweQpNakEzTWpreE5UUTFNREJhTUNFeER6QU5CZ05WQkFzVEJtTnNhV1Z1ZERFT01Bd0dBMVVFQXhNRllXUnRhVzR3CldUQVRCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFUTUxMR2VwR2oyWEo3aWFhU1hXWXBpSGtCc3RqbXUKcStzd3hIOTdxWi9vS0JWMHFoa21kcUlkTmNNaTdwNHNYQzM1NTN6Nm5DUHpqSWtjQzdqWi9IVDBvNEdJTUlHRgpNQTRHQTFVZER3RUIvd1FFQXdJQkJqQU1CZ05WSFJNQkFmOEVBakFBTUIwR0ExVWREZ1FXQkJRZUdkWDNVdUxMCnZWVHpDVkdwcVVJQjFFdEhMREFmQmdOVkhTTUVHREFXZ0JUcTdoVzQ5Yno0WjAyK2YyM3hVSGxCbzd5eGFqQWwKQmdOVkhSRUVIakFjZ2hwTFlXeGxhV1J2Y3kxTllXTkNiMjlyTFZCeWJ5NXNiMk5oYkRBS0JnZ3Foa2pPUFFRRApBZ05JQURCRkFpRUF1bzVtbGh6UXc4RnIrcUFhUzAxcCsxTlVaNEF5ZmdQb21kQ2RKTzJUYXJRQ0lIUG1pTUhuCk9jekc5cS9kT3NiQUQ1c3dZbWcyTEZpM05mQkswK0cvUC9TUAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	res["caCert"] = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJmVENDQVNPZ0F3SUJBZ0lVYndac0FnK2Zac0FmSUF2VWFlWXBpOXF3NG9jd0NnWUlLb1pJemowRUF3SXcKR3pFWk1CY0dBMVVFQXhNUVptRmljbWxqTFdOaExYTmxjblpsY2pBZUZ3MHlNVEEzTWpNd01URTRNREJhRncwegpOakEzTVRrd01URTRNREJhTUJzeEdUQVhCZ05WQkFNVEVHWmhZbkpwWXkxallTMXpaWEoyWlhJd1dUQVRCZ2NxCmhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUlZNajcyR1dTeXk1UjRQN084ckpidXkrNHd6NWJWSE94dHBxRlUKamNadVE0Q2VSUGJoNDF3KzR1dFJsTlRTbFhLdTBMblBlVEZLSjlRT00xd0xwTGJtbzBVd1F6QU9CZ05WSFE4QgpBZjhFQkFNQ0FRWXdFZ1lEVlIwVEFRSC9CQWd3QmdFQi93SUJBREFkQmdOVkhRNEVGZ1FVNnU0VnVQVzgrR2ROCnZuOXQ4VkI1UWFPOHNXb3dDZ1lJS29aSXpqMEVBd0lEU0FBd1JRSWhBTzRod085UjB2Z3htMUphaGdTOWJnajQKZm9JNmc1QnRrUzRKcmgvc0ZpbzlBaUFRVVhnTUhXYzZSMVZhTHpXTkx0U0tkbHMvWTFuM3Z5MnlPZE1PL1Y4cApCZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

	responder, _ := httpmock.NewJsonResponder(200, res)
	httpmock.RegisterResponder("GET", `http://localhost:12345/identities/signer001`, responder)
	resolved, err := e.ResolveSigningKey(context.Background(), "signer001")
	assert.NoError(t, err)
	assert.Equal(t, "org1MSP::x509::CN=admin,OU=client::CN=fabric-ca-server", resolved)
}

func TestResolveSignerFailedFabricCARequest(t *testing.T) {
	e, cancel := newTestFabric()
	e.idCache = make(map[string]*fabIdentity)
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	res := make(map[string]string)

	responder, _ := httpmock.NewJsonResponder(503, res)
	httpmock.RegisterResponder("GET", `http://localhost:12345/identities/signer001`, responder)
	_, err := e.ResolveSigningKey(context.Background(), "signer001")
	assert.EqualError(t, err, "FF10284: Error from fabconnect: %!!(MISSING)s()")
}

func TestResolveSignerBadECertReturned(t *testing.T) {
	e, cancel := newTestFabric()
	e.idCache = make(map[string]*fabIdentity)
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	res := make(map[string]string)
	res["name"] = "signer001"
	res["mspId"] = "org1MSP"
	res["enrollmentCert"] = base64.StdEncoding.EncodeToString([]byte(badCert))
	res["caCert"] = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJmVENDQVNPZ0F3SUJBZ0lVYndac0FnK2Zac0FmSUF2VWFlWXBpOXF3NG9jd0NnWUlLb1pJemowRUF3SXcKR3pFWk1CY0dBMVVFQXhNUVptRmljbWxqTFdOaExYTmxjblpsY2pBZUZ3MHlNVEEzTWpNd01URTRNREJhRncwegpOakEzTVRrd01URTRNREJhTUJzeEdUQVhCZ05WQkFNVEVHWmhZbkpwWXkxallTMXpaWEoyWlhJd1dUQVRCZ2NxCmhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUlZNajcyR1dTeXk1UjRQN084ckpidXkrNHd6NWJWSE94dHBxRlUKamNadVE0Q2VSUGJoNDF3KzR1dFJsTlRTbFhLdTBMblBlVEZLSjlRT00xd0xwTGJtbzBVd1F6QU9CZ05WSFE4QgpBZjhFQkFNQ0FRWXdFZ1lEVlIwVEFRSC9CQWd3QmdFQi93SUJBREFkQmdOVkhRNEVGZ1FVNnU0VnVQVzgrR2ROCnZuOXQ4VkI1UWFPOHNXb3dDZ1lJS29aSXpqMEVBd0lEU0FBd1JRSWhBTzRod085UjB2Z3htMUphaGdTOWJnajQKZm9JNmc1QnRrUzRKcmgvc0ZpbzlBaUFRVVhnTUhXYzZSMVZhTHpXTkx0U0tkbHMvWTFuM3Z5MnlPZE1PL1Y4cApCZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

	responder, _ := httpmock.NewJsonResponder(200, res)
	httpmock.RegisterResponder("GET", `http://localhost:12345/identities/signer001`, responder)
	_, err := e.ResolveSigningKey(context.Background(), "signer001")
	assert.Contains(t, err.Error(), "FF10286: Failed to decode certificate:")
}

func TestResolveSignerBadCACertReturned(t *testing.T) {
	e, cancel := newTestFabric()
	e.idCache = make(map[string]*fabIdentity)
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	res := make(map[string]string)
	res["name"] = "signer001"
	res["mspId"] = "org1MSP"
	res["enrollmentCert"] = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJmVENDQVNPZ0F3SUJBZ0lVYndac0FnK2Zac0FmSUF2VWFlWXBpOXF3NG9jd0NnWUlLb1pJemowRUF3SXcKR3pFWk1CY0dBMVVFQXhNUVptRmljbWxqTFdOaExYTmxjblpsY2pBZUZ3MHlNVEEzTWpNd01URTRNREJhRncwegpOakEzTVRrd01URTRNREJhTUJzeEdUQVhCZ05WQkFNVEVHWmhZbkpwWXkxallTMXpaWEoyWlhJd1dUQVRCZ2NxCmhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUlZNajcyR1dTeXk1UjRQN084ckpidXkrNHd6NWJWSE94dHBxRlUKamNadVE0Q2VSUGJoNDF3KzR1dFJsTlRTbFhLdTBMblBlVEZLSjlRT00xd0xwTGJtbzBVd1F6QU9CZ05WSFE4QgpBZjhFQkFNQ0FRWXdFZ1lEVlIwVEFRSC9CQWd3QmdFQi93SUJBREFkQmdOVkhRNEVGZ1FVNnU0VnVQVzgrR2ROCnZuOXQ4VkI1UWFPOHNXb3dDZ1lJS29aSXpqMEVBd0lEU0FBd1JRSWhBTzRod085UjB2Z3htMUphaGdTOWJnajQKZm9JNmc1QnRrUzRKcmgvc0ZpbzlBaUFRVVhnTUhXYzZSMVZhTHpXTkx0U0tkbHMvWTFuM3Z5MnlPZE1PL1Y4cApCZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
	res["caCert"] = base64.StdEncoding.EncodeToString([]byte(badCert))

	responder, _ := httpmock.NewJsonResponder(200, res)
	httpmock.RegisterResponder("GET", `http://localhost:12345/identities/signer001`, responder)
	_, err := e.ResolveSigningKey(context.Background(), "signer001")
	assert.Contains(t, err.Error(), "FF10286: Failed to decode certificate:")
}

func TestGetUserNameWithMatches(t *testing.T) {
	result := getUserName(signer)
	assert.Equal(t, result, "signer001")
}

func TestGetUserNameNoMatches(t *testing.T) {
	result := getUserName("orgMSP::x509::OU=client::OU=CA")
	assert.Equal(t, result, "")
}

func TestHandleMessageBatchPinOK(t *testing.T) {
	data := []byte(`
[
  {
    "chaincodeId": "firefly",
    "blockNumber": 91,
    "transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
    "eventName": "BatchPin",
    "payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMxNjY3LCJuYW5vcyI6NzkxNDk5MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIjB4MTliODIwOTNkZTVjZTkyYTAxZTMzMzA0OGU4NzdlMjM3NDM1NGJmODQ2ZGQwMzQ4NjRlZjZmZmJkNjQzODc3MSJdfQ==",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  },
  {
    "chaincodeId": "firefly",
    "blockNumber": 77,
    "transactionId": "a488800a70c8f765871611168d422fb29cc37da2d0a196a3200c8068ba1706fd",
    "eventName": "BatchPin",
    "payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMxNjY3LCJuYW5vcyI6NzkxNDk5MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIjB4MTliODIwOTNkZTVjZTkyYTAxZTMzMzA0OGU4NzdlMjM3NDM1NGJmODQ2ZGQwMzQ4NjRlZjZmZmJkNjQzODc3MSJdfQ==",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }	
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: em,
	}
	e.initInfo.sub = &subscription{
		ID: "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e",
	}

	em.On("BatchPinComplete", mock.Anything, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", mock.Anything, mock.Anything).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	b := em.Calls[0].Arguments[0].(*blockchain.BatchPin)
	assert.Equal(t, "ns1", b.Namespace)
	assert.Equal(t, "e19af8b3-9060-4051-812d-7597d19adfb9", b.TransactionID.String())
	assert.Equal(t, "847d3bfd-0742-49ef-b65d-3fed15f5b0a6", b.BatchID.String())
	assert.Equal(t, "d71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be", b.BatchHash.String())
	assert.Equal(t, "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD", b.BatchPayloadRef)
	assert.Equal(t, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", em.Calls[0].Arguments[1])
	assert.Len(t, b.Contexts, 2)
	assert.Equal(t, "68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a", b.Contexts[0].String())
	assert.Equal(t, "19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771", b.Contexts[1].String())

	em.AssertExpectations(t)

}

func TestHandleMessageEmptyPayloadRef(t *testing.T) {
	data := []byte(`
[
  {
    "chaincodeId": "firefly",
    "blockNumber": 91,
    "transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
    "eventName": "BatchPin",
    "payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMyMDQwLCJuYW5vcyI6MjI5MjM1MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiIiwiY29udGV4dHMiOlsiMHg2OGU0ZGE3OWY4MDViY2E1YjkxMmJjZGE5YzYzZDAzZTZlODY3MTA4ZGFiYjliOTQ0MTA5YWVhNTQxZWY1MjJhIiwiMHgxOWI4MjA5M2RlNWNlOTJhMDFlMzMzMDQ4ZTg3N2UyMzc0MzU0YmY4NDZkZDAzNDg2NGVmNmZmYmQ2NDM4NzcxIl19",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: em,
	}
	e.initInfo.sub = &subscription{
		ID: "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e",
	}

	em.On("BatchPinComplete", mock.Anything, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", mock.Anything, mock.Anything).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	b := em.Calls[0].Arguments[0].(*blockchain.BatchPin)
	assert.Equal(t, "ns1", b.Namespace)
	assert.Equal(t, "e19af8b3-9060-4051-812d-7597d19adfb9", b.TransactionID.String())
	assert.Equal(t, "847d3bfd-0742-49ef-b65d-3fed15f5b0a6", b.BatchID.String())
	assert.Equal(t, "d71eb138d74c229a388eb0e1abc03f4c7cbb21d4fc4b839fbf0ec73e4263f6be", b.BatchHash.String())
	assert.Empty(t, b.BatchPayloadRef)
	assert.Equal(t, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", em.Calls[0].Arguments[1])
	assert.Len(t, b.Contexts, 2)
	assert.Equal(t, "68e4da79f805bca5b912bcda9c63d03e6e867108dabb9b944109aea541ef522a", b.Contexts[0].String())
	assert.Equal(t, "19b82093de5ce92a01e333048e877e2374354bf846dd034864ef6ffbd6438771", b.Contexts[1].String())

	em.AssertExpectations(t)

}

func TestHandleMessageBatchPinExit(t *testing.T) {
	data := []byte(`
[
  {
    "chaincodeId": "firefly",
    "blockNumber": 91,
    "transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
    "eventName": "BatchPin",
    "payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMyMDQwLCJuYW5vcyI6MjI5MjM1MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiIiwiY29udGV4dHMiOlsiMHg2OGU0ZGE3OWY4MDViY2E1YjkxMmJjZGE5YzYzZDAzZTZlODY3MTA4ZGFiYjliOTQ0MTA5YWVhNTQxZWY1MjJhIiwiMHgxOWI4MjA5M2RlNWNlOTJhMDFlMzMzMDQ4ZTg3N2UyMzc0MzU0YmY4NDZkZDAzNDg2NGVmNmZmYmQ2NDM4NzcxIl19",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: em,
	}
	e.initInfo.sub = &subscription{
		ID: "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e",
	}

	em.On("BatchPinComplete", mock.Anything, "u0vgwu9s00-x509::CN=user2,OU=client::CN=fabric-ca-server", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

}

func TestHandleMessageBatchPinEmpty(t *testing.T) {
	data := []byte(`
[
  {
    "chaincodeId": "firefly",
    "blockNumber": 91,
    "transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
    "eventName": "BatchPin",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	e.initInfo.sub = &subscription{
		ID: "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e",
	}

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageUnknownEventName(t *testing.T) {
	data := []byte(`
[
  {
    "chaincodeId": "firefly",
    "blockNumber": 91,
    "transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
    "eventName": "UnknownEvent",
    "subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
  }
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	e.initInfo.sub = &subscription{
		ID: "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e",
	}

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadBatchHash(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	e.initInfo.sub = &subscription{
		ID: "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e",
	}
	data := []byte(`[{
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"eventName": "BatchPin",
		"payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMzMjMzLCJuYW5vcyI6NTAwMDc3MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIhZ29vZCIsInBheWxvYWRSZWYiOiJRbWY0MTJqUVppdVZVdGRnbkIzNkZYRlg3eGc1VjZLRWJTSjRkcFF1aGtMeWZEIiwiY29udGV4dHMiOltdfQ==",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
	}]`)
	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadPin(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	e.initInfo.sub = &subscription{
		ID: "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e",
	}
	data := []byte(`[{
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"eventName": "BatchPin",
		"payload": "eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMzMzQ0LCJuYW5vcyI6OTY1NjE4MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIiFnb29kIl19",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
	}]`)
	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadPayloadEncoding(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	e.initInfo.sub = &subscription{
		ID: "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e",
	}
	data := []byte(`[{
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"eventName": "BatchPin",
		"payload": "--eyJzaWduZXIiOiJ1MHZnd3U5czAwLXg1MDk6OkNOPXVzZXIyLE9VPWNsaWVudDo6Q049ZmFicmljLWNhLXNlcnZlciIsInRpbWVzdGFtcCI6eyJzZWNvbmRzIjoxNjMwMDMzMzQ0LCJuYW5vcyI6OTY1NjE4MDAwfSwibmFtZXNwYWNlIjoibnMxIiwidXVpZHMiOiIweGUxOWFmOGIzOTA2MDQwNTE4MTJkNzU5N2QxOWFkZmI5ODQ3ZDNiZmQwNzQyNDllZmI2NWQzZmVkMTVmNWIwYTYiLCJiYXRjaEhhc2giOiIweGQ3MWViMTM4ZDc0YzIyOWEzODhlYjBlMWFiYzAzZjRjN2NiYjIxZDRmYzRiODM5ZmJmMGVjNzNlNDI2M2Y2YmUiLCJwYXlsb2FkUmVmIjoiUW1mNDEyalFaaXVWVXRkZ25CMzZGWEZYN3hnNVY2S0ViU0o0ZHBRdWhrTHlmRCIsImNvbnRleHRzIjpbIjB4NjhlNGRhNzlmODA1YmNhNWI5MTJiY2RhOWM2M2QwM2U2ZTg2NzEwOGRhYmI5Yjk0NDEwOWFlYTU0MWVmNTIyYSIsIiFnb29kIl19",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
	}]`)
	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchPinBadPayloadUUIDs(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	e.initInfo.sub = &subscription{
		ID: "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e",
	}
	data := []byte(`[{
		"chaincodeId": "firefly",
		"blockNumber": 91,
		"transactionId": "ce79343000e851a0c742f63a733ce19a5f8b9ce1c719b6cecd14f01bcf81fff2",
		"eventName": "BatchPin",
		"payload": "eyJzaWduZXIiOiJPcmcxTVNQOjp4NTA5OjpDTj0weDA3YTA5YzE2ZWQ5ZWYyYmIwYmNiYzUxNzk4OGU4MmIzNzA0NDk4YzQsT1U9Y2xpZW50OjpDTj1mYWJyaWNfY2Eub3JnMS5leGFtcGxlLmNvbSxPVT1IeXBlcmxlZGdlciBGYWJyaWMsTz1vcmcxLmV4YW1wbGUuY29tLEw9U2FuIEZyYW5jaXNjbyxTVD1DYWxpZm9ybmlhLEM9VVMiLCJ0aW1lc3RhbXAiOnsic2Vjb25kcyI6MTYzNDMwNDAzNSwibmFub3MiOjI5OTcwMjUwMH0sIm5hbWVzcGFjZSI6ImRlZmF1bHQiLCJ1dWlkcyI6IjB4MjYxNjY2OGExYjIxNGFkY2JkN2IyOGE3ZjkxMDM3MjNiNzEwMTk4ODc4NWE0NzZmYTM2YjM1OWUyZCIsImJhdGNoSGFzaCI6IjB4ZDRkYjliNmQ3YWYzNWQyYTU4ZDgwYmFlY2QxOTI2MjM0Mzg0YmIxODljMGQ2YmRmMzQzNGMyZmE5YzY2MGM0MiIsInBheWxvYWRSZWYiOiJRbWNuRUVjY0tkV0tHZDZqV2ZaNGZOV2dqTnBVSm15bm5lV0tMRW11Rjh3UlNDIiwiY29udGV4dHMiOlsiMHgzN2E4ZWVjMWNlMTk2ODdkMTMyZmUyOTA1MWRjYTYyOWQxNjRlMmM0OTU4YmExNDFkNWY0MTMzYTMzZjA2ODhmIl19",
		"subId": "sb-0910f6a8-7bd6-4ced-453e-2db68149ce8e"
	}]`)
	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestHandleMessageBatchBadJSON(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	e := &Fabric{callbacks: em}
	err := e.handleMessageBatch(context.Background(), []interface{}{10, 20})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(em.Calls))
}

func TestEventLoopContextCancelled(t *testing.T) {
	e, cancel := newTestFabric()
	cancel()
	r := make(<-chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	wsm.On("Receive").Return(r)
	wsm.On("Close").Return()
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopReceiveClosed(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	close(r)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Close").Return()
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopSendClosed(t *testing.T) {
	e, cancel := newTestFabric()
	cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	close(r)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Close").Return()
	wsm.On("Send", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	e.closed = make(chan struct{})
	e.eventLoop() // we're simply looking for it exiting
}

func TestEventLoopUnexpectedMessage(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	r := make(chan []byte)
	wsm := e.wsconn.(*wsmocks.WSClient)
	wsm.On("Receive").Return((<-chan []byte)(r))
	wsm.On("Close").Return()
	e.closed = make(chan struct{})
	operationID := fftypes.NewUUID()
	data := []byte(`{
		"_id": "6fb94fff-81d3-4094-567d-e031b1871694",
		"errorMessage": "Packing arguments for method 'broadcastBatch': abi: cannot use [3]uint8 as type [32]uint8 as argument",
		"headers": {
			"id": "3a37b17b-13b6-4dc5-647a-07c11eae0be3",
			"requestId": "` + operationID.String() + `",
			"requestOffset": "zzn4y4v4si-zzjjepe9x4-requests:0:0",
			"timeElapsed": 0.020969053,
			"timeReceived": "2021-05-31T02:35:11.458880504Z",
			"type": "Error"
		},
		"receivedAt": 1622428511616,
		"requestPayload": "{\"from\":\"0x91d2b4381a4cd5c7c0f27565a7d4b829844c8635\",\"gas\":0,\"gasPrice\":0,\"headers\":{\"id\":\"6fb94fff-81d3-4094-567d-e031b1871694\",\"type\":\"SendTransaction\"},\"method\":{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"txnId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"batchId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"payloadRef\",\"type\":\"bytes32\"}],\"name\":\"broadcastBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},\"params\":[\"12345\",\"!\",\"!\"],\"to\":\"0xd3266a857285fb75eb7df37353b4a15c8bb828f5\",\"value\":0}"
	}`)
	em := e.callbacks.(*blockchainmocks.Callbacks)
	txsu := em.On("BlockchainOpUpdate",
		operationID,
		fftypes.OpStatusFailed,
		"Packing arguments for method 'broadcastBatch': abi: cannot use [3]uint8 as type [32]uint8 as argument",
		mock.Anything).Return(fmt.Errorf("Shutdown"))
	done := make(chan struct{})
	txsu.RunFn = func(a mock.Arguments) {
		close(done)
	}

	go e.eventLoop()
	r <- []byte(`!badjson`)        // ignored bad json
	r <- []byte(`"not an object"`) // ignored wrong type
	r <- data
	e.ctx.Done()
}

func TestHandleReceiptTXSuccess(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: em,
		wsconn:    wsm,
	}

	var reply fftypes.JSONObject
	operationID := fftypes.NewUUID()
	data := []byte(`{
		"_id": "748e7587-9e72-4244-7351-808f69b88291",
    "headers": {
        "id": "0ef91fb6-09c5-4ca2-721c-74b4869097c2",
        "requestId": "` + operationID.String() + `",
        "requestOffset": "",
        "timeElapsed": 0.475721,
        "timeReceived": "2021-08-27T03:04:34.199742Z",
        "type": "TransactionSuccess"
    },
    "receivedAt": 1630033474675
  }`)

	em.On("BlockchainOpUpdate",
		operationID,
		fftypes.OpStatusSucceeded,
		"",
		mock.Anything).Return(nil)

	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	err = e.handleReceipt(context.Background(), reply)
	assert.NoError(t, err)

}

func TestHandleReceiptNoRequestID(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: em,
		wsconn:    wsm,
	}

	var reply fftypes.JSONObject
	data := []byte(`{}`)
	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	err = e.handleReceipt(context.Background(), reply)
	assert.NoError(t, err)
}

func TestHandleReceiptBadRequestID(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: em,
		wsconn:    wsm,
	}

	var reply fftypes.JSONObject
	data := []byte(`{
		"_id": "748e7587-9e72-4244-7351-808f69b88291",
    "headers": {
        "id": "0ef91fb6-09c5-4ca2-721c-74b4869097c2",
        "requestId": "bad-UUID",
        "requestOffset": "",
        "timeElapsed": 0.475721,
        "timeReceived": "2021-08-27T03:04:34.199742Z",
        "type": "TransactionSuccess"
    },
    "receivedAt": 1630033474675
  }`)

	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	err = e.handleReceipt(context.Background(), reply)
	assert.NoError(t, err)
}

func TestHandleReceiptFailedTx(t *testing.T) {
	em := &blockchainmocks.Callbacks{}
	wsm := &wsmocks.WSClient{}
	e := &Fabric{
		ctx:       context.Background(),
		topic:     "topic1",
		callbacks: em,
		wsconn:    wsm,
	}

	var reply fftypes.JSONObject
	operationID := fftypes.NewUUID()
	data := []byte(`{
		"_id": "748e7587-9e72-4244-7351-808f69b88291",
    "headers": {
        "id": "0ef91fb6-09c5-4ca2-721c-74b4869097c2",
        "requestId": "` + operationID.String() + `",
        "requestOffset": "",
        "timeElapsed": 0.475721,
        "timeReceived": "2021-08-27T03:04:34.199742Z",
        "type": "TransactionFailure"
    },
    "receivedAt": 1630033474675
  }`)

	em.On("BlockchainOpUpdate",
		operationID,
		fftypes.OpStatusFailed,
		"",
		mock.Anything).Return(nil)

	err := json.Unmarshal(data, &reply)
	assert.NoError(t, err)
	err = e.handleReceipt(context.Background(), reply)
	assert.NoError(t, err)
}

func TestFormatNil(t *testing.T) {
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", hexFormatB32(nil))
}

func TestAddSubscription(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.initInfo.stream = &eventStream{
		ID: "es-1",
	}
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"channel":   "firefly",
				"chaincode": "mycode",
			}.String()),
			Event: &fftypes.FFISerializedEvent{},
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/subscriptions`,
		httpmock.NewJsonResponderOrPanic(200, &subscription{}))

	err := e.AddSubscription(context.Background(), sub)

	assert.NoError(t, err)
}

func TestAddSubscriptionBadLocation(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.initInfo.stream = &eventStream{
		ID: "es-1",
	}
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.JSONAnyPtr(""),
			Event:    &fftypes.FFISerializedEvent{},
		},
	}

	err := e.AddSubscription(context.Background(), sub)

	assert.Regexp(t, "FF10310", err)
}

func TestAddSubscriptionFail(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.initInfo.stream = &eventStream{
		ID: "es-1",
	}
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &fftypes.ContractSubscriptionInput{
		ContractSubscription: fftypes.ContractSubscription{
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"channel":   "firefly",
				"chaincode": "mycode",
			}.String()),
			Event: &fftypes.FFISerializedEvent{},
		},
	}

	httpmock.RegisterResponder("POST", `http://localhost:12345/subscriptions`,
		httpmock.NewStringResponder(500, "pop"))

	err := e.AddSubscription(context.Background(), sub)

	assert.Regexp(t, "FF10284", err)
	assert.Regexp(t, "pop", err)
}

func TestDeleteSubscription(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.initInfo.stream = &eventStream{
		ID: "es-1",
	}
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &fftypes.ContractSubscription{
		ProtocolID: "sb-1",
	}

	httpmock.RegisterResponder("DELETE", `http://localhost:12345/subscriptions/sb-1`,
		httpmock.NewStringResponder(204, ""))

	err := e.DeleteSubscription(context.Background(), sub)

	assert.NoError(t, err)
}

func TestDeleteSubscriptionFail(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()

	e.initInfo.stream = &eventStream{
		ID: "es-1",
	}
	e.streams = &streamManager{
		client: e.client,
	}

	sub := &fftypes.ContractSubscription{
		ProtocolID: "sb-1",
	}

	httpmock.RegisterResponder("DELETE", `http://localhost:12345/subscriptions/sb-1`,
		httpmock.NewStringResponder(500, "pop"))

	err := e.DeleteSubscription(context.Background(), sub)

	assert.Regexp(t, "FF10284", err)
	assert.Regexp(t, "pop", err)
}

func TestHandleMessageContractEvent(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
	  "blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"eventName": "AssetCreated",
		"payload": "eyJBcHByYWlzZWRWYWx1ZSI6MTAsIkNvbG9yIjoicmVkIiwiSUQiOiIxMjM0IiwiT3duZXIiOiJtZSIsIlNpemUiOjN9",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: em,
	}
	e.initInfo.sub = &subscription{
		ID: "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
	}

	em.On("ContractEvent", mock.Anything).Return(nil)

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	ev := em.Calls[0].Arguments[0].(*blockchain.ContractEvent)
	assert.Equal(t, "sb-cb37cc07-e873-4f58-44ab-55add6bba320", ev.Subscription)
	assert.Equal(t, "AssetCreated", ev.Event.Name)

	outputs := fftypes.JSONObject{
		"AppraisedValue": float64(10),
		"Color":          "red",
		"ID":             "1234",
		"Owner":          "me",
		"Size":           float64(3),
	}
	assert.Equal(t, outputs, ev.Event.Output)

	info := fftypes.JSONObject{
		"blockNumber":   float64(10),
		"chaincodeId":   "basic",
		"eventName":     "AssetCreated",
		"subId":         "sb-cb37cc07-e873-4f58-44ab-55add6bba320",
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
	}
	assert.Equal(t, info, ev.Event.Info)

	em.AssertExpectations(t)
}

func TestHandleMessageContractEventBadPayload(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
	  "blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"eventName": "AssetCreated",
		"payload": "bad",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: em,
	}
	e.initInfo.sub = &subscription{
		ID: "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
	}

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.NoError(t, err)

	em.AssertExpectations(t)
}

func TestHandleMessageContractEventError(t *testing.T) {
	data := []byte(`
[
	{
		"chaincodeId": "basic",
	  "blockNumber": 10,
		"transactionId": "4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d",
		"eventName": "AssetCreated",
		"payload": "eyJBcHByYWlzZWRWYWx1ZSI6MTAsIkNvbG9yIjoicmVkIiwiSUQiOiIxMjM0IiwiT3duZXIiOiJtZSIsIlNpemUiOjN9",
		"subId": "sb-cb37cc07-e873-4f58-44ab-55add6bba320"
	}
]`)

	em := &blockchainmocks.Callbacks{}
	e := &Fabric{
		callbacks: em,
	}
	e.initInfo.sub = &subscription{
		ID: "sb-b5b97a4e-a317-4053-6400-1474650efcb5",
	}

	em.On("ContractEvent", mock.Anything).Return(fmt.Errorf("pop"))

	var events []interface{}
	err := json.Unmarshal(data, &events)
	assert.NoError(t, err)
	err = e.handleMessageBatch(context.Background(), events)
	assert.EqualError(t, err, "pop")

	em.AssertExpectations(t)
}

func TestInvokeContractOK(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)
			assert.Equal(t, signingKey, req.URL.Query().Get(defaultPrefixShort+"-signer"))
			assert.Equal(t, "firefly", req.URL.Query().Get(defaultPrefixShort+"-channel"))
			assert.Equal(t, "simplestorage", req.URL.Query().Get(defaultPrefixShort+"-chaincode"))
			assert.Equal(t, "false", req.URL.Query().Get(defaultPrefixShort+"-sync"))
			assert.Equal(t, "1", body["args"].(map[string]interface{})["x"])
			assert.Equal(t, "2", body["args"].(map[string]interface{})["y"])
			return httpmock.NewJsonResponderOrPanic(200, asyncTXSubmission{})(req)
		})
	_, err = e.InvokeContract(context.Background(), nil, signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params)
	assert.NoError(t, err)
}

func TestInvokeContractChaincodeNotSet(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.InvokeContract(context.Background(), nil, signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params)
	assert.Regexp(t, "FF10310", err)
}

func TestInvokeContractFabconnectError(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{
		Channel:   "fabric",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponderOrPanic(400, asyncTXSubmission{})(req)
		})
	_, err = e.InvokeContract(context.Background(), nil, signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params)
	assert.Regexp(t, "FF10284", err)
}

func TestInvokeContractUnmarshalResponseError(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			json.NewDecoder(req.Body).Decode(&body)

			assert.Equal(t, signingKey, req.URL.Query().Get(defaultPrefixShort+"-signer"))
			assert.Equal(t, "firefly", req.URL.Query().Get(defaultPrefixShort+"-channel"))
			assert.Equal(t, "simplestorage", req.URL.Query().Get(defaultPrefixShort+"-chaincode"))
			assert.Equal(t, "false", req.URL.Query().Get(defaultPrefixShort+"-sync"))
			assert.Equal(t, "1", body["args"].(map[string]interface{})["x"])
			assert.Equal(t, "2", body["args"].(map[string]interface{})["y"])
			return httpmock.NewStringResponder(200, "[definitely not JSON}")(req)
		})
	_, err = e.InvokeContract(context.Background(), nil, signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params)
	assert.Regexp(t, "invalid character", err)
}

func TestQueryContractOK(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": float64(1),
		"y": float64(2),
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	_, err = e.QueryContract(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes), method, params)
	assert.EqualError(t, err, "not yet supported")
}

func TestValidateContractLocation(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	location := &Location{
		Channel:   "firefly",
		Chaincode: "simplestorage",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	err = e.ValidateContractLocation(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes))
	assert.NoError(t, err)
}

func TestValidateNoContractLocationChaincode(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	location := &Location{
		Channel: "firefly",
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	err = e.ValidateContractLocation(context.Background(), fftypes.JSONAnyPtrBytes(locationBytes))
	assert.Regexp(t, "FF10310", err)
}

func TestParseParam(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	param := &fftypes.FFIParam{
		Name: "x",
		Type: "integer",
	}
	err := e.ValidateFFIParam(context.Background(), param)
	assert.NoError(t, err)
}

func TestInvokeJSONEncodeParamsError(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()
	httpmock.ActivateNonDefault(e.client.GetClient())
	defer httpmock.DeactivateAndReset()
	signingKey := fftypes.NewRandB32().String()
	location := &Location{
		Channel:   "fabric",
		Chaincode: "simplestorage",
	}
	method := testFFIMethod()
	params := map[string]interface{}{
		"x": map[bool]interface{}{true: false},
		"y": float64(2),
	}
	locationBytes, err := json.Marshal(location)
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", `http://localhost:12345/transactions`,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponderOrPanic(400, asyncTXSubmission{})(req)
		})
	_, err = e.InvokeContract(context.Background(), nil, signingKey, fftypes.JSONAnyPtrBytes(locationBytes), method, params)
	assert.Regexp(t, "FF10151", err)
}
