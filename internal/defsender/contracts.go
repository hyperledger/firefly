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

package defsender

import (
	"context"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
)

func (bm *definitionSender) CreateFFI(ctx context.Context, ffi *core.FFI, waitConfirm bool) (output *core.FFI, err error) {
	ffi.ID = fftypes.NewUUID()
	ffi.Namespace = bm.namespace
	for _, method := range ffi.Methods {
		method.ID = fftypes.NewUUID()
	}
	for _, event := range ffi.Events {
		event.ID = fftypes.NewUUID()
	}

	if err := bm.contracts.ResolveFFI(ctx, ffi); err != nil {
		return nil, err
	}

	msg, err := bm.CreateDefinition(ctx, ffi, core.SystemTagDefineFFI, waitConfirm)
	if err != nil {
		return nil, err
	}
	ffi.Message = msg.Header.ID
	return ffi, nil
}

func (bm *definitionSender) CreateContractAPI(ctx context.Context, httpServerURL string, api *core.ContractAPI, waitConfirm bool) (output *core.ContractAPI, err error) {
	api.ID = fftypes.NewUUID()
	api.Namespace = bm.namespace

	if err := bm.contracts.ResolveContractAPI(ctx, httpServerURL, api); err != nil {
		return nil, err
	}

	msg, err := bm.CreateDefinition(ctx, api, core.SystemTagDefineContractAPI, waitConfirm)
	if err != nil {
		return nil, err
	}
	api.Message = msg.Header.ID
	return api, nil
}
