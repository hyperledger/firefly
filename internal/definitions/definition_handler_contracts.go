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

package definitions

import (
	"context"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (dh *definitionHandlers) persistContractInterface(ctx context.Context, contractInterface *fftypes.FFI) (valid bool, err error) {
	err = dh.database.InsertContractInterface(ctx, contractInterface)
	if err != nil {
		return false, err
	}

	for _, method := range contractInterface.Methods {
		err := dh.database.UpsertContractMethod(ctx, contractInterface.Namespace, contractInterface.ID, method)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (dh *definitionHandlers) persistContractAPI(ctx context.Context, api *fftypes.ContractAPI) (valid bool, err error) {
	err = dh.database.InsertContractAPI(ctx, api)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (dh *definitionHandlers) handleContractInterfaceBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (SystemBroadcastAction, error) {
	l := log.L(ctx)
	var broadcast fftypes.FFI
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &broadcast)
	if valid {
		if err := broadcast.Validate(ctx, true); err != nil {
			l.Warnf("Unable to process contract definition broadcast %s - validate failed: %s", msg.Header.ID, err)
			valid = false
		} else {
			broadcast.Message = msg.Header.ID
			valid, err = dh.persistContractInterface(ctx, &broadcast)
			if err != nil {
				return ActionReject, err
			}
		}
	}

	var event *fftypes.Event
	if valid {
		l.Infof("Contract definition created id=%s author=%s", broadcast.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypePoolConfirmed, broadcast.Namespace, broadcast.ID)
	} else {
		l.Warnf("Contract definition rejected id=%s author=%s", broadcast.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypePoolRejected, broadcast.Namespace, broadcast.ID)
	}
	err := dh.database.InsertEvent(ctx, event)
	return ActionConfirm, err
}

func (dh *definitionHandlers) handleContractAPIBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (SystemBroadcastAction, error) {
	l := log.L(ctx)
	var broadcast fftypes.ContractAPI
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &broadcast)
	if valid {
		if err := broadcast.Validate(ctx, true); err != nil {
			l.Warnf("Unable to process contract API broadcast %s - validate failed: %s", msg.Header.ID, err)
			valid = false
		} else {
			broadcast.Message = msg.Header.ID
			valid, err = dh.persistContractAPI(ctx, &broadcast)
			if err != nil {
				return ActionReject, err
			}
		}
	}

	var event *fftypes.Event
	if valid {
		l.Infof("Contract API created id=%s author=%s", broadcast.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypePoolConfirmed, broadcast.Namespace, broadcast.ID)
	} else {
		l.Warnf("Contract API rejected id=%s author=%s", broadcast.ID, msg.Header.Author)
		event = fftypes.NewEvent(fftypes.EventTypePoolRejected, broadcast.Namespace, broadcast.ID)
	}
	err := dh.database.InsertEvent(ctx, event)
	return ActionConfirm, err
}
