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

package cmd

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/hyperledger/firefly/mocks/apiservermocks"
	"github.com/hyperledger/firefly/mocks/namespacemocks"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const configDir = "../test/data/config"

func setupCmdUnitTest() (*namespacemocks.Manager, func()) {
	mockManager := &namespacemocks.Manager{}
	_utManager = mockManager
	BuildVersionOverride = "v" + apiVersion + "-unittest"
	return mockManager, func() { _utManager = nil }
}

func TestGetEngine(t *testing.T) {
	BuildVersionOverride = "v" + apiVersion + "-unittest"
	assert.NotNil(t, getRootManager())
}

func TestGetEngineBadVersion(t *testing.T) {
	BuildVersionOverride = "" // needs to be set to (devel) or match apiVersion
	assert.Panics(t, func() {
		getVersion()
	})
}

func TestExecMissingConfig(t *testing.T) {
	_, done := setupCmdUnitTest()
	defer done()
	defer viper.Reset()
	err := Execute()
	assert.Regexp(t, "Not Found", err)
}

func TestShowConfig(t *testing.T) {
	_, done := setupCmdUnitTest()
	defer done()
	viper.Reset()
	rootCmd.SetArgs([]string{"showconf"})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestExecEngineInitFail(t *testing.T) {
	utm, done := setupCmdUnitTest()
	defer done()
	utm.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("splutter"))
	os.Chdir(configDir)
	err := Execute()
	assert.Regexp(t, "splutter", err)
}

func TestExecEngineStartFail(t *testing.T) {
	utm, done := setupCmdUnitTest()
	defer done()
	utm.On("Init", mock.Anything, mock.Anything).Return(nil)
	utm.On("Start").Return(fmt.Errorf("bang"))
	os.Chdir(configDir)
	err := Execute()
	assert.Regexp(t, "bang", err)
}

func TestExecOkExitSIGINT(t *testing.T) {
	utm, done := setupCmdUnitTest()
	defer done()
	utm.On("Init", mock.Anything, mock.Anything).Return(fmt.Errorf("splutter"))
	utm.On("Init", mock.Anything, mock.Anything).Return(nil)
	utm.On("Start").Return(nil)
	utm.On("WaitStop").Return()

	os.Chdir(configDir)
	go func() {
		sigs <- syscall.SIGINT
	}()
	err := Execute()
	assert.NoError(t, err)
}

func TestExecOkRestartThenExit(t *testing.T) {
	utm, done := setupCmdUnitTest()
	defer done()
	var orContext context.Context
	initCount := 0
	init := utm.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	init.RunFn = func(a mock.Arguments) {
		orContext = a[0].(context.Context)
		cancelOrContext := a[1].(context.CancelFunc)
		initCount++
		if initCount == 2 {
			init.ReturnArguments = mock.Arguments{fmt.Errorf("second run")}
		}
		cancelOrContext()
	}
	utm.On("Start").Return(nil)
	ws := utm.On("WaitStop")
	ws.RunFn = func(a mock.Arguments) {
		<-orContext.Done()
	}

	os.Chdir(configDir)
	err := Execute()
	assert.EqualError(t, err, "second run")
}

func TestExecOkRestartConfigProblem(t *testing.T) {
	utm, done := setupCmdUnitTest()
	defer done()
	tmpDir, err := os.MkdirTemp(os.TempDir(), "ut")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	var orContext context.Context
	init := utm.On("Init", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	init.RunFn = func(a mock.Arguments) {
		orContext = a[0].(context.Context)
		cancelOrContext := a[1].(context.CancelFunc)
		cancelOrContext()
	}
	utm.On("Start").Return(nil)
	utm.On("WaitStop").Run(func(args mock.Arguments) {
		<-orContext.Done()
		os.Chdir(tmpDir) // this will mean we fail to read the config
	})

	os.Chdir(configDir)
	err = Execute()
	assert.Regexp(t, "Config File.*Not Found", err)
}

func TestAPIServerError(t *testing.T) {
	utm, done := setupCmdUnitTest()
	defer done()
	utm.On("Init", mock.Anything, mock.Anything).Return(nil)
	utm.On("Start").Return(nil)
	as := &apiservermocks.Server{}
	as.On("Serve", mock.Anything, utm).Return(fmt.Errorf("pop"))

	errChan := make(chan error)
	go startFirefly(context.Background(), func() {}, utm, as, errChan)
	err := <-errChan
	assert.EqualError(t, err, "pop")
}
