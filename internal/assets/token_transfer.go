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

package assets

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (am *assetManager) GetTokenTransfers(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenTransfer, *database.FilterResult, error) {
	return am.database.GetTokenTransfers(ctx, am.scopeNS(ns, filter))
}

func (am *assetManager) GetTokenTransferByID(ctx context.Context, ns, id string) (*fftypes.TokenTransfer, error) {
	transferID, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}

	return am.database.GetTokenTransfer(ctx, transferID)
}

func (am *assetManager) GetTokenTransfersByPool(ctx context.Context, ns, connector, name string, filter database.AndFilter) ([]*fftypes.TokenTransfer, *database.FilterResult, error) {
	pool, err := am.GetTokenPool(ctx, ns, connector, name)
	if err != nil {
		return nil, nil, err
	}
	return am.database.GetTokenTransfers(ctx, filter.Condition(filter.Builder().Eq("pool", pool.ID)))
}

func (am *assetManager) NewTransfer(ns string, transfer *fftypes.TokenTransferInput) sysmessaging.MessageSender {
	sender := &transferSender{
		mgr:       am,
		namespace: ns,
		transfer:  transfer,
	}
	sender.setDefaults()
	return sender
}

type transferSender struct {
	mgr       *assetManager
	namespace string
	transfer  *fftypes.TokenTransferInput
	resolved  bool
}

// sendMethod is the specific operation requested of the transferSender.
// To minimize duplication and group database operations, there is a single internal flow with subtle differences for each method.
type sendMethod int

const (
	// methodPrepare requests that the transfer be validated and prepared, but not sent (i.e. no database writes are performed)
	methodPrepare sendMethod = iota
	// methodSend requests that the transfer be sent to the blockchain, but does not wait for confirmation
	methodSend
	// methodSendAndWait requests that the transfer be sent and waits until it is confirmed by the blockchain
	methodSendAndWait
)

func (s *transferSender) Prepare(ctx context.Context) error {
	return s.resolveAndSend(ctx, methodPrepare)
}

func (s *transferSender) Send(ctx context.Context) error {
	return s.resolveAndSend(ctx, methodSend)
}

func (s *transferSender) SendAndWait(ctx context.Context) error {
	return s.resolveAndSend(ctx, methodSendAndWait)
}

func (s *transferSender) setDefaults() {
	s.transfer.LocalID = fftypes.NewUUID()
}

func (am *assetManager) validateTransfer(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput) error {
	if transfer.Connector == "" {
		connector, err := am.getTokenConnectorName(ctx, ns)
		if err != nil {
			return err
		}
		transfer.Connector = connector
	}
	if transfer.Pool == "" {
		pool, err := am.getTokenPoolName(ctx, ns)
		if err != nil {
			return err
		}
		transfer.Pool = pool
	}
	if transfer.Key == "" {
		org, err := am.identity.GetLocalOrganization(ctx)
		if err != nil {
			return err
		}
		transfer.Key = org.Identity
	}
	if transfer.From == "" {
		transfer.From = transfer.Key
	}
	if transfer.To == "" {
		transfer.To = transfer.Key
	}
	return nil
}

