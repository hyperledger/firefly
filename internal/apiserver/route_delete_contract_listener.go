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

	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/oapispec"
)

var deleteContractListener = &oapispec.Route{
	Name:   "deleteContractListener",
	Path:   "contracts/listeners/{nameOrId}",
	Method: http.MethodDelete,
	PathParams: []*oapispec.PathParam{
		{Name: "nameOrId", Description: coremsgs.APIParamsContractListenerNameOrID},
	},
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     coremsgs.APIEndpointsDeleteContractListener,
	JSONInputValue:  nil,
	JSONOutputValue: nil,
	JSONOutputCodes: []int{http.StatusNoContent}, // Sync operation, no output
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		ns := extractNamespace(r.PP)
		err = getOr(r.Ctx, ns).Contracts().DeleteContractListenerByNameOrID(r.Ctx, ns, r.PP["nameOrId"])
		return nil, err
	},
}
