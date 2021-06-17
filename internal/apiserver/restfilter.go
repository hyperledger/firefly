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
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
)

func (as *apiServer) getValues(values url.Values, key string) (results []string) {
	for queryName, queryValues := range values {
		// We choose to be case insensitive for our filters, so protocolID and protocolid can be used interchangeably
		if strings.EqualFold(queryName, key) {
			results = append(results, queryValues...)
		}
	}
	return results
}

func (as *apiServer) buildFilter(req *http.Request, ff database.QueryFactory) (database.AndFilter, error) {
	ctx := req.Context()
	log.L(ctx).Debugf("Query: %s", req.URL.RawQuery)
	fb := ff.NewFilterLimit(ctx, as.defaultFilterLimit)
	possibleFields := fb.Fields()
	sort.Strings(possibleFields)
	filter := fb.And()
	_ = req.ParseForm()
	for _, field := range possibleFields {
		values := as.getValues(req.Form, field)
		if len(values) == 1 {
			filter.Condition(as.getCondition(fb, field, values[0]))
		} else if len(values) > 0 {
			sort.Strings(values)
			fs := make([]database.Filter, len(values))
			for i, value := range values {
				fs[i] = as.getCondition(fb, field, value)
			}
			filter.Condition(fb.Or(fs...))
		}
	}
	skipVals := as.getValues(req.Form, "skip")
	if len(skipVals) > 0 {
		s, _ := strconv.ParseUint(skipVals[0], 10, 64)
		if as.maxFilterSkip != 0 && s > as.maxFilterSkip {
			return nil, i18n.NewError(req.Context(), i18n.MsgMaxFilterSkip, as.maxFilterSkip)
		}
		filter.Skip(s)
	}
	limitVals := as.getValues(req.Form, "limit")
	if len(limitVals) > 0 {
		l, _ := strconv.ParseUint(limitVals[0], 10, 64)
		if as.maxFilterLimit != 0 && l > as.maxFilterLimit {
			return nil, i18n.NewError(req.Context(), i18n.MsgMaxFilterLimit, as.maxFilterLimit)
		}
		filter.Limit(l)
	}
	sortVals := as.getValues(req.Form, "sort")
	for _, sv := range sortVals {
		subSortVals := strings.Split(sv, ",")
		for _, ssv := range subSortVals {
			ssv = strings.TrimSpace(ssv)
			if ssv != "" {
				filter.Sort(ssv)
			}
		}
	}
	descendingVals := as.getValues(req.Form, "descending")
	ascendingVals := as.getValues(req.Form, "ascending")
	if len(descendingVals) > 0 && (descendingVals[0] == "" || strings.EqualFold(descendingVals[0], "true")) {
		filter.Descending()
	} else if len(ascendingVals) > 0 && (ascendingVals[0] == "" || strings.EqualFold(ascendingVals[0], "true")) {
		filter.Ascending()
	}
	return filter, nil
}

func (as *apiServer) getCondition(fb database.FilterBuilder, field, value string) database.Filter {
	switch {
	case strings.HasPrefix(value, ">="):
		return fb.Gte(field, value[2:])
	case strings.HasPrefix(value, "<="):
		return fb.Lte(field, value[2:])
	case strings.HasPrefix(value, ">"):
		return fb.Gt(field, value[1:])
	case strings.HasPrefix(value, "<"):
		return fb.Lt(field, value[1:])
	case strings.HasPrefix(value, "@"):
		return fb.Contains(field, value[1:])
	case strings.HasPrefix(value, "^"):
		return fb.IContains(field, value[1:])
	case strings.HasPrefix(value, "!@"):
		return fb.NotContains(field, value[2:])
	case strings.HasPrefix(value, "!^"):
		return fb.NotIContains(field, value[2:])
	case strings.HasPrefix(value, "!"):
		return fb.Neq(field, value[1:])
	default:
		return fb.Eq(field, value)
	}
}
