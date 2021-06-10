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
	"github.com/go-resty/resty/v2"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var (
	urlGetNamespaces    = "/namespaces"
	urlGetMessages      = "/namespaces/default/messages"
	urlBroadcastMessage = "/namespaces/default/broadcast/message"
	urlGetData          = "/namespaces/default/data"
)

func GetNamespaces(client *resty.Client) (*resty.Response, error) {
	return client.R().
		SetResult(&[]fftypes.Namespace{}).
		Get(urlGetNamespaces)
}

func GetMessages(client *resty.Client) (*resty.Response, error) {
	return client.R().
		SetResult(&[]fftypes.Message{}).
		Get(urlGetMessages)
}

func BroadcastMessage(client *resty.Client, data *fftypes.DataRefOrValue) (*resty.Response, error) {
	return client.R().
		SetBody(fftypes.MessageInput{
			InputData: fftypes.InputData{data},
		}).
		Post(urlBroadcastMessage)
}

func GetData(client *resty.Client) (*resty.Response, error) {
	return client.R().
		SetResult(&[]fftypes.Data{}).
		Get(urlGetData)
}
