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

package onchain

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/identitymocks"
	"github.com/hyperledger/firefly/pkg/identity"
	"github.com/stretchr/testify/assert"
)

var utConfPrefix = config.NewPluginConfig("onchain_unit_tests")

func TestInit(t *testing.T) {
	var oc identity.Plugin = &OnChain{}
	oc.InitPrefix(utConfPrefix)
	err := oc.Init(context.Background(), utConfPrefix, &identitymocks.Callbacks{})
	assert.NoError(t, err)
	assert.Equal(t, "onchain", oc.Name())
	err = oc.Start()
	assert.NoError(t, err)
	capabilities := oc.Capabilities()
	assert.NotNil(t, capabilities)
}

func TestResolve(t *testing.T) {
	var oc identity.Plugin = &OnChain{}
	err := oc.Init(context.Background(), utConfPrefix, &identitymocks.Callbacks{})
	assert.NoError(t, err)

	id, err := oc.Resolve(context.Background(), "0x12345")
	assert.NoError(t, err)
	assert.Equal(t, "0x12345", id.Identifier)
	assert.Equal(t, "0x12345", id.OnChain)
}
