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

	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/core"
)

var getContractInterfaceNameVersion = &oapispec.Route{
	Name:   "getContractInterfaceByNameAndVersion",
	Path:   "contracts/interfaces/{name}/{version}",
	Method: http.MethodGet,
	PathParams: []*oapispec.PathParam{
		{Name: "name", Description: coremsgs.APIParamsContractInterfaceName},
		{Name: "version", Description: coremsgs.APIParamsContractInterfaceVersion},
	},
	QueryParams: []*oapispec.QueryParam{
		{Name: "fetchchildren", Example: "true", Description: coremsgs.APIParamsContractInterfaceFetchChildren, IsBool: true},
	},
	FilterFactory:   nil,
	Description:     coremsgs.APIEndpointsGetContractInterfaceNameVersion,
	JSONInputValue:  nil,
	JSONOutputValue: func() interface{} { return &core.FFI{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		ns := extractNamespace(r.PP)
		if strings.EqualFold(r.QP["fetchchildren"], "true") {
			return getOr(r.Ctx, ns).Contracts().GetFFIWithChildren(r.Ctx, ns, r.PP["name"], r.PP["version"])
		}
		return getOr(r.Ctx, ns).Contracts().GetFFI(r.Ctx, ns, r.PP["name"], r.PP["version"])
	},
}
