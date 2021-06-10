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
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pollForUp(t *testing.T, client *resty.Client) {
	var resp *resty.Response
	var err error
	for i := 0; i < 3; i++ {
		resp, err = GetNamespaces(client)
		if err == nil && resp.StatusCode() == 200 {
			break
		}
		time.Sleep(5 * time.Second)
	}
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode())
}

func TestEndToEnd(t *testing.T) {
	stackFile := os.Getenv("STACK_FILE")
	if stackFile == "" {
		t.Fatal("STACK_FILE must be set")
	}

	port1, err := GetMemberPort(stackFile, 0)
	require.NoError(t, err)
	port2, err := GetMemberPort(stackFile, 1)
	require.NoError(t, err)

	client1 := resty.New()
	client1.SetHostURL(fmt.Sprintf("http://localhost:%d/api/v1", port1))
	client2 := resty.New()
	client2.SetHostURL(fmt.Sprintf("http://localhost:%d/api/v1", port2))

	t.Logf("Client 1: " + client1.HostURL)
	t.Logf("Client 2: " + client2.HostURL)
	pollForUp(t, client1)
	pollForUp(t, client2)

	var resp *resty.Response
	definitionName := "definition1"

	resp, err = BroadcastDatatype(client1, definitionName)
	require.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode())

	resp, err = GetData(client1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode())
	data := resp.Result().(*[]fftypes.Data)
	assert.Equal(t, 1, len(*data))
	assert.Equal(t, "default", (*data)[0].Namespace)
	assert.Equal(t, fftypes.ValidatorType("datadef"), (*data)[0].Validator)
	assert.Equal(t, definitionName, (*data)[0].Value.JSONObject().GetString("name"))

	resp, err = GetData(client2)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode())
	data = resp.Result().(*[]fftypes.Data)
	t.Logf("Returned results from member 2: %d", len(*data))
}
