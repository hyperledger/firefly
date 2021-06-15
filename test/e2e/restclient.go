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
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/require"
)

var (
	urlGetNamespaces    = "/namespaces"
	urlGetMessages      = "/namespaces/default/messages"
	urlBroadcastMessage = "/namespaces/default/broadcast/message"
	urlPrivateMessage   = "/namespaces/default/send/message"
	urlGetData          = "/namespaces/default/data"
	urlGetOrganizations = "/network/organizations"
)

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
