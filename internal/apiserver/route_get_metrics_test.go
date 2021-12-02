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
	"net/http/httptest"
	"testing"

	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetMetricsBadStartTime(t *testing.T) {
	_, r := newTestAPIServer()
	req := httptest.NewRequest("GET", "/api/v1/namespaces/mynamespace/metrics?collection=test&startTime=abc&endTime=456&buckets=30", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Result().StatusCode)
}

func TestGetMetricsBadEndTime(t *testing.T) {
	_, r := newTestAPIServer()
	req := httptest.NewRequest("GET", "/api/v1/namespaces/mynamespace/metrics?collection=test&startTime=123&endTime=abc&buckets=30", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Result().StatusCode)
}

func TestGetMetricsBadBuckets(t *testing.T) {
	_, r := newTestAPIServer()
	req := httptest.NewRequest("GET", "/api/v1/namespaces/mynamespace/metrics?collection=test&startTime=123&endTime=456&buckets=abc", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Result().StatusCode)
}

func TestGetMetricsSuccess(t *testing.T) {
	o, r := newTestAPIServer()
	req := httptest.NewRequest("GET", "/api/v1/namespaces/mynamespace/metrics?collection=test&startTime=123&endTime=456&buckets=30", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	o.On("GetMetrics", mock.Anything, "mynamespace", int64(123), int64(456), int64(30), database.CollectionName("test")).
		Return([]*fftypes.Metric{}, nil)
	r.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Result().StatusCode)
}
