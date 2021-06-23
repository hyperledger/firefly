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

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/oapispec"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

const anyJSONSchema = `{
	"anyOf": [
		{
			"type": "string"
		},
		{
			"type": "number"
		},
		{
			"type": "object",
			"additionalProperties": true
		},
		{
			"type": "array",
			"items": {
				"type": "object",
				"additionalProperties": true
			}
		}
	]
}`

var putConfigRecord = &oapispec.Route{
	Name:   "putConfigRecord",
	Path:   "config/records/{key}",
	Method: http.MethodPut,
	PathParams: []*oapispec.PathParam{
		{Name: "key", Example: "database", Description: i18n.MsgTBD},
	},
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.Byteable{} },
	JSONOutputValue: nil,
	JSONOutputCode:  http.StatusOK,
	JSONInputSchema: func(ctx context.Context) string { return anyJSONSchema },
	JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
		output, err = r.Or.PutConfigRecord(r.Ctx, r.PP["key"], *r.Input.(*fftypes.Byteable))
		return output, err
	},
}
