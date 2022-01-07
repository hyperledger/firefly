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

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (dh *definitionHandlers) persistFFI(ctx context.Context, ffi *fftypes.FFI) (valid bool, err error) {
	if err := dh.contracts.ValidateFFIAndSetPathnames(ctx, ffi); err != nil {
		log.L(ctx).Warnf("Unable to process FFI %s - validate failed: %s", ffi.ID, err)
		return false, nil
	}

	err = dh.database.UpsertFFI(ctx, ffi)
	if err != nil {
		return false, err
	}

	for _, method := range ffi.Methods {
		err := dh.database.UpsertFFIMethod(ctx, method)
		if err != nil {
			return false, err
		}
	}

	for _, event := range ffi.Events {
		err := dh.database.UpsertFFIEvent(ctx, event)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (dh *definitionHandlers) persistContractAPI(ctx context.Context, api *fftypes.ContractAPI) (valid bool, err error) {
	existing, err := dh.database.GetContractAPIByName(ctx, api.Namespace, api.Name)
	if err != nil {
		return false, err
	}
	if existing != nil && err == nil {
		if !api.LocationAndLedgerEquals(existing) {
			return false, nil
		}
	}
	err = dh.database.UpsertContractAPI(ctx, api, database.UpsertOptimizationSkip)
	return err == nil, err
}

func (dh *definitionHandlers) handleFFIBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (actionResult SystemBroadcastAction, err error) {
	l := log.L(ctx)
	var broadcast fftypes.FFI
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &broadcast)
	if valid {
		if validationErr := broadcast.Validate(ctx, true); validationErr != nil {
			l.Warnf("Unable to process contract definition broadcast %s - validate failed: %s", msg.Header.ID, err)
			valid = false
		} else {
			broadcast.Message = msg.Header.ID
			valid, err = dh.persistFFI(ctx, &broadcast)
		}
	}

	var event *fftypes.Event
	switch {
	case err != nil:
		actionResult = ActionRetry
	case valid:
		l.Infof("Contract interface created id=%s author=%s", broadcast.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypeContractInterfaceConfirmed, broadcast.Namespace, broadcast.ID)
		actionResult = ActionConfirm
		err = dh.database.InsertEvent(ctx, event)
	default:
		l.Warnf("Contract interface rejected id=%s author=%s", broadcast.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypeContractInterfaceRejected, broadcast.Namespace, broadcast.ID)
		actionResult = ActionReject
		err = dh.database.InsertEvent(ctx, event)
	}
	return
}

func (dh *definitionHandlers) handleContractAPIBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (actionResult SystemBroadcastAction, err error) {
	l := log.L(ctx)
	var broadcast fftypes.ContractAPI
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &broadcast)
	if valid {
		if validateErr := broadcast.Validate(ctx, true); validateErr != nil {
			l.Warnf("Unable to process contract API broadcast %s - validate failed: %s", msg.Header.ID, err)
			valid = false
		} else {
			broadcast.Message = msg.Header.ID
			valid, err = dh.persistContractAPI(ctx, &broadcast)
		}
	}

	var event *fftypes.Event
	switch {
	case err != nil:
		actionResult = ActionRetry
	case valid:
		l.Infof("Contract API created id=%s author=%s", broadcast.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypeContractAPIConfirmed, broadcast.Namespace, broadcast.ID)
		actionResult = ActionConfirm
		err = dh.database.InsertEvent(ctx, event)
	default:
		l.Warnf("Contract API rejected id=%s author=%s", broadcast.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypeContractAPIRejected, broadcast.Namespace, broadcast.ID)
		actionResult = ActionReject
		err = dh.database.InsertEvent(ctx, event)
	}
	return
}
