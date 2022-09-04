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
	"database/sql/driver"
	"fmt"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/cache"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

type OperationHandler interface {
	core.Named
	PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error)
	RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error)
	OnOperationUpdate(ctx context.Context, op *core.Operation, update *core.OperationUpdate) error
}

type Manager interface {
	RegisterHandler(ctx context.Context, handler OperationHandler, ops []core.OpType)
	PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error)
	RunOperation(ctx context.Context, op *core.PreparedOperation, options ...RunOperationOption) (fftypes.JSONObject, error)
	RetryOperation(ctx context.Context, opID *fftypes.UUID) (*core.Operation, error)
	AddOrReuseOperation(ctx context.Context, op *core.Operation, hooks ...database.PostCompletionHook) error
	SubmitOperationUpdate(update *core.OperationUpdate)
	GetOperationByIDCached(ctx context.Context, opID *fftypes.UUID) (*core.Operation, error)
	ResolveOperationByID(ctx context.Context, opID *fftypes.UUID, op *core.OperationUpdateDTO) error
	Start() error
	WaitStop()
}

type RunOperationOption int

const (
	RemainPendingOnFailure RunOperationOption = iota
)

type operationsManager struct {
	ctx       context.Context
	namespace string
	database  database.Plugin
	handlers  map[core.OpType]OperationHandler
	updater   *operationUpdater
	cache     cache.CInterface
}

func NewOperationsManager(ctx context.Context, ns string, di database.Plugin, txHelper txcommon.Helper, cacheManager cache.Manager) (Manager, error) {
	if di == nil || txHelper == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "OperationsManager")
	}

	cache, err := cacheManager.GetCache(
		cache.NewCacheConfig(
			ctx,
			coreconfig.CacheOperationsLimit,
			coreconfig.CacheOperationsTTL,
			ns,
		),
	)

	if err != nil {
		return nil, err
	}

	om := &operationsManager{
		ctx:       ctx,
		namespace: ns,
		database:  di,
		handlers:  make(map[core.OpType]OperationHandler),
	}
	om.updater = newOperationUpdater(ctx, om, di, txHelper)
	om.cache = cache
	return om, nil
}

func (om *operationsManager) RegisterHandler(ctx context.Context, handler OperationHandler, ops []core.OpType) {
	for _, opType := range ops {
		log.L(ctx).Debugf("OpType=%s registered to handler %s", opType, handler.Name())
		om.handlers[opType] = handler
	}
}

func (om *operationsManager) PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error) {
	handler, ok := om.handlers[op.Type]
	if !ok {
		return nil, i18n.NewError(ctx, coremsgs.MsgOperationNotSupported, op.Type)
	}
	return handler.PrepareOperation(ctx, op)
}

func (om *operationsManager) RunOperation(ctx context.Context, op *core.PreparedOperation, options ...RunOperationOption) (fftypes.JSONObject, error) {
	failState := core.OpStatusFailed
	for _, o := range options {
		if o == RemainPendingOnFailure {
			failState = core.OpStatusPending
		}
	}

	handler, ok := om.handlers[op.Type]
	if !ok {
		return nil, i18n.NewError(ctx, coremsgs.MsgOperationNotSupported, op.Type)
	}
	log.L(ctx).Infof("Executing %s operation %s via handler %s", op.Type, op.ID, handler.Name())
	log.L(ctx).Tracef("Operation detail: %+v", op)
	outputs, complete, err := handler.RunOperation(ctx, op)
	if err != nil {
		om.SubmitOperationUpdate(&core.OperationUpdate{
			NamespacedOpID: op.NamespacedIDString(),
			Plugin:         op.Plugin,
			Status:         failState,
			ErrorMessage:   err.Error(),
			Output:         outputs,
		})
	} else if complete {
		om.SubmitOperationUpdate(&core.OperationUpdate{
			NamespacedOpID: op.NamespacedIDString(),
			Plugin:         op.Plugin,
			Status:         core.OpStatusSucceeded,
			Output:         outputs,
		})
	}
	return outputs, err
}

func (om *operationsManager) findLatestRetry(ctx context.Context, opID *fftypes.UUID) (op *core.Operation, err error) {
	op, err = om.GetOperationByIDCached(ctx, opID)
	if err != nil {
		return nil, err
	}
	if op.Retry == nil {
		return op, nil
	}
	return om.findLatestRetry(ctx, op.Retry)
}

