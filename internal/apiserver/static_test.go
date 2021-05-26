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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticHosting(t *testing.T) {
	sc := newStaticHandler(configDir, "index.html", "/config")
	var handler http.HandlerFunc = sc.ServeHTTP
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/config/firefly.core.yaml", nil)
	handler(res, req)
	assert.Equal(t, 200, res.Result().StatusCode)
	b, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.NotEmpty(t, b)

}

func Test404UnRelablePath(t *testing.T) {
	sc := newStaticHandler("test", "firefly.core.yaml", "test")
	var handler http.HandlerFunc = sc.ServeHTTP
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	handler(res, req)
	assert.Equal(t, 404, res.Result().StatusCode)
}

func TestServeDefault(t *testing.T) {
	sc := newStaticHandler(configDir, "firefly.core.yaml", "/config")
	var handler http.HandlerFunc = sc.ServeHTTP
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/config", nil)
	handler(res, req)
	assert.Equal(t, 200, res.Result().StatusCode)
	b, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.NotEmpty(t, b)
}

func TestServeNotFoundServeDefault(t *testing.T) {
	sc := newStaticHandler(configDir, "firefly.core.yaml", "/config")
	var handler http.HandlerFunc = sc.ServeHTTP
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/config/wrong", nil)
	handler(res, req)
	assert.Equal(t, 200, res.Result().StatusCode)
}

type fakeOS struct{}

func (f *fakeOS) Stat(name string) (os.FileInfo, error) {
	return nil, fmt.Errorf("pop")
}

func TestStatError(t *testing.T) {

	sc := &staticHandler{
		staticPath: "../../test",
		urlPrefix:  "/test",
		indexPath:  "firefly.core.yaml",
		os:         &fakeOS{},
	}
	var handler http.HandlerFunc = sc.ServeHTTP
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test/config", nil)
	handler(res, req)
	assert.Equal(t, 500, res.Result().StatusCode)
	b, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Regexp(t, "FF10185", string(b))
}
