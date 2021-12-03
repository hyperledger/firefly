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
	"strconv"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var getChartHistogram = &oapispec.Route{
	Name:   "getChartHistogram",
	Path:   "namespaces/{ns}/charts/histogram",
	Method: http.MethodGet,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
	},
	QueryParams: []*oapispec.QueryParam{
		{Name: "startTime", Description: i18n.MsgMetricStartTimeParam, IsBool: false},
		{Name: "endTime", Description: i18n.MsgMetricEndTimeParam, IsBool: false},
		{Name: "buckets", Description: i18n.MsgMetricBucketsParam, IsBool: false},
		{Name: "collection", Description: i18n.MsgMetricCollectionParam, IsBool: false},
	},
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  nil,
	JSONOutputValue: func() interface{} { return []*fftypes.ChartHistogram{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		startTime, err := strconv.ParseInt(r.QP["startTime"], 10, 64)
		if err != nil {
			return nil, i18n.NewError(r.Ctx, i18n.MsgInvalidMetricParam, "startTime")
		}
		endTime, err := strconv.ParseInt(r.QP["endTime"], 10, 64)
		if err != nil {
			return nil, i18n.NewError(r.Ctx, i18n.MsgInvalidMetricParam, "endTime")
		}
		buckets, err := strconv.ParseInt(r.QP["buckets"], 10, 64)
		if err != nil {
			return nil, i18n.NewError(r.Ctx, i18n.MsgInvalidMetricParam, "buckets")
		}
		return r.Or.GetChartHistogram(r.Ctx, r.PP["ns"], startTime, endTime, buckets, database.CollectionName(r.QP["collection"]))
	},
}
