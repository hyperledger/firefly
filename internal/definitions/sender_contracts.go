// Copyright © 2023 Kaleido, Inc.
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

package definitions

import (
	"context"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
)

func (ds *definitionSender) DefineFFI(ctx context.Context, ffi *fftypes.FFI, waitConfirm bool) error {
	ffi.ID = fftypes.NewUUID()
	for _, method := range ffi.Methods {
		method.ID = fftypes.NewUUID()
	}
	for _, event := range ffi.Events {
		event.ID = fftypes.NewUUID()
	}
	for _, errorDef := range ffi.Errors {
		errorDef.ID = fftypes.NewUUID()
	}

	if ds.multiparty {
		if err := ds.contracts.ResolveFFI(ctx, ffi); err != nil {
			return err
		}

		ffi.Namespace = ""
		msg, err := ds.getSenderDefault(ctx, ffi, core.SystemTagDefineFFI).send(ctx, waitConfirm)
		if msg != nil {
			ffi.Message = msg.Header.ID
		}
		ffi.Namespace = ds.namespace
		return err
	}

	return fakeBatch(ctx, func(ctx context.Context, state *core.BatchState) (HandlerResult, error) {
		return ds.handler.handleFFIDefinition(ctx, state, ffi, nil)
	})
}

func (ds *definitionSender) DefineContractAPI(ctx context.Context, httpServerURL string, api *core.ContractAPI, waitConfirm bool) error {
	if api.ID == nil {
		api.ID = fftypes.NewUUID()
	}

	if ds.multiparty {
		if err := ds.contracts.ResolveContractAPI(ctx, httpServerURL, api); err != nil {
			return err
		}

		api.Namespace = ""
		msg, err := ds.getSenderDefault(ctx, api, core.SystemTagDefineContractAPI).send(ctx, waitConfirm)
		if msg != nil {
			api.Message = msg.Header.ID
		}
		api.Namespace = ds.namespace
		return err
	}

	return fakeBatch(ctx, func(ctx context.Context, state *core.BatchState) (HandlerResult, error) {
		return ds.handler.handleContractAPIDefinition(ctx, state, httpServerURL, api, nil)
	})
}
