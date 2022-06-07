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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

func (or *orchestrator) verifyNamespaceSyntax(ctx context.Context, ns string) error {
	return core.ValidateFFNameField(ctx, ns, "namespace")
}

func (or *orchestrator) checkNamespace(ctx context.Context, requiredNS, objectNS string) error {
	if objectNS != requiredNS {
		log.L(ctx).Warnf("Object queried by ID in wrong namespace. Required=%s Found=%s", requiredNS, objectNS)
		return i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	return nil
}

func (or *orchestrator) verifyIDAndNamespace(ctx context.Context, ns, id string) (*fftypes.UUID, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	err = or.verifyNamespaceSyntax(ctx, ns)
	return u, err
}

func (or *orchestrator) GetNamespace(ctx context.Context, ns string) (*core.Namespace, error) {
	return or.database().GetNamespace(ctx, ns)
}

func (or *orchestrator) GetTransactionByID(ctx context.Context, ns, id string) (*core.Transaction, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	tx, err := or.database().GetTransactionByID(ctx, u)
	if err == nil && tx != nil {
		err = or.checkNamespace(ctx, ns, tx.Namespace)
	}
	return tx, err
}

func (or *orchestrator) GetTransactionOperations(ctx context.Context, ns, id string) ([]*core.Operation, *database.FilterResult, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, nil, err
	}
	fb := database.OperationQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx", u),
		fb.Eq("namespace", ns),
	)
	return or.database().GetOperations(ctx, filter)
}

func (or *orchestrator) getMessageByID(ctx context.Context, ns, id string) (*core.Message, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	msg, err := or.database().GetMessageByID(ctx, u)
	if err == nil && msg == nil {
		return nil, i18n.NewError(ctx, coremsgs.Msg404NotFound)
	}
	if err == nil && msg != nil {
		err = or.checkNamespace(ctx, ns, msg.Header.Namespace)
	}
	return msg, err
}

func (or *orchestrator) GetMessageByID(ctx context.Context, ns, id string) (*core.Message, error) {
	return or.getMessageByID(ctx, ns, id)
}

func (or *orchestrator) fetchMessageData(ctx context.Context, msg *core.Message) (*core.MessageInOut, error) {
	msgI := &core.MessageInOut{
		Message: *msg,
	}
	// Lookup the full data
	data, _, err := or.data.GetMessageDataCached(ctx, msg)
	if err != nil {
		return nil, err
	}
	msgI.SetInlineData(data)
	return msgI, err
}

func (or *orchestrator) GetMessageByIDWithData(ctx context.Context, ns, id string) (*core.MessageInOut, error) {
	msg, err := or.getMessageByID(ctx, ns, id)
	if err == nil && msg != nil {
		err = or.checkNamespace(ctx, ns, msg.Header.Namespace)
	}
	if err != nil {
		return nil, err
	}
	return or.fetchMessageData(ctx, msg)
}

func (or *orchestrator) GetBatchByID(ctx context.Context, ns, id string) (*core.BatchPersisted, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	b, err := or.database().GetBatchByID(ctx, u)
	if err == nil && b != nil {
		err = or.checkNamespace(ctx, ns, b.Namespace)
	}
	return b, err
}

func (or *orchestrator) GetDataByID(ctx context.Context, ns, id string) (*core.Data, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	d, err := or.database().GetDataByID(ctx, u, true)
	if err == nil && d != nil {
		err = or.checkNamespace(ctx, ns, d.Namespace)
	}
	return d, err
}

func (or *orchestrator) GetDatatypeByID(ctx context.Context, ns, id string) (*core.Datatype, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	dt, err := or.database().GetDatatypeByID(ctx, u)
	if err == nil && dt != nil {
		err = or.checkNamespace(ctx, ns, dt.Namespace)
	}
	return dt, err
}

func (or *orchestrator) GetDatatypeByName(ctx context.Context, ns, name, version string) (*core.Datatype, error) {
	if err := core.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}
	if err := core.ValidateFFNameFieldNoUUID(ctx, name, "name"); err != nil {
		return nil, err
	}
	dt, err := or.database().GetDatatypeByName(ctx, ns, name, version)
	if err == nil && dt != nil {
		err = or.checkNamespace(ctx, ns, dt.Namespace)
	}
	return dt, err
}

func (or *orchestrator) GetOperationByID(ctx context.Context, ns, id string) (*core.Operation, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	o, err := or.database().GetOperationByID(ctx, u)
	if err == nil && o != nil {
		err = or.checkNamespace(ctx, ns, o.Namespace)
	}
	return o, err
}

func (or *orchestrator) GetOperationByNamespacedID(ctx context.Context, nsOpID string) (*core.Operation, error) {
	ns, u, err := core.ParseNamespacedOpID(ctx, nsOpID)
	if err != nil {
		return nil, err
	}
	o, err := or.database().GetOperationByID(ctx, u)
	if err == nil && o != nil {
		err = or.checkNamespace(ctx, ns, o.Namespace)
	}
	return o, err
}

func (or *orchestrator) GetEventByID(ctx context.Context, ns, id string) (*core.Event, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	e, err := or.database().GetEventByID(ctx, u)
	if err == nil && e != nil {
		err = or.checkNamespace(ctx, ns, e.Namespace)
	}
	return e, err
}

func (or *orchestrator) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (or *orchestrator) GetTransactions(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Transaction, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database().GetTransactions(ctx, filter)
}

func (or *orchestrator) GetMessages(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Message, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database().GetMessages(ctx, filter)
}

