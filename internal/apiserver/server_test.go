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
	"context"
	"testing"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestStartStopServer(t *testing.T) {
	config.Reset()
	config.Set(config.HttpPort, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // server will immediately shut down
	err := Serve(ctx)
	assert.NoError(t, err)
}

func TestInvalidListener(t *testing.T) {
	config.Reset()
	config.Set(config.HttpAddress, "...")
	_, err := createListener(context.Background())
	assert.Error(t, err)
}

func TestTLSServer(t *testing.T) {
	config.Reset()
	config.Set(config.HttpAddress, "...")
	_, err := createListener(context.Background())
	assert.Error(t, err)
}

// func TestJSONHTTPServe200E2E(t *testing.T) {

// 	origRoutes := routes
// 	defer func() { routes = origRoutes }()

// 	config.Reset()
// 	routes = []Route{
// 		{
// 			Name:            "testRoute",
// 			Path:            "/test",
// 			Method:          "POST",
// 			JSONInputValue:  func() interface{} { var input map[string]interface{}; return input },
// 			JSONOutputValue: func() interface{} { var output map[string]interface{}; return output },
// 			JSONHandler: func(req *http.Request, input interface{}, output interface{}) (status int, err error) {
// 				assert.Equal(t, "value1", input.(map[string]interface{})["input1"])
// 				output.(map[string]interface{})["output1"] = "value2"
// 				return 200, nil
// 			},
// 		},
// 	}
// 	config.Set(config.HttpPort, 0)

// }
