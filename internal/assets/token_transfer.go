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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

func (am *assetManager) GetTokenTransfers(ctx context.Context, ns string, filter database.AndFilter) ([]*core.TokenTransfer, *database.FilterResult, error) {
	return am.database.GetTokenTransfers(ctx, am.scopeNS(ns, filter))
}

func (am *assetManager) GetTokenTransferByID(ctx context.Context, ns, id string) (*core.TokenTransfer, error) {
	transferID, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}

	return am.database.GetTokenTransferByID(ctx, transferID)
}

func (am *assetManager) NewTransfer(ns string, transfer *core.TokenTransferInput) sysmessaging.MessageSender {
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
	transfer  *core.TokenTransferInput
	resolved  bool
	msgSender sysmessaging.MessageSender
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

func (am *assetManager) validateTransfer(ctx context.Context, ns string, transfer *core.TokenTransferInput) (pool *core.TokenPool, err error) {
	if transfer.Pool == "" {
		pool, err = am.getDefaultTokenPool(ctx, ns)
		if err != nil {
			return nil, err
		}
	} else {
		pool, err = am.GetTokenPoolByNameOrID(ctx, ns, transfer.Pool)
		if err != nil {
			return nil, err
		}
	}
	transfer.TokenTransfer.Pool = pool.ID
	transfer.TokenTransfer.Connector = pool.Connector

	if pool.State != core.TokenPoolStateConfirmed {
		return nil, i18n.NewError(ctx, coremsgs.MsgTokenPoolNotConfirmed)
	}
	if transfer.Key, err = am.identity.NormalizeSigningKey(ctx, transfer.Key, am.keyNormalization); err != nil {
		return nil, err
	}
	if transfer.From == "" {
		transfer.From = transfer.Key
	}
	if transfer.To == "" {
		transfer.To = transfer.Key
	}
	return pool, nil
}

func (am *assetManager) MintTokens(ctx context.Context, ns string, transfer *core.TokenTransferInput, waitConfirm bool) (out *core.TokenTransfer, err error) {
	transfer.Type = core.TokenTransferTypeMint

	sender := am.NewTransfer(ns, transfer)
	if am.metrics.IsMetricsEnabled() {
		am.metrics.TransferSubmitted(&transfer.TokenTransfer)
	}
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (am *assetManager) BurnTokens(ctx context.Context, ns string, transfer *core.TokenTransferInput, waitConfirm bool) (out *core.TokenTransfer, err error) {
	transfer.Type = core.TokenTransferTypeBurn

	sender := am.NewTransfer(ns, transfer)
	if am.metrics.IsMetricsEnabled() {
		am.metrics.TransferSubmitted(&transfer.TokenTransfer)
	}
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (am *assetManager) TransferTokens(ctx context.Context, ns string, transfer *core.TokenTransferInput, waitConfirm bool) (out *core.TokenTransfer, err error) {
	transfer.Type = core.TokenTransferTypeTransfer

	sender := am.NewTransfer(ns, transfer)
	if am.metrics.IsMetricsEnabled() {
		am.metrics.TransferSubmitted(&transfer.TokenTransfer)
	}
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
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

func (s *transferSender) resolve(ctx context.Context) (err error) {
	// Resolve the attached message
	if s.transfer.Message != nil {
		s.msgSender, err = s.buildTransferMessage(ctx, s.namespace, s.transfer.Message)
		if err != nil {
			return err
		}
		if err = s.msgSender.Prepare(ctx); err != nil {
			return err
		}
		s.transfer.TokenTransfer.Message = s.transfer.Message.Header.ID
		s.transfer.TokenTransfer.MessageHash = s.transfer.Message.Hash
	}
	return nil
}

func (s *transferSender) sendInternal(ctx context.Context, method sendMethod) (err error) {
	if method == methodSendAndWait {
		out, err := s.mgr.syncasync.WaitForTokenTransfer(ctx, s.namespace, s.transfer.LocalID, s.Send)
		if out != nil {
			s.transfer.TokenTransfer = *out
		}
		return err
	}

	var op *core.Operation
	var pool *core.TokenPool
	err = s.mgr.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		pool, err = s.mgr.validateTransfer(ctx, s.namespace, s.transfer)
		if err != nil {
			return err
		}
		if s.transfer.Type == core.TokenTransferTypeTransfer && s.transfer.From == s.transfer.To {
			return i18n.NewError(ctx, coremsgs.MsgCannotTransferToSelf)
		}

		plugin, err := s.mgr.selectTokenPlugin(ctx, s.transfer.Connector)
		if err != nil {
			return err
		}

		if method == methodPrepare {
			return nil
		}

		txid, err := s.mgr.txHelper.SubmitNewTransaction(ctx, s.namespace, core.TransactionTypeTokenTransfer)
		if err != nil {
			return err
		}
		s.transfer.TX.ID = txid
		s.transfer.TX.Type = core.TransactionTypeTokenTransfer

		op = core.NewOperation(
			plugin,
			s.namespace,
			txid,
			core.OpTypeTokenTransfer)
		if err = txcommon.AddTokenTransferInputs(op, &s.transfer.TokenTransfer); err == nil {
			err = s.mgr.database.InsertOperation(ctx, op)
		}
		return err
	})
	if err != nil {
		return err
	} else if method == methodPrepare {
		return nil
	}

	// Write the transfer message outside of any DB transaction, as it will use the background message writer.
	if s.transfer.Message != nil {
		s.transfer.Message.State = core.MessageStateStaged
		if err = s.msgSender.Send(ctx); err != nil {
			return err
		}
	}

	_, err = s.mgr.operations.RunOperation(ctx, opTransfer(op, pool, &s.transfer.TokenTransfer))
	return err
}

func (s *transferSender) buildTransferMessage(ctx context.Context, ns string, in *core.MessageInOut) (sysmessaging.MessageSender, error) {
	allowedTypes := []core.FFEnum{
		core.MessageTypeTransferBroadcast,
		core.MessageTypeTransferPrivate,
	}
	if in.Header.Type == "" {
		in.Header.Type = core.MessageTypeTransferBroadcast
	}
	switch in.Header.Type {
	case core.MessageTypeTransferBroadcast:
		return s.mgr.broadcast.NewBroadcast(ns, in), nil
	case core.MessageTypeTransferPrivate:
		return s.mgr.messaging.NewMessage(ns, in), nil
	default:
		return nil, i18n.NewError(ctx, coremsgs.MsgInvalidMessageType, allowedTypes)
	}
}