func (or *orchestrator) GetMessagesWithData(ctx context.Context, ns string, filter database.AndFilter) ([]*core.MessageInOut, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	msgs, fr, err := or.database().GetMessages(ctx, filter)
	if err != nil {
		return nil, nil, err
	}
	msgsData := make([]*core.MessageInOut, len(msgs))
	for i, msg := range msgs {
		if msgsData[i], err = or.fetchMessageData(ctx, msg); err != nil {
			return nil, nil, err
		}
	}
	return msgsData, fr, err
}

func (or *orchestrator) GetMessageData(ctx context.Context, ns, id string) (core.DataArray, error) {
	msg, err := or.getMessageByID(ctx, ns, id)
	if err != nil || msg == nil {
		return nil, err
	}
	data, _, err := or.data.GetMessageDataCached(ctx, msg)
	return data, err
}

func (or *orchestrator) getMessageTransactionID(ctx context.Context, ns, id string) (*fftypes.UUID, error) {
	msg, err := or.getMessageByID(ctx, ns, id)
	if err != nil || msg == nil {
		return nil, err
	}
	var txID *fftypes.UUID
	if msg.Header.TxType == core.TransactionTypeBatchPin {
		if msg.BatchID == nil {
			return nil, i18n.NewError(ctx, coremsgs.MsgBatchNotSet)
		}
		batch, err := or.database().GetBatchByID(ctx, msg.BatchID)
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return nil, i18n.NewError(ctx, coremsgs.MsgBatchNotFound, msg.BatchID)
		}
		txID = batch.TX.ID
		if txID == nil {
			return nil, i18n.NewError(ctx, coremsgs.MsgBatchTXNotSet, msg.BatchID)
		}
	} else {
		return nil, i18n.NewError(ctx, coremsgs.MsgNoTransaction)
	}
	return txID, nil
}

func (or *orchestrator) GetMessageTransaction(ctx context.Context, ns, id string) (*core.Transaction, error) {
	txID, err := or.getMessageTransactionID(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	return or.database().GetTransactionByID(ctx, txID)
}

func (or *orchestrator) GetMessageOperations(ctx context.Context, ns, id string) ([]*core.Operation, *database.FilterResult, error) {
	txID, err := or.getMessageTransactionID(ctx, ns, id)
	if err != nil {
		return nil, nil, err
	}
	filter := database.OperationQueryFactory.NewFilter(ctx).Eq("tx", txID)
	return or.database().GetOperations(ctx, filter)
}

func (or *orchestrator) GetMessageEvents(ctx context.Context, ns, id string, filter database.AndFilter) ([]*core.Event, *database.FilterResult, error) {
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
	return or.database().GetEvents(ctx, filter)
}

func (or *orchestrator) GetBatches(ctx context.Context, ns string, filter database.AndFilter) ([]*core.BatchPersisted, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database().GetBatches(ctx, filter)
}

func (or *orchestrator) GetData(ctx context.Context, ns string, filter database.AndFilter) (core.DataArray, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database().GetData(ctx, filter)
}

func (or *orchestrator) GetMessagesForData(ctx context.Context, ns, dataID string, filter database.AndFilter) ([]*core.Message, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	u, err := or.verifyIDAndNamespace(ctx, ns, dataID)
	if err != nil {
		return nil, nil, err
	}
	return or.database().GetMessagesForData(ctx, u, filter)
}

func (or *orchestrator) GetDatatypes(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Datatype, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database().GetDatatypes(ctx, filter)
}

func (or *orchestrator) GetOperationsNamespaced(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Operation, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database().GetOperations(ctx, filter)
}

func (or *orchestrator) GetOperations(ctx context.Context, filter database.AndFilter) ([]*core.Operation, *database.FilterResult, error) {
	return or.database().GetOperations(ctx, filter)
}

func (or *orchestrator) GetEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*core.Event, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	return or.database().GetEvents(ctx, filter)
}

func (or *orchestrator) GetBlockchainEventByID(ctx context.Context, ns, id string) (*core.BlockchainEvent, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, err
	}
	be, err := or.database().GetBlockchainEventByID(ctx, u)
	if err == nil && be != nil {
		err = or.checkNamespace(ctx, ns, be.Namespace)
	}
	return be, err
}

func (or *orchestrator) GetBlockchainEvents(ctx context.Context, ns string, filter database.AndFilter) ([]*core.BlockchainEvent, *database.FilterResult, error) {
	return or.database().GetBlockchainEvents(ctx, or.scopeNS(ns, filter))
}

func (or *orchestrator) GetTransactionBlockchainEvents(ctx context.Context, ns, id string) ([]*core.BlockchainEvent, *database.FilterResult, error) {
	u, err := or.verifyIDAndNamespace(ctx, ns, id)
	if err != nil {
		return nil, nil, err
	}
	fb := database.BlockchainEventQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("tx.id", u),
		fb.Eq("namespace", ns),
	)
	return or.database().GetBlockchainEvents(ctx, filter)
}

func (or *orchestrator) GetPins(ctx context.Context, filter database.AndFilter) ([]*core.Pin, *database.FilterResult, error) {
	return or.database().GetPins(ctx, filter)
}

func (or *orchestrator) GetEventsWithReferences(ctx context.Context, ns string, filter database.AndFilter) ([]*core.EnrichedEvent, *database.FilterResult, error) {
	filter = or.scopeNS(ns, filter)
	events, fr, err := or.database().GetEvents(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	enriched := make([]*core.EnrichedEvent, len(events))
	for i, event := range events {
		enrichedEvent, err := or.txHelper.EnrichEvent(or.ctx, event)
		if err != nil {
			return nil, nil, err
		}
		enriched[i] = enrichedEvent
	}
	return enriched, fr, err
}
