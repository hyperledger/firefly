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
	"context"
	"net/http"
	"strings"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/oapispec"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var broadcastSchema = `{
	"properties": {
		 "data": {
				"items": {
					 "properties": {
							"id": {"type": "string"},
							"hash": {"type": "string"},
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
					 "tag": {
							"type": "string"
					 },
					 "topics": {
						 	"items": {
								 "type": "string"
							 }
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

var postNewMessageBroadcast = &oapispec.Route{
	Name:   "postNewMessageBroadcast",
	Path:   "namespaces/{ns}/messages/broadcast",
	Method: http.MethodPost,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
	},
	QueryParams: []*oapispec.QueryParam{
		{Name: "confirm", Description: i18n.MsgConfirmQueryParam, IsBool: true},
	},
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.MessageInOut{} },
	JSONInputSchema: func(ctx context.Context) string { return broadcastSchema },
	JSONOutputValue: func() interface{} { return &fftypes.Message{} },
	JSONOutputCodes: []int{http.StatusAccepted, http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		waitConfirm := strings.EqualFold(r.QP["confirm"], "true")
		r.SuccessStatus = syncRetcode(waitConfirm)
		output, err = r.Or.Broadcast().BroadcastMessage(r.Ctx, r.PP["ns"], r.Input.(*fftypes.MessageInOut), waitConfirm)
		return output, err
	},
}
