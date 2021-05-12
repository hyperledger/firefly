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

package cmd

import (
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/kaleido-io/firefly/mocks/enginemocks"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetEngine(t *testing.T) {
	assert.NotNil(t, getEngine())
}

func TestExecMissingConfig(t *testing.T) {
	_utEngine = &enginemocks.Engine{}
	defer func() { _utEngine = nil }()
	viper.Reset()
	err := Execute()
	assert.Regexp(t, "Not Found", err.Error())
}

func TestShowConfig(t *testing.T) {
	_utEngine = &enginemocks.Engine{}
	defer func() { _utEngine = nil }()
	viper.Reset()
	rootCmd.SetArgs([]string{"showconf"})
	defer rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestExecEngineInitFail(t *testing.T) {
	me := &enginemocks.Engine{}
	me.On("Init", mock.Anything).Return(fmt.Errorf("splutter"))
	_utEngine = me
	defer func() { _utEngine = nil }()
	os.Chdir("../test/config")
	err := Execute()
	assert.Regexp(t, "splutter", err.Error())
}

func TestExecEngineStartFail(t *testing.T) {
	me := &enginemocks.Engine{}
	me.On("Init", mock.Anything).Return(nil)
	me.On("Start").Return(fmt.Errorf("bang"))
	_utEngine = me
	defer func() { _utEngine = nil }()
	os.Chdir("../test/config")
	err := Execute()
	assert.Regexp(t, "bang", err.Error())
}

func TestExecOkExitSIGINT(t *testing.T) {
	me := &enginemocks.Engine{}
	me.On("Init", mock.Anything).Return(nil)
	me.On("Start").Return(nil)
	_utEngine = me
	defer func() { _utEngine = nil }()

	os.Chdir("../test/config")
	go func() {
		sigs <- syscall.SIGINT
	}()
	err := Execute()
	assert.NoError(t, err)
}
