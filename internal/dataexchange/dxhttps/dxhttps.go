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
	"io"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/pkg/dataexchange"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

type HTTPS struct {
	capabilities *dataexchange.Capabilities
	callbacks    dataexchange.Callbacks
}

func (h *HTTPS) Name() string {
	return "https"
}

func (h *HTTPS) Init(ctx context.Context, prefix config.Prefix, callbacks dataexchange.Callbacks) (err error) {
	h.callbacks = callbacks
	h.capabilities = &dataexchange.Capabilities{}
	return nil
}

func (h *HTTPS) Start() error {
	return nil
}

func (h *HTTPS) Capabilities() *dataexchange.Capabilities {
	return h.capabilities
}

func (h *HTTPS) GetEndpointInfo(ctx context.Context) (endpoint fftypes.JSONObject, err error) {
	return
}

func (h *HTTPS) UploadBLOB(ctx context.Context, ns string, id fftypes.UUID, reader io.Reader) error {
	return nil
}
