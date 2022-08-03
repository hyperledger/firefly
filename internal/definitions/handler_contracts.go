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

package definitions

import (
	"context"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

func (dh *definitionHandler) persistFFI(ctx context.Context, ffi *fftypes.FFI) (retry bool, err error) {
	if err = dh.contracts.ResolveFFI(ctx, ffi); err != nil {
		return false, i18n.NewError(ctx, coremsgs.MsgDefRejectedValidateFail, "contract interface", ffi.ID, err)
	}

	err = dh.database.UpsertFFI(ctx, ffi)
	if err != nil {
		return true, err
	}

	for _, method := range ffi.Methods {
		if err := dh.database.UpsertFFIMethod(ctx, method); err != nil {
			return true, err
		}
	}
	for _, event := range ffi.Events {
		if err := dh.database.UpsertFFIEvent(ctx, event); err != nil {
			return true, err
		}
	}

	return false, nil
}

func (dh *definitionHandler) persistContractAPI(ctx context.Context, httpServerURL string, api *core.ContractAPI) (retry bool, err error) {
	if err := dh.contracts.ResolveContractAPI(ctx, httpServerURL, api); err != nil {
		return false, i18n.NewError(ctx, coremsgs.MsgDefRejectedValidateFail, "contract API", api.ID, err)
	}

	err = dh.database.UpsertContractAPI(ctx, api)
	if err != nil {
		if err == database.IDMismatch {
			return false, i18n.NewError(ctx, coremsgs.MsgDefRejectedIDMismatch, "contract API", api.ID)
		}
		return true, err
	}
	return false, nil
}

func (dh *definitionHandler) handleFFIBroadcast(ctx context.Context, state *core.BatchState, msg *core.Message, data core.DataArray, tx *fftypes.UUID) (HandlerResult, error) {
	var ffi fftypes.FFI
	if valid := dh.getSystemBroadcastPayload(ctx, msg, data, &ffi); !valid {
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedBadPayload, "contract interface", msg.Header.ID)
	}
	ffi.Message = msg.Header.ID
	return dh.handleFFIDefinition(ctx, state, &ffi, tx)
}

func (dh *definitionHandler) handleFFIDefinition(ctx context.Context, state *core.BatchState, ffi *fftypes.FFI, tx *fftypes.UUID) (HandlerResult, error) {
	l := log.L(ctx)
	ffi.Namespace = dh.namespace.Name
	if err := ffi.Validate(ctx, true); err != nil {
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedValidateFail, "contract interface", ffi.ID, err)
	}

	if retry, err := dh.persistFFI(ctx, ffi); err != nil {
		if retry {
			return HandlerResult{Action: ActionRetry}, err
		}
		return HandlerResult{Action: ActionReject}, err
	}

	l.Infof("Contract interface created id=%s", ffi.ID)
	state.AddFinalize(func(ctx context.Context) error {
		event := core.NewEvent(core.EventTypeContractInterfaceConfirmed, ffi.Namespace, ffi.ID, tx, ffi.Topic())
		return dh.database.InsertEvent(ctx, event)
	})
	return HandlerResult{Action: ActionConfirm}, nil
}

func (dh *definitionHandler) handleContractAPIBroadcast(ctx context.Context, state *core.BatchState, msg *core.Message, data core.DataArray, tx *fftypes.UUID) (HandlerResult, error) {
	var api core.ContractAPI
	if valid := dh.getSystemBroadcastPayload(ctx, msg, data, &api); !valid {
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedBadPayload, "contract API", msg.Header.ID)
	}
	api.Message = msg.Header.ID
	return dh.handleContractAPIDefinition(ctx, state, "", &api, tx)
}

func (dh *definitionHandler) handleContractAPIDefinition(ctx context.Context, state *core.BatchState, httpServerURL string, api *core.ContractAPI, tx *fftypes.UUID) (HandlerResult, error) {
	l := log.L(ctx)
	api.Namespace = dh.namespace.Name
	if err := api.Validate(ctx, true); err != nil {
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedValidateFail, "contract API", api.ID, err)
	}

	if retry, err := dh.persistContractAPI(ctx, httpServerURL, api); err != nil {
		if retry {
			return HandlerResult{Action: ActionRetry}, err
		}
		return HandlerResult{Action: ActionReject}, err
	}

	l.Infof("Contract API created id=%s", api.ID)
	state.AddFinalize(func(ctx context.Context) error {
		event := core.NewEvent(core.EventTypeContractAPIConfirmed, api.Namespace, api.ID, tx, core.SystemTopicDefinitions)
		return dh.database.InsertEvent(ctx, event)
	})
	return HandlerResult{Action: ActionConfirm}, nil
}
