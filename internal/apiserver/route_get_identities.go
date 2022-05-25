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
	"github.com/hyperledger/firefly/pkg/database"
)

var getIdentities = &oapispec.Route{
	Name:       "getIdentities",
	Path:       "identities",
	Method:     http.MethodGet,
	PathParams: nil,
	QueryParams: []*oapispec.QueryParam{
		{Name: "fetchverifiers", Example: "true", Description: coremsgs.APIParamsFetchVerifiers, IsBool: true},
	},
	FilterFactory:   database.IdentityQueryFactory,
	Description:     coremsgs.APIEndpointsGetIdentities,
	JSONInputValue:  nil,
	JSONOutputValue: func() interface{} { return &[]*core.IdentityWithVerifiers{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		if strings.EqualFold(r.QP["fetchverifiers"], "true") {
			return filterResult(getOr(r.Ctx).NetworkMap().GetIdentitiesWithVerifiers(r.Ctx, extractNamespace(r.PP), r.Filter))
		}
		return filterResult(getOr(r.Ctx).NetworkMap().GetIdentities(r.Ctx, extractNamespace(r.PP), r.Filter))
	},
}
