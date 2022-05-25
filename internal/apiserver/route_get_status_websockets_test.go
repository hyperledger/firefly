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

	"github.com/hyperledger/firefly/mocks/eventmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestGetStatusWebSockets(t *testing.T) {
	o, r := newTestAPIServer()
	mem := &eventmocks.EventManager{}
	o.On("Events").Return(mem)
	req := httptest.NewRequest("GET", "/api/v1/status/websockets", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	mem.On("GetWebSocketStatus").Return(&core.WebSocketStatus{})
	r.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Result().StatusCode)
}
