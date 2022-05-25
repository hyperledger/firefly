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

var getEvents = &oapispec.Route{
	Name:       "getEvents",
	Path:       "events",
	Method:     http.MethodGet,
	PathParams: nil,
	QueryParams: []*oapispec.QueryParam{
		{Name: "fetchreferences", Example: "true", Description: coremsgs.APIParamsFetchReferences, IsBool: true},
	},
	FilterFactory:   database.EventQueryFactory,
	Description:     coremsgs.APIEndpointsGetEvents,
	JSONInputValue:  nil,
	JSONOutputValue: func() interface{} { return []*core.Event{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		if strings.EqualFold(r.QP["fetchreferences"], "true") {
			return filterResult(getOr(r.Ctx).GetEventsWithReferences(r.Ctx, extractNamespace(r.PP), r.Filter))
		}
		return filterResult(getOr(r.Ctx).GetEvents(r.Ctx, extractNamespace(r.PP), r.Filter))
	},
}
