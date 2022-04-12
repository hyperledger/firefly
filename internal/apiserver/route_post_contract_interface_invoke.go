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

	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var postContractInterfaceInvoke = &oapispec.Route{
	Name:   "postContractInterfaceInvoke",
	Path:   "namespaces/{ns}/contracts/interfaces/{interfaceId}/invoke/{methodPath}",
	Method: http.MethodPost,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: coreconfig.NamespacesDefault, Description: coremsgs.APIParamsNamespace},
		{Name: "interfaceId", Description: coremsgs.APIParamsInterfaceID},
		{Name: "methodPath", Description: coremsgs.APIParamsMethodPath},
	},
	QueryParams: []*oapispec.QueryParam{
		{Name: "confirm", Description: coremsgs.APIConfirmQueryParam, IsBool: true, Example: "true"},
	},
	FilterFactory:   nil,
	Description:     coremsgs.APIEndpointsPostContractInterfaceInvoke,
	JSONInputValue:  func() interface{} { return &fftypes.ContractCallRequest{} },
	JSONOutputValue: func() interface{} { return &fftypes.ContractCallResponse{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		req := r.Input.(*fftypes.ContractCallRequest)
		req.Type = fftypes.CallTypeInvoke
		if req.Interface, err = fftypes.ParseUUID(r.Ctx, r.PP["interfaceId"]); err != nil {
			return nil, err
		}
		req.Method = &fftypes.FFIMethod{Pathname: r.PP["methodPath"]}
		return getOr(r.Ctx).Contracts().InvokeContract(r.Ctx, r.PP["ns"], req)
	},
}
