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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/restclient"
	"github.com/hyperledger-labs/firefly/mocks/publicstoragemocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
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
	err := i.Init(context.Background(), utConfPrefix, &publicstoragemocks.Callbacks{})
	assert.Regexp(t, "FF10138", err)
}

func TestInitMissingGWURL(t *testing.T) {
	i := &IPFS{}
	resetConf()

	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	err := i.Init(context.Background(), utConfPrefix, &publicstoragemocks.Callbacks{})
	assert.Regexp(t, "FF10138", err)
}

func TestInit(t *testing.T) {
	i := &IPFS{}
	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")

	err := i.Init(context.Background(), utConfPrefix, &publicstoragemocks.Callbacks{})
	assert.Equal(t, "ipfs", i.Name())
	assert.NoError(t, err)
	assert.NotNil(t, i.Capabilities())
}

func TestIPFSHashToBytes32(t *testing.T) {
	i := IPFS{ctx: context.Background()}
	ipfsHash := "QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL"
	var expectedSHA256 fftypes.Bytes32
	expectedSHA256.UnmarshalText([]byte("29f35e27c4b008b58d3e70fda9518eac65fc1ef4894de91f42b4799841c0a683"))
	res, err := i.ipfsHashToBytes32(ipfsHash)
	assert.NoError(t, err)
	assert.Equal(t, expectedSHA256, *res)
}

func TestBytes32ToIPFSHash(t *testing.T) {
	var b32 fftypes.Bytes32
	hex.Decode(b32[:], []byte("29f35e27c4b008b58d3e70fda9518eac65fc1ef4894de91f42b4799841c0a683"))
	i := IPFS{ctx: context.Background()}
	assert.Equal(t, "QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL", i.bytes32ToIPFSHash(&b32))
}

func TestIPFSHashToBytes32BadData(t *testing.T) {
	i := IPFS{ctx: context.Background()}
	ipfsHash := "!!"
	_, err := i.ipfsHashToBytes32(ipfsHash)
	assert.Regexp(t, "FF10135", err)
}

func TestIPFSHashToBytes32WrongLen(t *testing.T) {
	i := IPFS{ctx: context.Background()}
	ipfsHash := "QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptm"
	_, err := i.ipfsHashToBytes32(ipfsHash)
	assert.Regexp(t, "FF10135", err)
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

	err := i.Init(context.Background(), utConfPrefix, &publicstoragemocks.Callbacks{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("POST", "http://localhost:12345/api/v0/add",
		httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
			"Hash": "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		}))

	data := []byte(`hello world`)
	hash, backendID, err := i.PublishData(context.Background(), bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Equal(t, `Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD`, backendID)
	assert.Equal(t, `f852c7fa62f971817f54d8a80dcd63fcf7098b3cbde9ae8ec1ee449013ec5db0`, hash.String())

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

	err := i.Init(context.Background(), utConfPrefix, &publicstoragemocks.Callbacks{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("POST", "http://localhost:12345/api/v0/add",
		httpmock.NewJsonResponderOrPanic(500, map[string]interface{}{"error": "pop"}))

	data := []byte(`hello world`)
	_, _, err = i.PublishData(context.Background(), bytes.NewReader(data))
	assert.Regexp(t, "FF10136", err)

}

func TestIPFSDownloadSuccess(t *testing.T) {
	i := &IPFS{}
	var b32 fftypes.Bytes32
	hex.Decode(b32[:], []byte("29f35e27c4b008b58d3e70fda9518eac65fc1ef4894de91f42b4799841c0a683"))

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPCustomClient, mockedClient)

	err := i.Init(context.Background(), utConfPrefix, &publicstoragemocks.Callbacks{})
	assert.NoError(t, err)

	data := []byte(`{"hello": "world"}`)
	httpmock.RegisterResponder("GET", "http://localhost:12345/ipfs/QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL",
		httpmock.NewBytesResponder(200, data))

	r, err := i.RetrieveData(context.Background(), &b32)
	assert.NoError(t, err)
	defer r.Close()

	var resJSON fftypes.JSONObject
	json.NewDecoder(r).Decode(&resJSON)
	assert.Equal(t, "world", resJSON["hello"])

}

func TestIPFSDownloadFail(t *testing.T) {
	i := &IPFS{}
	var b32 fftypes.Bytes32
	hex.Decode(b32[:], []byte("29f35e27c4b008b58d3e70fda9518eac65fc1ef4894de91f42b4799841c0a683"))

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPCustomClient, mockedClient)

	err := i.Init(context.Background(), utConfPrefix, &publicstoragemocks.Callbacks{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("GET", "http://localhost:12345/ipfs/QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL",
		httpmock.NewJsonResponderOrPanic(500, map[string]interface{}{"error": "pop"}))

	_, err = i.RetrieveData(context.Background(), &b32)
	assert.Regexp(t, "FF10136", err)

}

func TestIPFSDownloadError(t *testing.T) {
	i := &IPFS{}
	var b32 fftypes.Bytes32
	hex.Decode(b32[:], []byte("29f35e27c4b008b58d3e70fda9518eac65fc1ef4894de91f42b4799841c0a683"))

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	resetConf()
	utConfPrefix.SubPrefix(IPFSConfAPISubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPConfigURL, "http://localhost:12345")
	utConfPrefix.SubPrefix(IPFSConfGatewaySubconf).Set(restclient.HTTPCustomClient, mockedClient)

	err := i.Init(context.Background(), utConfPrefix, &publicstoragemocks.Callbacks{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("GET", "http://localhost:12345/ipfs/QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptmmL",
		httpmock.NewErrorResponder(fmt.Errorf("pop")))

	_, err = i.RetrieveData(context.Background(), &b32)
	assert.Regexp(t, "FF10136", err)

}
