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

package webhooks

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/mocks/eventsmocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestWebHooks(t *testing.T) (wh *WebHooks, cancel func()) {
	config.Reset()

	cbs := &eventsmocks.Callbacks{}
	cbs.On("RegisterConnection", "*", mock.Anything).Return(nil)
	wh = &WebHooks{}
	ctx, cancelCtx := context.WithCancel(context.Background())
	svrPrefix := config.NewPluginConfig("ut.webhooks")
	wh.InitPrefix(svrPrefix)
	wh.Init(ctx, svrPrefix, cbs)
	assert.Equal(t, "webhooks", wh.Name())
	assert.NotNil(t, wh.Capabilities())
	assert.NotNil(t, wh.GetOptionsSchema())
	return wh, cancelCtx
}

func TestValidateOptionsWithDataFalse(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	no := false
	opts := &fftypes.SubscriptionOptions{
		SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
			WithData: &no,
		},
	}
	err := wh.ValidateOptions(opts)
	assert.NoError(t, err)
	assert.False(t, *opts.WithData)
}

func TestValidateOptionsWithDataDefaulTrue(t *testing.T) {
	wh, cancel := newTestWebHooks(t)
	defer cancel()

	opts := &fftypes.SubscriptionOptions{}
	err := wh.ValidateOptions(opts)
	assert.NoError(t, err)
	assert.True(t, *opts.WithData)
}