func (am *assetManager) MintTokens(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
	transfer.Type = fftypes.TokenTransferTypeMint
	if err := am.validateTransfer(ctx, ns, transfer); err != nil {
		return nil, err
	}

	sender := am.NewTransfer(ns, transfer)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (am *assetManager) MintTokensByType(ctx context.Context, ns, connector, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
	transfer.Connector = connector
	transfer.Pool = poolName
	return am.MintTokens(ctx, ns, transfer, waitConfirm)
}

func (am *assetManager) BurnTokens(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
	transfer.Type = fftypes.TokenTransferTypeBurn
	if err := am.validateTransfer(ctx, ns, transfer); err != nil {
		return nil, err
	}

	sender := am.NewTransfer(ns, transfer)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (am *assetManager) BurnTokensByType(ctx context.Context, ns, connector, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
	transfer.Connector = connector
	transfer.Pool = poolName
	return am.BurnTokens(ctx, ns, transfer, waitConfirm)
}

func (am *assetManager) TransferTokens(ctx context.Context, ns string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
	transfer.Type = fftypes.TokenTransferTypeTransfer
	if err := am.validateTransfer(ctx, ns, transfer); err != nil {
		return nil, err
	}
	if transfer.From == transfer.To {
		return nil, i18n.NewError(ctx, i18n.MsgCannotTransferToSelf)
	}

	sender := am.NewTransfer(ns, transfer)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (am *assetManager) TransferTokensByType(ctx context.Context, ns, connector, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
	transfer.Connector = connector
	transfer.Pool = poolName
	return am.TransferTokens(ctx, ns, transfer, waitConfirm)
}

func (s *transferSender) resolveAndSend(ctx context.Context, method sendMethod) (err error) {
	if !s.resolved {
		if err = s.resolve(ctx); err != nil {
			return err
		}
		s.resolved = true
	}

	if method == methodSendAndWait && s.transfer.Message != nil {
		// Begin waiting for the message, and trigger the transfer.
		// A successful transfer will trigger the message via the event handler, so we can wait for it all to complete.
		_, err := s.mgr.syncasync.WaitForMessage(ctx, s.namespace, s.transfer.Message.Header.ID, func(ctx context.Context) error {
			return s.sendInternal(ctx, methodSendAndWait)
		})
		return err
	}

	return s.sendInternal(ctx, method)
}

func (s *transferSender) resolve(ctx context.Context) error {
	// Resolve the attached message
	if s.transfer.Message != nil {
		sender, err := s.buildTransferMessage(ctx, s.namespace, s.transfer.Message)
		if err != nil {
			return err
		}
		if err = sender.Prepare(ctx); err != nil {
			return err
		}
		s.transfer.TokenTransfer.Message = s.transfer.Message.Header.ID
		s.transfer.TokenTransfer.MessageHash = s.transfer.Message.Hash
	}
	return nil
}

func (s *transferSender) sendInternal(ctx context.Context, method sendMethod) error {
	if method == methodSendAndWait {
		out, err := s.mgr.syncasync.WaitForTokenTransfer(ctx, s.namespace, s.transfer.LocalID, s.Send)
		if out != nil {
			s.transfer.TokenTransfer = *out
		}
		return err
	}

	plugin, err := s.mgr.selectTokenPlugin(ctx, s.transfer.Connector)
	if err != nil {
		return err
	}

	if method == methodPrepare {
		return nil
	}

	tx := &fftypes.Transaction{
		ID:        fftypes.NewUUID(),
		Namespace: s.namespace,
		Type:      fftypes.TransactionTypeTokenTransfer,
		Created:   fftypes.Now(),
	}
	s.transfer.TX.ID = tx.ID
	s.transfer.TX.Type = tx.Type

	op := fftypes.NewTXOperation(
		plugin,
		s.namespace,
		tx.ID,
		"",
		fftypes.OpTypeTokenTransfer,
		fftypes.OpStatusPending)
	txcommon.AddTokenTransferInputs(op, &s.transfer.TokenTransfer)

	var pool *fftypes.TokenPool
	err = s.mgr.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		pool, err = s.mgr.GetTokenPoolByNameOrID(ctx, s.namespace, s.transfer.Pool)
		if err != nil {
			return err
		}
		if pool.State != fftypes.TokenPoolStateConfirmed {
			return i18n.NewError(ctx, i18n.MsgTokenPoolNotConfirmed)
		}

		err = s.mgr.database.UpsertTransaction(ctx, tx)
		if err != nil {
			return err
		}
		if err = s.mgr.database.InsertOperation(ctx, op); err != nil {
			return err
		}
		if s.transfer.Message != nil {
			s.transfer.Message.State = fftypes.MessageStateStaged
			err = s.mgr.database.UpsertMessage(ctx, &s.transfer.Message.Message, database.UpsertOptimizationNew)
		}
		return err
	})
	if err != nil {
		return err
	}

	switch s.transfer.Type {
	case fftypes.TokenTransferTypeMint:
		err = plugin.MintTokens(ctx, op.ID, pool.ProtocolID, &s.transfer.TokenTransfer)
	case fftypes.TokenTransferTypeTransfer:
		err = plugin.TransferTokens(ctx, op.ID, pool.ProtocolID, &s.transfer.TokenTransfer)
	case fftypes.TokenTransferTypeBurn:
		err = plugin.BurnTokens(ctx, op.ID, pool.ProtocolID, &s.transfer.TokenTransfer)
	default:
		panic(fmt.Sprintf("unknown transfer type: %v", s.transfer.Type))
	}

	// if transaction fails,  mark op as failed in DB
	if err != nil {
		_ = s.mgr.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
			l := log.L(ctx)
			update := database.OperationQueryFactory.NewUpdate(ctx).
				Set("status", fftypes.OpStatusFailed)
			if err = s.mgr.database.UpdateOperation(ctx, op.ID, update); err != nil {
				l.Errorf("Operation update failed: %s update=[ %s ]", err, update)
			}

			return nil
		})
	}

	return err
}

func (s *transferSender) buildTransferMessage(ctx context.Context, ns string, in *fftypes.MessageInOut) (sysmessaging.MessageSender, error) {
	allowedTypes := []fftypes.FFEnum{
		fftypes.MessageTypeTransferBroadcast,
		fftypes.MessageTypeTransferPrivate,
	}
	if in.Header.Type == "" {
		in.Header.Type = fftypes.MessageTypeTransferBroadcast
	}
	switch in.Header.Type {
	case fftypes.MessageTypeTransferBroadcast:
		return s.mgr.broadcast.NewBroadcast(ns, in), nil
	case fftypes.MessageTypeTransferPrivate:
		return s.mgr.messaging.NewMessage(ns, in), nil
	default:
		return nil, i18n.NewError(ctx, i18n.MsgInvalidMessageType, allowedTypes)
	}
}
