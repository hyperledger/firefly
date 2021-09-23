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

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/identity"
)

type OnChain struct {
	capabilities *identity.Capabilities
	callbacks    identity.Callbacks
}

func (oc *OnChain) Name() string {
	return "onchain"
}

func (oc *OnChain) Init(ctx context.Context, prefix config.Prefix, callbacks identity.Callbacks) (err error) {
	oc.callbacks = callbacks
	oc.capabilities = &identity.Capabilities{}
	return nil
}

func (oc *OnChain) Start() error {
	return nil
}

func (oc *OnChain) Capabilities() *identity.Capabilities {
	return oc.capabilities
}

func (oc *OnChain) Resolve(ctx context.Context, identifier string) (*fftypes.Identity, error) {
	return &fftypes.Identity{
		Identifier: identifier,
		OnChain:    identifier,
	}, nil
}
