// Copyright © 2022 Kaleido, Inc.
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

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var postOpRetry = &oapispec.Route{
	Name:   "postOpRetry",
	Path:   "namespaces/{ns}/operations/{opid}/retry",
	Method: http.MethodPost,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
		{Name: "opid", Description: i18n.MsgTBD},
	},
	QueryParams:     []*oapispec.QueryParam{},
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.EmptyInput{} },
	JSONInputMask:   nil,
	JSONInputSchema: func(ctx context.Context) string { return emptyObjectSchema },
	JSONOutputValue: func() interface{} { return &fftypes.Operation{} },
	JSONOutputCodes: []int{http.StatusAccepted},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		opid, err := fftypes.ParseUUID(r.Ctx, r.PP["opid"])
		if err != nil {
			return nil, err
		}
		return getOr(r.Ctx).Operations().RetryOperation(r.Ctx, r.PP["ns"], opid)
	},
}
