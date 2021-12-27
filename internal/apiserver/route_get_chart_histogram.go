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
	Path:   "namespaces/{ns}/charts/histogram/{collection}",
	Method: http.MethodGet,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
		{Name: "collection", Description: i18n.MsgTBD},
	},
	QueryParams: []*oapispec.QueryParam{
		{Name: "startTime", Description: i18n.MsgHistogramStartTimeParam, IsBool: false},
		{Name: "endTime", Description: i18n.MsgHistogramEndTimeParam, IsBool: false},
		{Name: "buckets", Description: i18n.MsgHistogramBucketsParam, IsBool: false},
	},
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  nil,
	JSONOutputValue: func() interface{} { return []*fftypes.ChartHistogram{} },
	JSONOutputCodes: []int{http.StatusOK},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		startTime, err := fftypes.ParseString(r.QP["startTime"])
		if err != nil {
			return nil, i18n.NewError(r.Ctx, i18n.MsgInvalidChartNumberParam, "startTime")
		}
		endTime, err := fftypes.ParseString(r.QP["endTime"])
		if err != nil {
			return nil, i18n.NewError(r.Ctx, i18n.MsgInvalidChartNumberParam, "endTime")
		}
		buckets, err := strconv.ParseInt(r.QP["buckets"], 10, 64)
		if err != nil {
			return nil, i18n.NewError(r.Ctx, i18n.MsgInvalidChartNumberParam, "buckets")
		}
		return getOr(r.Ctx).GetChartHistogram(r.Ctx, r.PP["ns"], startTime.UnixNano(), endTime.UnixNano(), buckets, database.CollectionName(r.PP["collection"]))
	},
}
