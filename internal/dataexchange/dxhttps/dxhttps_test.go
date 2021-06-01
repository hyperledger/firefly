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

package dxhttps

import (
	"context"
	"testing"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/mocks/dataexchangemocks"
	"github.com/kaleido-io/firefly/pkg/dataexchange"
	"github.com/stretchr/testify/assert"
)

var utConfPrefix = config.NewPluginConfig("dxhttps_unit_tests")

func TestInit(t *testing.T) {
	var oc dataexchange.Plugin = &HTTPS{}
	oc.InitPrefix(utConfPrefix)
	err := oc.Init(context.Background(), utConfPrefix, &dataexchangemocks.Callbacks{})
	assert.NoError(t, err)
	assert.Equal(t, "https", oc.Name())
	err = oc.Start()
	assert.NoError(t, err)
	capabilities := oc.Capabilities()
	assert.NotNil(t, capabilities)
}

func TestGetEndpointInfo(t *testing.T) {
	var h dataexchange.Plugin = &HTTPS{}
	endpoint, err := h.GetEndpointInfo(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, endpoint)
}
