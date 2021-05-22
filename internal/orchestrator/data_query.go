// Copyright © 2021 Kaleido, Inc.
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

package orchestrator

import (
	"context"
	"database/sql/driver"

	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

func (or *orchestrator) verifyNamespace(ctx context.Context, ns string) error {
	return fftypes.ValidateFFNameField(ctx, ns, "namespace")
}

func (or *orchestrator) verifyIDAndNamespace(ctx context.Context, ns, id string) (*fftypes.UUID, error) {
	u, err := fftypes.ParseUUID(id)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgInvalidUUID)
	}
	err = or.verifyNamespace(ctx, ns)
	return u, err
}

func (or *orchestrator) GetNamespace(ctx context.Context, ns string) (*fftypes.Namespace, error) {
	return or.database.GetNamespace(ctx, ns)
}

func (or *orchestrator) GetTransactionByID(ctx context.Context, ns, id string) (*fftypes.Transaction, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetTransactionByID(ctx, u)
}

func (or *orchestrator) GetMessageByID(ctx context.Context, ns, id string) (*fftypes.Message, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetMessageByID(ctx, u)
}

func (or *orchestrator) GetBatchByID(ctx context.Context, ns, id string) (*fftypes.Batch, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetBatchByID(ctx, u)
}

func (or *orchestrator) GetDataByID(ctx context.Context, ns, id string) (*fftypes.Data, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetDataByID(ctx, u)
}

func (or *orchestrator) GetDataDefinitionByID(ctx context.Context, ns, id string) (*fftypes.DataDefinition, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetDataDefinitionByID(ctx, u)
}

func (or *orchestrator) GetOperationByID(ctx context.Context, ns, id string) (*fftypes.Operation, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetOperationByID(ctx, u)
}

func (or *orchestrator) GetEventByID(ctx context.Context, ns, id string) (*fftypes.Event, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetEventByID(ctx, u)
}

func (or *orchestrator) GetNamespaces(ctx context.Context, filter database.AndFilter) ([]*fftypes.Namespace, error) {
	return or.database.GetNamespaces(ctx, filter)
}

func (or *orchestrator) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (or *orchestrator) GetTransactions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Transaction, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetTransactions(ctx, filter)
}

func (or *orchestrator) GetMessages(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Message, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetMessages(ctx, filter)
}

func (or *orchestrator) GetMessageOperations(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Operation, error) {
	filter = or.scopeNS(ns, filter)
	filter = filter.Condition(filter.Builder().Eq("message", id))
	return or.database.GetOperations(ctx, filter)
}

func (or *orchestrator) GetMessageEvents(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Event, error) {
	msg, err := or.GetMessageByID(ctx, ns, id)
	if err != nil || msg == nil {
		return nil, err
	}
	// Events can refer to the message, or any data in the message
	// So scope the event down to those referred UUIDs, in addition to any and conditions passed in
	referencedIDs := make([]driver.Value, len(msg.Data)+1)
	referencedIDs[0] = msg.Header.ID
	for i, dataRef := range msg.Data {
		referencedIDs[i+1] = dataRef.ID
	}
	filter = filter.Condition(filter.Builder().In("reference", referencedIDs))
	// Execute the filter
	return or.database.GetEvents(ctx, filter)
}

func (or *orchestrator) GetBatches(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Batch, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetBatches(ctx, filter)
}

func (or *orchestrator) GetData(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Data, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetData(ctx, filter)
}

func (or *orchestrator) GetMessagesForData(ctx context.Context, ns, dataID string, filter database.AndFilter) ([]*fftypes.Message, error) {
	filter = or.scopeNS(ns, filter)
	u, err := or.verifyIDAndNamespace(ctx, ns, dataID)
	if err != nil {
		return nil, err
	}
	return or.database.GetMessagesForData(ctx, u, filter)
}

func (or *orchestrator) GetDataDefinitions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.DataDefinition, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetDataDefinitions(ctx, filter)
}

func (or *orchestrator) GetOperations(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Operation, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetOperations(ctx, filter)
}

func (or *orchestrator) GetEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Event, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetEvents(ctx, filter)
}
