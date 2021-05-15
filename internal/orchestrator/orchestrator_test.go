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

package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/mocks/batchingmocks"
	"github.com/kaleido-io/firefly/mocks/blockchainmocks"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
)

func TestInitDatabasePluginFail(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	err := o.Init(context.Background())
	assert.Regexp(t, "FF10122", err.Error())
}

func TestInitBlockchainPluginFail(t *testing.T) {
	o := &orchestrator{
		database: &databasemocks.Plugin{},
	}
	err := o.Init(context.Background())
	assert.Regexp(t, "FF10110", err.Error())
}

func TestInitP2PFilesystemPluginFail(t *testing.T) {
	o := &orchestrator{
		database:   &databasemocks.Plugin{},
		blockchain: &blockchainmocks.Plugin{},
	}
	err := o.Init(context.Background())
	assert.Regexp(t, "FF10134", err.Error())
}

func TestInitBatchComponentFail(t *testing.T) {
	o := &orchestrator{}
	err := o.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err.Error())
}

func TestInitBroadcastComponentFail(t *testing.T) {
	o := &orchestrator{
		batch: &batchingmocks.BatchManager{},
	}
	err := o.initComponents(context.Background())
	assert.Regexp(t, "FF10128", err.Error())
}

func TestInitOK(t *testing.T) {
	e := NewOrchestrator()
	err := config.ReadConfig("../../test/config/firefly.core.yaml")
	assert.NoError(t, err)
	err = e.Init(context.Background())
	assert.NoError(t, err)
	e.Close()
}

func TestInitBadIdentity(t *testing.T) {
	e := NewOrchestrator()
	err := config.ReadConfig("../../test/config/firefly.core.yaml")
	config.Set(config.NodeIdentity, "!!!!wrongun")
	assert.NoError(t, err)
	err = e.Init(context.Background())
	assert.Regexp(t, "FF10131", err.Error())
	e.Close()
}

func TestStartBatchFail(t *testing.T) {
	config.Reset()
	mb := &batchingmocks.BatchManager{}
	mblk := &blockchainmocks.Plugin{}
	o := &orchestrator{
		batch:      mb,
		blockchain: mblk,
	}
	mb.On("Start").Return(fmt.Errorf("pop"))
	mblk.On("Start").Return(nil)
	err := o.Start()
	assert.Regexp(t, "pop", err.Error())
}

func TestStartBlockchainFail(t *testing.T) {
	config.Reset()
	mblk := &blockchainmocks.Plugin{}
	o := &orchestrator{
		blockchain: mblk,
	}
	mblk.On("Start").Return(fmt.Errorf("pop"))
	err := o.Start()
	assert.Regexp(t, "pop", err.Error())
}
