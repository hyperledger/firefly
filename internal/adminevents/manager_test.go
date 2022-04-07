// Copyright © 2022 Kaleido, Inc.
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

package adminevents

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/config/wsconfig"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/pkg/wsclient"
	"github.com/stretchr/testify/assert"
)

func newTestAdminEventsManager(t *testing.T) (ae *adminEventManager, wsc wsclient.WSClient, cancel func()) {
	config.Reset()

	ae = NewAdminEventManager(context.Background()).(*adminEventManager)
	svr := httptest.NewServer(http.HandlerFunc(ae.ServeHTTPWebSocketListener))

	clientPrefix := config.NewPluginConfig("ut.wsclient")
	wsconfig.InitPrefix(clientPrefix)
	clientPrefix.Set(restclient.HTTPConfigURL, fmt.Sprintf("http://%s", svr.Listener.Addr()))
	wsConfig := wsconfig.GenerateConfigFromPrefix(clientPrefix)

	wsc, err := wsclient.New(ae.ctx, wsConfig, nil, nil)
	assert.NoError(t, err)
	err = wsc.Connect()
	assert.NoError(t, err)

	return ae, wsc, func() {
		ae.cancelCtx()
		wsc.Close()
		ae.WaitStop()
		svr.Close()
	}
}

func TestAdminEventsE2E(t *testing.T) {
	_, _, cancel := newTestAdminEventsManager(t)
	defer cancel()

}