func (om *operationsManager) RetryOperation(ctx context.Context, opID *fftypes.UUID) (op *core.Operation, err error) {
	var po *core.PreparedOperation
	err = om.database.RunAsGroup(ctx, func(ctx context.Context) error {
		op, err = om.findLatestRetry(ctx, opID)
		if err != nil {
			return err
		}

		// Create a copy of the operation with a new ID
		op.ID = fftypes.NewUUID()
		op.Status = core.OpStatusPending
		op.Error = ""
		op.Output = nil
		op.Created = fftypes.Now()
		op.Updated = op.Created
		if err = om.database.InsertOperation(ctx, op); err != nil {
			return err
		}
		om.cacheOperation(op)

		// Update the old operation to point to the new one
		update := database.OperationQueryFactory.NewUpdate(ctx).Set("retry", op.ID)
		om.updateCachedOperation(opID, "", nil, nil, op.ID)
		if err = om.database.UpdateOperation(ctx, om.namespace, opID, update); err != nil {
			return err
		}

		po, err = om.PrepareOperation(ctx, op)
		return err
	})
	if err != nil {
		return nil, err
	}

	_, err = om.RunOperation(ctx, po)
	return op, err
}

func (om *operationsManager) ResolveOperationByID(ctx context.Context, opID *fftypes.UUID, op *core.OperationUpdateDTO) error {
	return om.updater.resolveOperation(ctx, om.namespace, opID, op.Status, op.Error, op.Output)
}

func (om *operationsManager) SubmitOperationUpdate(update *core.OperationUpdate) {
	errString := ""
	if update.ErrorMessage != "" {
		errString = fmt.Sprintf(" error=%s", update.ErrorMessage)
	}
	log.L(om.ctx).Debugf("%s updating operation %s status=%s%s", update.Plugin, update.NamespacedOpID, update.Status, errString)
	om.updater.SubmitOperationUpdate(om.ctx, update)
}

func (om *operationsManager) Start() error {
	om.updater.start()
	return nil
}

func (om *operationsManager) WaitStop() {
	om.updater.close()
}

func (om *operationsManager) GetOperationByIDCached(ctx context.Context, opID *fftypes.UUID) (*core.Operation, error) {
	if cached := om.getCachedOperation(opID); cached != nil {
		return cached, nil
	}
	op, err := om.database.GetOperationByID(ctx, om.namespace, opID)
	if err == nil && op != nil {
		om.cacheOperation(op)
	}
	return op, err
}

func (om *operationsManager) getOperationsCached(ctx context.Context, opIDs []*fftypes.UUID) ([]*core.Operation, error) {
	ops := make([]*core.Operation, 0, len(opIDs))
	cacheMisses := make([]driver.Value, 0)
	for _, id := range opIDs {
		if op := om.getCachedOperation(id); op != nil {
			ops = append(ops, op)
		} else {
			cacheMisses = append(cacheMisses, id)
		}
	}

	if len(cacheMisses) > 0 {
		opFilter := database.OperationQueryFactory.NewFilter(ctx).In("id", cacheMisses)
		dbOps, _, err := om.database.GetOperations(ctx, om.namespace, opFilter)
		if err != nil {
			return nil, err
		}
		for _, op := range dbOps {
			om.cacheOperation(op)
		}
		ops = append(ops, dbOps...)
	}
	return ops, nil
}

func (om *operationsManager) getCachedOperation(id *fftypes.UUID) *core.Operation {
	if cachedValue := om.cache.Get(id.String()); cachedValue != nil {
		return cachedValue.(*core.Operation)
	}
	return nil
}

func (om *operationsManager) cacheOperation(op *core.Operation) {
	om.cache.Set(op.ID.String(), op)
}

func (om *operationsManager) updateCachedOperation(id *fftypes.UUID, status core.OpStatus, errorMsg *string, output fftypes.JSONObject, retry *fftypes.UUID) {
	if cachedValue := om.cache.Get(id.String()); cachedValue != nil {
		val := cachedValue.(*core.Operation)
		if status != "" {
			val.Status = status
		}
		if errorMsg != nil {
			val.Error = *errorMsg
		}
		if output != nil {
			val.Output = output
		}
		if retry != nil {
			val.Retry = retry
		}
		om.cacheOperation(val)
	}
}
