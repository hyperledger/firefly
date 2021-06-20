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

package e2e

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

var (
	urlGetNamespaces    = "/namespaces"
	urlUploadData       = "/namespaces/default/data"
	urlGetMessages      = "/namespaces/default/messages"
	urlBroadcastMessage = "/namespaces/default/broadcast/message"
	urlPrivateMessage   = "/namespaces/default/send/message"
	urlGetData          = "/namespaces/default/data"
	urlGetDataBlob      = "/namespaces/default/data/%s/blob"
	urlGetOrganizations = "/network/organizations"
)

func NewResty(t *testing.T) *resty.Client {
	client := resty.New()
	client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		t.Logf("==> %s %s", req.Method, req.URL)
		return nil
	})
	client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		if resp == nil {
			return nil
		}
		t.Logf("<== %d", resp.StatusCode())
		if resp.IsError() {
			t.Logf("<!! %s", resp.String())
			t.Logf("Headers: %+v", resp.Header())
		}
		return nil
	})

	return client
}

func GetNamespaces(client *resty.Client) (*resty.Response, error) {
	return client.R().
		SetResult(&[]fftypes.Namespace{}).
		Get(urlGetNamespaces)
}

func GetMessages(t *testing.T, client *resty.Client, startTime time.Time, msgType fftypes.MessageType, expectedStatus int) (msgs []*fftypes.Message) {
	path := urlGetMessages
	resp, err := client.R().
		SetQueryParam("type", string(msgType)).
		SetQueryParam("created", fmt.Sprintf(">%d", startTime.UnixNano())).
		SetResult(&msgs).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return msgs
}

func GetData(t *testing.T, client *resty.Client, startTime time.Time, expectedStatus int) (data []*fftypes.Data) {
	path := urlGetData
	resp, err := client.R().
		SetQueryParam("created", fmt.Sprintf(">%d", startTime.UnixNano())).
		SetResult(&data).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return data
}

func GetBlob(t *testing.T, client *resty.Client, data *fftypes.Data, expectedStatus int) []byte {
	path := fmt.Sprintf(urlGetDataBlob, data.ID)
	resp, err := client.R().
		SetDoNotParseResponse(true).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	blob, err := ioutil.ReadAll(resp.RawBody())
	require.NoError(t, err)
	return blob
}

func GetOrgs(t *testing.T, client *resty.Client, expectedStatus int) (orgs []*fftypes.Organization) {
	path := urlGetOrganizations
	resp, err := client.R().
		SetResult(&orgs).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return orgs
}

func BroadcastMessage(client *resty.Client, data *fftypes.DataRefOrValue) (*resty.Response, error) {
	return client.R().
		SetBody(fftypes.MessageInput{
			InputData: fftypes.InputData{data},
		}).
		Post(urlBroadcastMessage)
}

func BroadcastBlobMessage(t *testing.T, client *resty.Client) (*resty.Response, error) {

	r, _ := rand.Int(rand.Reader, big.NewInt(1024*1024))
	blob := make([]byte, r.Int64()+1024*1024)
	for i := 0; i < len(blob); i++ {
		blob[i] = byte('a' + i%26)
	}
	var blobHash fftypes.Bytes32 = sha256.Sum256(blob)
	t.Logf("Blob size=%d hash=%s", len(blob), &blobHash)
	var data fftypes.Data
	res, err := client.R().
		SetFormData(map[string]string{
			"autometa": "true",
			"metadata": `{"mymeta": "data"}`,
		}).
		SetFileReader("file", "myfile.txt", bytes.NewReader(blob)).
		SetResult(&data).
		Post(urlUploadData)
	if err != nil || res.IsError() {
		return res, err
	}
	assert.Equal(t, "data", data.Value.JSONObject().GetString("mymeta"))
	assert.Equal(t, "myfile.txt", data.Value.JSONObject().GetString("filename"))
	assert.Equal(t, float64(len(blob)), data.Value.JSONObject()["size"])
	assert.Equal(t, blobHash, *data.Blob.Hash)

	return client.R().
		SetBody(fftypes.MessageInput{
			InputData: fftypes.InputData{
				{DataRef: fftypes.DataRef{ID: data.ID}},
			},
		}).
		Post(urlBroadcastMessage)
}

func PrivateMessage(t *testing.T, client *resty.Client, data *fftypes.DataRefOrValue, orgNames []string) (*resty.Response, error) {
	members := make([]fftypes.MemberInput, len(orgNames))
	for i, oName := range orgNames {
		// We let FireFly resolve the friendly name of the org to the identity
		members[i] = fftypes.MemberInput{
			Identity: oName,
		}
	}
	return client.R().
		SetBody(fftypes.MessageInput{
			InputData: fftypes.InputData{data},
			Group: &fftypes.InputGroup{
				Members: members,
				Name:    fmt.Sprintf("test_%d", time.Now().Unix()),
			},
		}).
		Post(urlPrivateMessage)
}
