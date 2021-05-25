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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFavIcon16(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/favicon-16x16.png", nil)
	var handler http.HandlerFunc = favIcons
	handler(res, req)
	assert.Equal(t, 200, res.Result().StatusCode)
	b, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Equal(t, b, ffLogo16)
}

func TestFavIcon32(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/favicon-32x32.png", nil)
	var handler http.HandlerFunc = favIcons
	handler(res, req)
	assert.Equal(t, 200, res.Result().StatusCode)
	b, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Equal(t, b, ffLogo32)
}
