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

package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetConfigRecord(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecord", mock.Anything, mock.Anything).Return(&fftypes.ConfigRecord{
		Key:   "foobar",
		Value: fftypes.JSONAnyPtr(`{"foo": "bar"}`),
	}, nil)
	ctx := context.Background()
	configRecord, err := or.GetConfigRecord(ctx, "foo")
	assert.NoError(t, err)
	assert.Equal(t, fftypes.JSONAnyPtr(`{"foo": "bar"}`), configRecord.Value)
}

func TestGetConfigRecords(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("GetConfigRecords", mock.Anything, mock.Anything).Return([]*fftypes.ConfigRecord{
		{
			Key:   "foobar",
			Value: fftypes.JSONAnyPtr(`{"foo": "bar"}`),
		},
	}, nil, nil)
	ctx := context.Background()
	configRecords, _, err := or.GetConfigRecords(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, fftypes.JSONAnyPtr(`{"foo": "bar"}`), configRecords[0].Value)
}

func TestPutConfigRecord(t *testing.T) {
	testValue := fftypes.JSONAnyPtr(`{"foo": "bar"}`)
	or := newTestOrchestrator()
	or.mdi.On("UpsertConfigRecord", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx := context.Background()
	outputValue, err := or.PutConfigRecord(ctx, "foobar", testValue)
	assert.NoError(t, err)
	assert.Equal(t, testValue, outputValue)
}

func TestPutConfigRecordFail(t *testing.T) {
	testValue := fftypes.JSONAnyPtr(`{"foo": "bar"}`)
	or := newTestOrchestrator()
	or.mdi.On("UpsertConfigRecord", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	ctx := context.Background()
	_, err := or.PutConfigRecord(ctx, "foobar", testValue)
	assert.EqualError(t, err, "pop")
}

func TestDeleteConfigRecord(t *testing.T) {
	or := newTestOrchestrator()
	or.mdi.On("DeleteConfigRecord", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ctx := context.Background()
	err := or.DeleteConfigRecord(ctx, "foobar")
	assert.NoError(t, err)
}

func TestGetConfig(t *testing.T) {
	or := newTestOrchestrator()
	config.Reset()
	config.Set(config.LogLevel, "trace")

	conf := or.GetConfig(or.ctx)
	t.Log(conf.String())
	assert.Equal(t, "trace", conf.GetObject("log").GetString("level"))
}

func TestResetConfig(t *testing.T) {
	or := newTestOrchestrator()
	requestCtx, cancelFunc := context.WithCancel(context.Background())
	or.ResetConfig(requestCtx)
	// Simulates the HTTP completing
	cancelFunc()
	<-or.ctx.Done()
}
