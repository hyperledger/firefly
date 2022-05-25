// Copyright © 2022 Kaleido, Inc.
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

//go:build reference
// +build reference

package apiserver

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/stretchr/testify/assert"
)

func TestDownloadSwaggerYAML(t *testing.T) {
	config.Set(coreconfig.APIOASPanicOnMissingDescription, true)
	as := &apiServer{}
	handler := as.apiWrapper(as.swaggerHandler(as.swaggerGenerator(routes, "http://localhost:5000")))
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	res, err := http.Get(fmt.Sprintf("http://%s/api/swagger.yaml", s.Listener.Addr()))
	assert.NoError(t, err)
	b, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, 200, res.StatusCode, string(b))
	doc, err := openapi3.NewLoader().LoadFromData(b)
	assert.NoError(t, err)
	err = doc.Validate(context.Background())
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join("..", "..", "docs", "swagger", "swagger.yaml"), b, 0644)
	assert.NoError(t, err)
}
