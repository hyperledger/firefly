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

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/networkmap"
	"github.com/hyperledger/firefly/internal/oapispec"
)

var getIdentityDID = &oapispec.Route{
	Name:   "getIdentityDID",
	Path:   "namespaces/{ns}/identities/{idid}/did",
	Method: http.MethodGet,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
		{Name: "idid", Example: "id", Description: i18n.MsgTBD},
	},
	QueryParams:     nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  nil,
	JSONOutputValue: func() interface{} { return &networkmap.DIDDocument{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		return getOr(r.Ctx).NetworkMap().GetDIDDocForIndentityByID(r.Ctx, r.PP["ns"], r.PP["idid"])
	},
}
