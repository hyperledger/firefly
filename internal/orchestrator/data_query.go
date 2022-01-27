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

package orchestrator

import (
	"context"
	"database/sql/driver"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (or *orchestrator) verifyNamespaceSyntax(ctx context.Context, ns string) error {
	return fftypes.ValidateFFNameField(ctx, ns, "namespace")
}

func (or *orchestrator) verifyIDAndNamespace(ctx context.Context, ns, id string) (*fftypes.UUID, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	err = or.verifyNamespaceSyntax(ctx, ns)
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

func (or *orchestrator) GetTransactionOperations(ctx context.Context, ns, id string) ([]*fftypes.Operation, *database.FilterResult, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, nil, err
	}
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", u),
		fb.Eq("namespace", ns),
	)
	return or.database.GetOperations(ctx, filter)
}

func (or *orchestrator) getMessageByID(ctx context.Context, ns, id string) (*fftypes.Message, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	msg, err := or.database.GetMessageByID(ctx, u)
	if err == nil && msg == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	return msg, err
}

func (or *orchestrator) GetMessageByID(ctx context.Context, ns, id string) (*fftypes.Message, error) {
	return or.getMessageByID(ctx, ns, id)
}

func (or *orchestrator) fetchMessageData(ctx context.Context, msg *fftypes.Message) (*fftypes.MessageInOut, error) {
	msgI := &fftypes.MessageInOut{
		Message: *msg,
	}
	// Lookup the full data
	data, _, err := or.data.GetMessageData(ctx, msg, true)
	if err != nil {
		return nil, err
	}
	msgI.SetInlineData(data)
	return msgI, err
}

func (or *orchestrator) GetMessageByIDWithData(ctx context.Context, ns, id string) (*fftypes.MessageInOut, error) {
	msg, err := or.getMessageByID(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.fetchMessageData(ctx, msg)
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
	return or.database.GetDataByID(ctx, u, true)
}

func (or *orchestrator) GetDatatypeByID(ctx context.Context, ns, id string) (*fftypes.Datatype, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetDatatypeByID(ctx, u)
}

func (or *orchestrator) GetDatatypeByName(ctx context.Context, ns, name, version string) (*fftypes.Datatype, error) {
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}
	if err := fftypes.ValidateFFNameFieldNoUUID(ctx, name, "name"); err != nil {
		return nil, err
	}
	return or.database.GetDatatypeByName(ctx, ns, name, version)
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

func (or *orchestrator) GetNamespaces(ctx context.Context, filter database.AndFilter) ([]*fftypes.Namespace, *database.FilterResult, error) {
	return or.database.GetNamespaces(ctx, filter)
}

func (or *orchestrator) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (or *orchestrator) GetTransactions(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Transaction, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetTransactions(ctx, filter)
}

func (or *orchestrator) GetMessages(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Message, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetMessages(ctx, filter)
}

func (or *orchestrator) GetMessagesWithData(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.MessageInOut, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	msgs, fr, err := or.database.GetMessages(ctx, filter)
	if err != nil {
		return nil, nil, err
	}
	msgsData := make([]*fftypes.MessageInOut, len(msgs))
	for i, msg := range msgs {
		if msgsData[i], err = or.fetchMessageData(ctx, msg); err != nil {
			return nil, nil, err
		}
	}
	return msgsData, fr, err
}

func (or *orchestrator) GetMessageData(ctx context.Context, ns, id string) ([]*fftypes.Data, error) {
	msg, err := or.getMessageByID(ctx, ns, id)
	if err != nil || msg == nil {
		return nil, err
	}
	data, _, err := or.data.GetMessageData(ctx, msg, true)
	return data, err
}

func (or *orchestrator) getMessageTransactionID(ctx context.Context, ns, id string) (*fftypes.UUID, error) {
	msg, err := or.getMessageByID(ctx, ns, id)
	if err != nil || msg == nil {
		return nil, err
	}
	var txID *fftypes.UUID
	if msg.Header.TxType == fftypes.TransactionTypeBatchPin {
		if msg.BatchID == nil {
			return nil, i18n.NewError(ctx, i18n.MsgBatchNotSet)
		}
		batch, err := or.database.GetBatchByID(ctx, msg.BatchID)
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return nil, i18n.NewError(ctx, i18n.MsgBatchNotFound, msg.BatchID)
		}
		txID = batch.Payload.TX.ID
		if txID == nil {
			return nil, i18n.NewError(ctx, i18n.MsgBatchTXNotSet, msg.BatchID)
		}
	} else {
		return nil, i18n.NewError(ctx, i18n.MsgNoTransaction)
	}
	return txID, nil
}

func (or *orchestrator) GetMessageTransaction(ctx context.Context, ns, id string) (*fftypes.Transaction, error) {
	txID, err := or.getMessageTransactionID(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database.GetTransactionByID(ctx, txID)
}

func (or *orchestrator) GetMessageOperations(ctx context.Context, ns, id string) ([]*fftypes.Operation, *database.FilterResult, error) {
	txID, err := or.getMessageTransactionID(ctx, ns, id)
	if err != nil {
		return nil, nil, err
	}
	filter := database.OperationQueryFactory.NewFilter(ctx).Eq("tx", txID)
	return or.database.GetOperations(ctx, filter)
}

func (or *orchestrator) GetMessageEvents(ctx context.Context, ns, id string, filter database.AndFilter) ([]*fftypes.Event, *database.FilterResult, error) {
	msg, err := or.getMessageByID(ctx, ns, id)
	if err != nil || msg == nil {
		return nil, nil, err
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

func (or *orchestrator) GetBatches(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Batch, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetBatches(ctx, filter)
}

func (or *orchestrator) GetData(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Data, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetData(ctx, filter)
}

func (or *orchestrator) GetMessagesForData(ctx context.Context, ns, dataID string, filter database.AndFilter) ([]*fftypes.Message, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	u, err := or.verifyIDAndNamespace(ctx, ns, dataID)
	if err != nil {
		return nil, nil, err
	}
	return or.database.GetMessagesForData(ctx, u, filter)
}

func (or *orchestrator) GetDatatypes(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Datatype, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetDatatypes(ctx, filter)
}

func (or *orchestrator) GetOperations(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Operation, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetOperations(ctx, filter)
}

func (or *orchestrator) GetEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Event, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database.GetEvents(ctx, filter)
}

func (or *orchestrator) GetBlockchainEventByID(ctx context.Context, id *fftypes.UUID) (*fftypes.BlockchainEvent, error) {
	return or.database.GetBlockchainEventByID(ctx, id)
}

func (or *orchestrator) GetBlockchainEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.BlockchainEvent, *database.FilterResult, error) {
	return or.database.GetBlockchainEvents(ctx, or.scopeNS(ns, filter))
}

func (or *orchestrator) GetTransactionBlockchainEvents(ctx context.Context, ns, id string) ([]*fftypes.BlockchainEvent, *database.FilterResult, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, nil, err
	}
	fb := database.BlockchainEventQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx.id", u),
		fb.Eq("namespace", ns),
	)
	return or.database.GetBlockchainEvents(ctx, filter)
}

func (or *orchestrator) GetTransactionStatus(ctx context.Context, ns, id string) (*fftypes.TransactionStatus, error) {
	return &fftypes.TransactionStatus{Status: fftypes.OpStatusPending}, nil
}
