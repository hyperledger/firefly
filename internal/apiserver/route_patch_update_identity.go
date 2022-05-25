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

var patchUpdateIdentity = &oapispec.Route{
	Name:   "patchUpdateIdentity",
	Path:   "identities/{iid}",
	Method: http.MethodPatch,
	PathParams: []*oapispec.PathParam{
		{Name: "iid", Description: coremsgs.APIParamsIdentityID},
	},
	QueryParams: []*oapispec.QueryParam{
		{Name: "confirm", Description: coremsgs.APIConfirmQueryParam, IsBool: true},
	},
	FilterFactory:   nil,
	Description:     coremsgs.APIEndpointsPatchUpdateIdentity,
	JSONInputValue:  func() interface{} { return &core.IdentityUpdateDTO{} },
	JSONOutputValue: func() interface{} { return &core.Identity{} },
	JSONOutputCodes: []int{http.StatusAccepted, http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		waitConfirm := strings.EqualFold(r.QP["confirm"], "true")
		r.SuccessStatus = syncRetcode(waitConfirm)
		org, err := getOr(r.Ctx).NetworkMap().UpdateIdentity(r.Ctx, extractNamespace(r.PP), r.PP["iid"], r.Input.(*core.IdentityUpdateDTO), waitConfirm)
		return org, err
	},
}
