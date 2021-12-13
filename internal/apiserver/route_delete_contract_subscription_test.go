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

	"github.com/hyperledger/firefly/mocks/contractmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteContractSubscriptionByID(t *testing.T) {
	o, r := newTestAPIServer()
	mcm := &contractmocks.Manager{}
	o.On("Contracts").Return(mcm)
	id := fftypes.NewUUID()
	req := httptest.NewRequest("DELETE", "/api/v1/namespaces/mynamespace/contracts/subscriptions/"+id.String(), nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	mcm.On("DeleteContractSubscriptionByID", mock.Anything, id).
		Return(nil, nil)
	r.ServeHTTP(res, req)

	assert.Equal(t, 204, res.Result().StatusCode)
}

func TestDeleteContractSubscriptionBadID(t *testing.T) {
	_, r := newTestAPIServer()
	req := httptest.NewRequest("DELETE", "/api/v1/namespaces/mynamespace/contracts/subscriptions/bad", nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Result().StatusCode)
}
