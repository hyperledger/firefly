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

package operations

import (
	"context"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type OperationHandler interface {
	PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error)
	RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (complete bool, err error)
}

type Manager interface {
	RegisterHandler(handler OperationHandler, ops []fftypes.OpType)
	PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error)
	RunOperation(ctx context.Context, op *fftypes.PreparedOperation) error
}

type operationsManager struct {
	ctx      context.Context
	database database.Plugin
	tokens   map[string]tokens.Plugin
	handlers map[fftypes.OpType]OperationHandler
}

func NewOperationsManager(ctx context.Context, di database.Plugin, ti map[string]tokens.Plugin) (Manager, error) {
	if di == nil || ti == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	om := &operationsManager{
		ctx:      ctx,
		database: di,
		tokens:   ti,
		handlers: make(map[fftypes.OpType]OperationHandler),
	}
	return om, nil
}

func (om *operationsManager) RegisterHandler(handler OperationHandler, ops []fftypes.OpType) {
	for _, opType := range ops {
		om.handlers[opType] = handler
	}
}

func (om *operationsManager) PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error) {
	handler, ok := om.handlers[op.Type]
	if !ok {
		return nil, i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
	return handler.PrepareOperation(ctx, op)
}

func (om *operationsManager) RunOperation(ctx context.Context, op *fftypes.PreparedOperation) error {
	handler, ok := om.handlers[op.Type]
	if !ok {
		return i18n.NewError(ctx, i18n.MsgOperationNotSupported)
	}
	if complete, err := handler.RunOperation(ctx, op); err != nil {
		om.writeOperationFailure(ctx, op.ID, err)
		return err
	} else if complete {
		om.writeOperationSuccess(ctx, op.ID, nil)
	}
	return nil
}

func (om *operationsManager) writeOperationSuccess(ctx context.Context, opID *fftypes.UUID, output fftypes.JSONObject) {
	if err := om.database.ResolveOperation(ctx, opID, fftypes.OpStatusSucceeded, "", output); err != nil {
		log.L(ctx).Errorf("Failed to update operation %s: %s", opID, err)
	}
}

func (om *operationsManager) writeOperationFailure(ctx context.Context, opID *fftypes.UUID, err error) {
	if err := om.database.ResolveOperation(ctx, opID, fftypes.OpStatusFailed, err.Error(), nil); err != nil {
		log.L(ctx).Errorf("Failed to update operation %s: %s", opID, err)
	}
}
