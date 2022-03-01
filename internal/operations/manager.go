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
	fftypes.Named
	PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error)
	RunOperation(ctx context.Context, op *fftypes.PreparedOperation) (complete bool, err error)
}

type Manager interface {
	RegisterHandler(ctx context.Context, handler OperationHandler, ops []fftypes.OpType)
	PrepareOperation(ctx context.Context, op *fftypes.Operation) (*fftypes.PreparedOperation, error)
	RunOperation(ctx context.Context, op *fftypes.PreparedOperation) error
	RetryOperation(ctx context.Context, ns string, opID *fftypes.UUID) (*fftypes.Operation, error)
	AddOrReuseOperation(ctx context.Context, op *fftypes.Operation) error
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

func (om *operationsManager) RegisterHandler(ctx context.Context, handler OperationHandler, ops []fftypes.OpType) {
	for _, opType := range ops {
		log.L(ctx).Debugf("OpType=%s registered to handler %s", opType, handler.Name())
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
	log.L(ctx).Infof("Executing %s operation %s via handler %s", op.Type, op.ID, handler.Name())
	log.L(ctx).Tracef("Operation detail: %+v", op)
	if complete, err := handler.RunOperation(ctx, op); err != nil {
		om.writeOperationFailure(ctx, op.ID, err)
		return err
	} else if complete {
		om.writeOperationSuccess(ctx, op.ID)
	}
	return nil
}

func (om *operationsManager) findLatestRetry(ctx context.Context, opID *fftypes.UUID) (op *fftypes.Operation, err error) {
	op, err = om.database.GetOperationByID(ctx, opID)
	if err != nil {
		return nil, err
	}
	if op.Retry == nil {
		return op, nil
	}
	return om.findLatestRetry(ctx, op.Retry)
}

func (om *operationsManager) RetryOperation(ctx context.Context, ns string, opID *fftypes.UUID) (op *fftypes.Operation, err error) {
	var po *fftypes.PreparedOperation
	err = om.database.RunAsGroup(ctx, func(ctx context.Context) error {
		op, err = om.findLatestRetry(ctx, opID)
		if err != nil {
			return err
		}

		// Create a copy of the operation with a new ID
		op.ID = fftypes.NewUUID()
		op.Status = fftypes.OpStatusPending
		op.Error = ""
		op.Output = nil
		op.Created = fftypes.Now()
		op.Updated = op.Created
		if err = om.database.InsertOperation(ctx, op); err != nil {
			return err
		}

		// Update the old operation to point to the new one
		update := database.OperationQueryFactory.NewUpdate(ctx).Set("retry", op.ID)
		if err = om.database.UpdateOperation(ctx, opID, update); err != nil {
			return err
		}

		po, err = om.PrepareOperation(ctx, op)
		return err
	})
	if err != nil {
		return nil, err
	}

	return op, om.RunOperation(ctx, po)
}

func (om *operationsManager) writeOperationSuccess(ctx context.Context, opID *fftypes.UUID) {
	if err := om.database.ResolveOperation(ctx, opID, fftypes.OpStatusSucceeded, "", nil); err != nil {
		log.L(ctx).Errorf("Failed to update operation %s: %s", opID, err)
	}
}

func (om *operationsManager) writeOperationFailure(ctx context.Context, opID *fftypes.UUID, err error) {
	if err := om.database.ResolveOperation(ctx, opID, fftypes.OpStatusFailed, err.Error(), nil); err != nil {
		log.L(ctx).Errorf("Failed to update operation %s: %s", opID, err)
	}
}
