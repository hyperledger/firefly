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
	"net/http"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/core"
)

var getContractInterface = &oapispec.Route{
	Name:   "getContractInterface",
	Path:   "contracts/interfaces/{interfaceId}",
	Method: http.MethodGet,
	PathParams: []*oapispec.PathParam{
		{Name: "interfaceId", Description: coremsgs.APIParamsContractInterfaceID},
	},
	QueryParams: []*oapispec.QueryParam{
		{Name: "fetchchildren", Example: "true", Description: coremsgs.APIParamsContractInterfaceFetchChildren, IsBool: true},
	},
	FilterFactory:   nil,
	Description:     coremsgs.APIEndpointsGetContractInterface,
	JSONInputValue:  nil,
	JSONOutputValue: func() interface{} { return &core.FFI{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		interfaceID, err := fftypes.ParseUUID(r.Ctx, r.PP["interfaceId"])
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(r.QP["fetchchildren"], "true") {
			return getOr(r.Ctx).Contracts().GetFFIByIDWithChildren(r.Ctx, interfaceID)
		}
		return getOr(r.Ctx).Contracts().GetFFIByID(r.Ctx, interfaceID)
	},
}
