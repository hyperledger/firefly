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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/oapispec"
)

var getContractAPISwagger = &oapispec.Route{
	Name:   "getContractAPISwagger",
	Path:   "namespaces/{ns}/apis/{apiName}/apispec",
	Method: http.MethodGet,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
		{Name: "apiName", Description: i18n.MsgTBD},
	},
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  nil,
	JSONInputMask:   nil,
	JSONOutputValue: func() interface{} { return &openapi3.T{} },
	JSONOutputSchema: func(ctx context.Context) string {
		return `{
			"type": "object",
			"properties": {
				"openapi": {
					"type": "string",
					"example": "3.0.2"
				},
				"info": {
					"type": "object",
					"properties": {
						"title": {
							"type": "string"
						},
						"description": {
							"type": "string"
						},
						"version": {
							"type": "string"
						}
					}
				}
			}
		}`
	},
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		return getOr(r.Ctx).Contracts().GetContractAPISwagger(r.Ctx, r.APIBaseURL, r.PP["ns"], r.PP["apiName"])
	},
}
