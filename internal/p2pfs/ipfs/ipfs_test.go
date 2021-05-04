// Copyright © 2021 Kaleido, Inc.
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
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/kaleido-io/firefly/internal/ffresty"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/p2pfsmocks"
	"github.com/stretchr/testify/assert"
)

func TestInitMissingURL(t *testing.T) {
	i := &IPFS{}
	err := i.Init(context.Background(), &Config{}, &p2pfsmocks.Events{})
	assert.Regexp(t, "FF10138", err.Error())
}

func TestConfigInterfaceCorrect(t *testing.T) {
	i := &IPFS{}
	_, ok := i.ConfigInterface().(*Config)
	assert.True(t, ok)
}

func TestInit(t *testing.T) {
	i := &IPFS{}
	conf := i.ConfigInterface().(*Config)
	conf.URL = "http://localhost:2345"
	err := i.Init(context.Background(), conf, &p2pfsmocks.Events{})
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

func TestIPFSHashToBytes32BadData(t *testing.T) {
	i := IPFS{ctx: context.Background()}
	ipfsHash := "!!"
	_, err := i.ipfsHashToBytes32(ipfsHash)
	assert.Regexp(t, "FF10135", err.Error())
}

func TestIPFSHashToBytes32WrongLen(t *testing.T) {
	i := IPFS{ctx: context.Background()}
	ipfsHash := "QmRAQfHNnknnz8S936M2yJGhhVNA6wXJ4jTRP3VXtptm"
	_, err := i.ipfsHashToBytes32(ipfsHash)
	assert.Regexp(t, "FF10135", err.Error())
}

func TestIPFSUploadSuccess(t *testing.T) {
	i := &IPFS{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	err := i.Init(context.Background(), &Config{
		HTTPConfig: ffresty.HTTPConfig{
			URL:        "http://localhost:12345",
			HttpClient: mockedClient,
		},
	}, &p2pfsmocks.Events{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("POST", "http://localhost:12345/api/v0/add",
		httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
			"Hash": "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD",
		}))

	data := []byte(`hello world`)
	hash, err := i.PublishData(context.Background(), bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Equal(t, `f852c7fa62f971817f54d8a80dcd63fcf7098b3cbde9ae8ec1ee449013ec5db0`, hash.String())

}

func TestIPFSUploadFail(t *testing.T) {
	i := &IPFS{}

	mockedClient := &http.Client{}
	httpmock.ActivateNonDefault(mockedClient)
	defer httpmock.DeactivateAndReset()

	err := i.Init(context.Background(), &Config{
		HTTPConfig: ffresty.HTTPConfig{
			URL:        "http://localhost:12345",
			HttpClient: mockedClient,
		},
	}, &p2pfsmocks.Events{})
	assert.NoError(t, err)

	httpmock.RegisterResponder("POST", "http://localhost:12345/api/v0/add",
		httpmock.NewJsonResponderOrPanic(500, map[string]interface{}{"error": "pop"}))

	data := []byte(`hello world`)
	_, err = i.PublishData(context.Background(), bytes.NewReader(data))
	assert.Regexp(t, "FF10136", err.Error())

}
