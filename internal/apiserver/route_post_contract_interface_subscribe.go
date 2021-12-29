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

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var postContractInterfaceSubscribe = &oapispec.Route{
	Name:   "postContractInterfaceSubscribe",
	Path:   "namespaces/{ns}/contracts/interfaces/{contractID}/subscribe/{eventPath}",
	Method: http.MethodPost,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
		{Name: "contractID", Example: "contractID", Description: i18n.MsgTBD},
		{Name: "eventPath", Example: "eventPath", Description: i18n.MsgTBD},
	},
	QueryParams: []*oapispec.QueryParam{
		{Name: "confirm", Description: i18n.MsgConfirmQueryParam, IsBool: true, Example: "true"},
	},
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.ContractSubscribeRequest{} },
	JSONInputMask:   nil,
	JSONOutputValue: func() interface{} { return &fftypes.ContractSubscription{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		contractSubscribeRequest := r.Input.(*fftypes.ContractSubscribeRequest)
		contractSubscribeRequest.ContractID = fftypes.MustParseUUID(r.PP["contractID"])
		return getOr(r.Ctx).Contracts().SubscribeContract(r.Ctx, r.PP["ns"], r.PP["eventPath"], contractSubscribeRequest)
	},
}
