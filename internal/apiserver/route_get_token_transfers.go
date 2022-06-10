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
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

var getTokenTransfers = &oapispec.Route{
	Name:       "getTokenTransfers",
	Path:       "tokens/transfers",
	Method:     http.MethodGet,
	PathParams: nil,
	QueryParams: []*oapispec.QueryParam{
		{Name: "fromOrTo", Description: coremsgs.APIParamsTokenTransferFromOrTo},
	},
	FilterFactory:   database.TokenTransferQueryFactory,
	DescriptionKey:  coremsgs.APIEndpointsGetTokenTransfers,
	JSONInputValue:  nil,
	JSONOutputValue: func() interface{} { return []*core.TokenTransfer{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		filter := r.Filter
		if fromOrTo, ok := r.QP["fromOrTo"]; ok {
			fb := database.TokenTransferQueryFactory.NewFilter(r.Ctx)
			filter.Condition(
				fb.Or().
					Condition(fb.Eq("from", fromOrTo)).
					Condition(fb.Eq("to", fromOrTo)))
		}
		return filterResult(getOr(r.Ctx).Assets().GetTokenTransfers(r.Ctx, extractNamespace(r.PP), filter))
	},
}
