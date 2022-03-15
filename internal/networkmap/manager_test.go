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

package networkmap

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/stretchr/testify/assert"
)

func newTestNetworkmap(t *testing.T) (*networkMap, func()) {
	config.Reset()
	ctx, cancel := context.WithCancel(context.Background())
	mdi := &databasemocks.Plugin{}
	mbm := &broadcastmocks.Manager{}
	mdx := &dataexchangemocks.Plugin{}
	mim := &identitymanagermocks.Manager{}
	msa := &syncasyncmocks.Bridge{}
	nm, err := NewNetworkMap(ctx, mdi, mbm, mdx, mim, msa)
	assert.NoError(t, err)
	return nm.(*networkMap), cancel

}

func TestNewNetworkMapMissingDep(t *testing.T) {
	_, err := NewNetworkMap(context.Background(), nil, nil, nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}
