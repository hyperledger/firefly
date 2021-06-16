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
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/hyperledger-labs/firefly/mocks/orchestratormocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPostResetConfig(t *testing.T) {
	o := &orchestratormocks.Orchestrator{}
	as := &apiServer{}
	r := as.createAdminMuxRouter(o)
	req := httptest.NewRequest("POST", "/admin/api/v1/config/reset", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	o.On("ResetConfig", mock.Anything).Return()
	r.ServeHTTP(res, req)

	assert.Equal(t, 204, res.Result().StatusCode)
}
