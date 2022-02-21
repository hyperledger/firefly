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

package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

var utConfPrefix = config.NewPluginConfig("ipfs_unit_tests")

func resetConf() {
	config.Reset()
	i := &IPFS{}
	i.InitPrefix(utConfPrefix)
}

func TestInitMissingAPIURL(t *testing.T) {
	i := &IPFS{}
	resetConf()

	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	err := i.Init(context.Background(), utConfPrefix, &sharedstoragemocks.Callbacks{})
	assert.Regexp(t, "FF10138", err)
}

func TestInitMissingGWURL(t *testing.T) {
	i := &IPFS{}
	resetConf()

	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	err := i.Init(context.Background(), utConfPrefix, &sharedstoragemocks.Callbacks{})
	assert.Regexp(t, "FF10138", err)
}

func TestInit(t *testing.T) {
	i := &IPFS{}
	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")

	err := i.Init(context.Background(), utConfPrefix, &sharedstoragemocks.Callbacks{})
	assert.Equal(t, "ipfs", i.Name())
	assert.NoError(t, err)
	assert.NotNil(t, i.Capabilities())
}

func TestIPFSUploadSuccess(t *testing.T) {
	i := &IPFS{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPCustomClient, mockedClient)

	err := i.Init(context.Background(), utConfPrefix, &sharedstoragemocks.Callbacks{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("POST", "http://localhost:12345/api/v0/add",
		httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
			"Hash": "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		}))

	data := []byte(`hello world`)
	payloadRef, err := i.PublishData(context.Background(), bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Equal(t, `Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD`, payloadRef)

}

func TestIPFSUploadFail(t *testing.T) {
	i := &IPFS{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPCustomClient, mockedClient)

	err := i.Init(context.Background(), utConfPrefix, &sharedstoragemocks.Callbacks{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("POST", "http://localhost:12345/api/v0/add",
		httpmock.NewJsonResponderOrPanic(500, map[string]interface{}{"error": "pop"}))

	data := []byte(`hello world`)
	_, err = i.PublishData(context.Background(), bytes.NewReader(data))
	assert.Regexp(t, "FF10136", err)

}

func TestIPFSDownloadSuccess(t *testing.T) {
	i := &IPFS{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPCustomClient, mockedClient)

	err := i.Init(context.Background(), utConfPrefix, &sharedstoragemocks.Callbacks{})
	assert.NoError(t, err)

	data := []byte(`{"hello": "world"}`)
	httpmock.RegisterResponder("GET", "http://localhost:12345/ipfs/QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL",
		httpmock.NewBytesResponder(200, data))

	r, err := i.RetrieveData(context.Background(), "QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL")
	assert.NoError(t, err)
	defer r.Close()

	var resJSON fftypes.JSONObject
	json.NewDecoder(r).Decode(&resJSON)
	assert.Equal(t, "world", resJSON["hello"])

}

func TestIPFSDownloadFail(t *testing.T) {
	i := &IPFS{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPCustomClient, mockedClient)

	err := i.Init(context.Background(), utConfPrefix, &sharedstoragemocks.Callbacks{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("GET", "http://localhost:12345/ipfs/QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL",
		httpmock.NewJsonResponderOrPanic(500, map[string]interface{}{"error": "pop"}))

	_, err = i.RetrieveData(context.Background(), "QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL")
	assert.Regexp(t, "FF10136", err)

}

func TestIPFSDownloadError(t *testing.T) {
	i := &IPFS{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPCustomClient, mockedClient)

	err := i.Init(context.Background(), utConfPrefix, &sharedstoragemocks.Callbacks{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("GET", "http://localhost:12345/ipfs/QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL",
		httpmock.NewErrorResponder(fmt.Errorf("pop")))

	_, err = i.RetrieveData(context.Background(), "QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL")
	assert.Regexp(t, "FF10136", err)

}
