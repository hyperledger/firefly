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

package apiserver

import (
	"net/http"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/oapispec"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var privateSendSchema = `{
	"properties": {
		 "data": {
				"items": {
					 "properties": {
							"validator": {"type": "string"},
							"datatype": {
								"type": "object",
								"properties": {
									"name": {"type": "string"},
									"version": {"type": "string"}
								}
							},
							"value": {
								"type": "object"
							}
					 },
					 "type": "object"
				},
				"type": "array"
		 },
		 "group": {
				"properties": {
					"name": {
						"type": "string"
					},
					"members": {
						"type": "array",
						"items": {
							"properties": {
								"identity": {
									"type": "string"
								},
								"node": {
									"type": "string"
								}
							},
							"required": ["identity"],
							"type": "object"
						}
					}
			},
			"required": ["members"],
			"type": "object"
		 },
		 "header": {
				"properties": {
					 "author": {
							"type": "string"
					 },
					 "cid": {},
					 "context": {
							"type": "string"
					 },
					 "group": {},
					 "topic": {
							"type": "string"
					 },
					 "tx": {
							"properties": {
								 "type": {
										"type": "string",
										"default": "pin"
								 }
							},
							"type": "object"
					 }
				},
				"type": "object"
		 }
	},
	"type": "object"
}`

var postSendMessage = &oapispec.Route{
	Name:   "postSendMessage",
	Path:   "namespaces/{ns}/send/message",
	Method: http.MethodPost,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
	},
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.MessageInput{} },
	JSONInputSchema: privateSendSchema,
	JSONOutputValue: func() interface{} { return &fftypes.Message{} },
	JSONOutputCode:  http.StatusAccepted, // Async operation
	JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
		output, err = r.Or.PrivateMessaging().SendMessage(r.Ctx, r.PP["ns"], r.Input.(*fftypes.MessageInput))
		return output, err
	},
}
