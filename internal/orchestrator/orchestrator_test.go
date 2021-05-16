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
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/batchingmocks"
	"github.com/kaleido-io/firefly/mocks/blockchainmocks"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestInitPublicStoragePluginFail(t *testing.T) {
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
	o := NewOrchestrator().(*orchestrator)
	dbm := &databasemocks.Plugin{}
	o.database = dbm
	dbm.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	dbm.On("UpsertNamespace", mock.Anything, mock.Anything).Return(nil)
	err := config.ReadConfig("../../test/config/firefly.core.yaml")
	assert.NoError(t, err)
	err = o.Init(context.Background())
	assert.NoError(t, err)
	o.Close()
}

func TestInitBadIdentity(t *testing.T) {
	o := NewOrchestrator()
	err := config.ReadConfig("../../test/config/firefly.core.yaml")
	config.Set(config.NodeIdentity, "!!!!wrongun")
	assert.NoError(t, err)
	err = o.Init(context.Background())
	assert.Regexp(t, "FF10131", err.Error())
	o.Close()
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

func TestInitNamespacesBadName(t *testing.T) {
	config.Reset()
	config.Set(config.NamespacesPredefined, fftypes.JSONObjectArray{
		{"name": "!Badness"},
	})
	o := &orchestrator{}
	err := o.initNamespaces(context.Background())
	assert.Regexp(t, "FF10131", err.Error())
}

func TestInitNamespacesGetFail(t *testing.T) {
	config.Reset()
	mdb := &databasemocks.Plugin{}
	o := &orchestrator{
		database: mdb,
	}
	mdb.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := o.initNamespaces(context.Background())
	assert.Regexp(t, "pop", err.Error())
}

func TestInitNamespacesUpsertFail(t *testing.T) {
	config.Reset()
	mdb := &databasemocks.Plugin{}
	o := &orchestrator{
		database: mdb,
	}
	mdb.On("GetNamespace", mock.Anything, mock.Anything).Return(nil, nil)
	mdb.On("UpsertNamespace", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	err := o.initNamespaces(context.Background())
	assert.Regexp(t, "pop", err.Error())
}

func TestInitNamespacesUpsertNotNeeded(t *testing.T) {
	config.Reset()
	mdb := &databasemocks.Plugin{}
	o := &orchestrator{
		database: mdb,
	}
	mdb.On("GetNamespace", mock.Anything, mock.Anything).Return(&fftypes.Namespace{
		Type: fftypes.NamespaceTypeStaticBroadcast, // any broadcasted NS will not be updated
	}, nil)
	err := o.initNamespaces(context.Background())
	assert.NoError(t, err)
}

func TestInitNamespacesDefaultMissing(t *testing.T) {
	config.Reset()
	mdb := &databasemocks.Plugin{}
	o := &orchestrator{
		database: mdb,
	}
	config.Set(config.NamespacesPredefined, fftypes.JSONObjectArray{})
	err := o.initNamespaces(context.Background())
	assert.Regexp(t, "FF10166", err.Error())
}
