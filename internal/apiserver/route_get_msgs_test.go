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
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetMessages(t *testing.T) {
	o, r := newTestAPIServer()
	req := httptest.NewRequest("GET", "/api/v1/namespaces/mynamespace/messages", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	o.On("GetMessages", mock.Anything, "mynamespace", mock.Anything).
		Return([]*fftypes.Message{}, nil, nil)
	r.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Result().StatusCode)
}

func TestGetMessagesWithCount(t *testing.T) {
	o, r := newTestAPIServer()
	req := httptest.NewRequest("GET", "/api/v1/namespaces/mynamespace/messages?count", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	var ten int64 = 10
	o.On("GetMessages", mock.Anything, "mynamespace", mock.Anything).
		Return([]*fftypes.Message{}, &database.FilterResult{
			Count: &ten,
		}, nil)
	r.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Result().StatusCode)
	var resWithCount filterResultsWithCount
	err := json.NewDecoder(res.Body).Decode(&resWithCount)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), resWithCount.Count)
}
