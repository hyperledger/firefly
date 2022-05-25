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

var putContractAPI = &oapispec.Route{
	Name:   "putContractAPI",
	Path:   "apis/{id}",
	Method: http.MethodPut,
	PathParams: []*oapispec.PathParam{
		{Name: "id", Example: "id", Description: coremsgs.APIParamsContractAPIName},
	},
	QueryParams: []*oapispec.QueryParam{
		{Name: "confirm", Description: coremsgs.APIConfirmQueryParam, IsBool: true, Example: "true"},
	},
	FilterFactory:   nil,
	Description:     coremsgs.APIParamsContractAPIID,
	JSONInputValue:  func() interface{} { return &core.ContractAPI{} },
	JSONOutputValue: func() interface{} { return &core.ContractAPI{} },
	JSONOutputCodes: []int{http.StatusOK, http.StatusAccepted},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		waitConfirm := strings.EqualFold(r.QP["confirm"], "true")
		r.SuccessStatus = syncRetcode(waitConfirm)
		api := r.Input.(*core.ContractAPI)
		api.ID, err = fftypes.ParseUUID(r.Ctx, r.PP["id"])
		var res interface{}
		if err == nil {
			res, err = getOr(r.Ctx).Contracts().BroadcastContractAPI(r.Ctx, r.APIBaseURL, extractNamespace(r.PP), api, waitConfirm)
		}
		return res, err
	},
}
