// Copyright © 2021 Kaleido, Inc.
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

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/persistence"
)

func getValues(values url.Values, key string) (results []string) {
	for queryName, queryValues := range values {
		// We choose to be case insensitive for our filters, so protocolId and protocolid can be used interchangably
		if strings.ToLower(queryName) == key {
			results = append(results, queryValues...)
		}
	}
	return results
}

func buildFilter(req *http.Request, ff persistence.FilterFactory) persistence.AndFilter {
	fb := ff.New(req.Context(), uint64(config.GetUint(config.APIDefaultFilterLimit)))
	possibleFields := fb.Fields()
	sort.Strings(possibleFields)
	filter := fb.And()
	_ = req.ParseForm()
	for _, field := range possibleFields {
		values := getValues(req.Form, field)
		if len(values) == 1 {
			filter.Condition(getCondition(fb, field, values[0]))
		} else if len(values) > 0 {
			sort.Strings(values)
			fs := make([]persistence.Filter, len(values))
			for i, value := range values {
				fs[i] = getCondition(fb, field, value)
			}
			filter.Condition(fb.Or(fs...))
		}
	}
	skipVals := getValues(req.Form, "skip")
	if len(skipVals) > 0 {
		s, _ := strconv.ParseUint(skipVals[0], 10, 64)
		filter.Skip(s)
	}
	limitVals := getValues(req.Form, "limit")
	if len(limitVals) > 0 {
		l, _ := strconv.ParseUint(limitVals[0], 10, 64)
		filter.Limit(l)
	}
	return filter
}

func getCondition(fb persistence.FilterBuilder, field, value string) persistence.Filter {
	if strings.HasPrefix(value, ">=") {
		return fb.Gte(field, value[2:])
	} else if strings.HasPrefix(value, "<=") {
		return fb.Lte(field, value[2:])
	} else if strings.HasPrefix(value, ">") {
		return fb.Gt(field, value[1:])
	} else if strings.HasPrefix(value, "<") {
		return fb.Lt(field, value[1:])
	} else if strings.HasPrefix(value, "@") {
		return fb.Contains(field, value[1:])
	} else if strings.HasPrefix(value, "^") {
		return fb.IContains(field, value[1:])
	} else if strings.HasPrefix(value, "!@") {
		return fb.NotContains(field, value[2:])
	} else if strings.HasPrefix(value, "!^") {
		return fb.NotIContains(field, value[2:])
	} else if strings.HasPrefix(value, "!") {
		return fb.Neq(field, value[1:])
	} else {
		return fb.Eq(field, value)
	}
}
