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

package assets

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

// Note: the counterpart to below (retrieveTokenTransferInputs) lives in the events package
func addTokenTransferInputs(op *fftypes.Operation, transfer *fftypes.TokenTransfer) {
	op.Input = fftypes.JSONObject{
		"id": transfer.LocalID.String(),
	}
}

func (am *assetManager) GetTokenTransfers(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.TokenTransfer, *database.FilterResult, error) {
	return am.database.GetTokenTransfers(ctx, filter)
}

func (am *assetManager) GetTokenTransfersByPool(ctx context.Context, ns, typeName, name string, filter database.AndFilter) ([]*fftypes.TokenTransfer, *database.FilterResult, error) {
	pool, err := am.GetTokenPool(ctx, ns, typeName, name)
	if err != nil {
		return nil, nil, err
	}
	return am.database.GetTokenTransfers(ctx, filter.Condition(filter.Builder().Eq("poolprotocolid", pool.ProtocolID)))
}

func (am *assetManager) NewTransfer(ns, typeName, poolName string, transfer *fftypes.TokenTransferInput) sysmessaging.MessageSender {
	sender := &transferSender{
		mgr:       am,
		namespace: ns,
		typeName:  typeName,
		poolName:  poolName,
		transfer:  transfer,
	}
	sender.setDefaults()
	return sender
}

type transferSender struct {
	mgr          *assetManager
	namespace    string
	typeName     string
	poolName     string
	transfer     *fftypes.TokenTransferInput
	sendCallback sysmessaging.BeforeSendCallback
}

func (s *transferSender) Send(ctx context.Context) error {
	return s.resolveAndSend(ctx, false)
}

func (s *transferSender) SendAndWait(ctx context.Context) error {
	return s.resolveAndSend(ctx, true)
}

func (s *transferSender) BeforeSend(cb sysmessaging.BeforeSendCallback) sysmessaging.MessageSender {
	s.sendCallback = cb
	return s
}

func (s *transferSender) setDefaults() {
	s.transfer.LocalID = fftypes.NewUUID()
}

func (am *assetManager) MintTokens(ctx context.Context, ns, typeName, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
	transfer.Type = fftypes.TokenTransferTypeMint
	if transfer.Key == "" {
		org, err := am.identity.GetLocalOrganization(ctx)
		if err != nil {
			return nil, err
		}
		transfer.Key = org.Identity
	}
	transfer.From = ""
	if transfer.To == "" {
		transfer.To = transfer.Key
	}

	sender := am.NewTransfer(ns, typeName, poolName, transfer)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (am *assetManager) BurnTokens(ctx context.Context, ns, typeName, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
	transfer.Type = fftypes.TokenTransferTypeBurn
	if transfer.Key == "" {
		org, err := am.identity.GetLocalOrganization(ctx)
		if err != nil {
			return nil, err
		}
		transfer.Key = org.Identity
	}
	if transfer.From == "" {
		transfer.From = transfer.Key
	}
	transfer.To = ""

	sender := am.NewTransfer(ns, typeName, poolName, transfer)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (am *assetManager) TransferTokens(ctx context.Context, ns, typeName, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
	transfer.Type = fftypes.TokenTransferTypeTransfer
	if transfer.Key == "" {
		org, err := am.identity.GetLocalOrganization(ctx)
		if err != nil {
			return nil, err
		}
		transfer.Key = org.Identity
	}
	if transfer.From == "" {
		transfer.From = transfer.Key
	}
	if transfer.To == "" {
		transfer.To = transfer.Key
	}
	if transfer.From == transfer.To {
		return nil, i18n.NewError(ctx, i18n.MsgCannotTransferToSelf)
	}

	sender := am.NewTransfer(ns, typeName, poolName, transfer)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (s *transferSender) resolveAndSend(ctx context.Context, waitConfirm bool) (err error) {
	plugin, err := s.mgr.selectTokenPlugin(ctx, s.typeName)
	if err != nil {
		return err
	}
	pool, err := s.mgr.GetTokenPool(ctx, s.namespace, s.typeName, s.poolName)
	if err != nil {
		return err
	}
	s.transfer.PoolProtocolID = pool.ProtocolID

	var messageSender sysmessaging.MessageSender
	if s.transfer.Message != nil {
		if messageSender, err = s.buildTransferMessage(ctx, s.namespace, s.transfer.Message); err != nil {
			return err
		}
	}

	switch {
	case waitConfirm && messageSender != nil:
		// prepare the message, send the transfer async, then send the message and wait
		return messageSender.
			BeforeSend(func(ctx context.Context) error {
				s.transfer.MessageHash = s.transfer.Message.Hash
				s.transfer.Message = nil
				return s.Send(ctx)
			}).
			SendAndWait(ctx)
	case waitConfirm:
		// no message - just send the transfer and wait
		return s.sendSync(ctx)
	case messageSender != nil:
		// send the message async and then move on to the transfer
		if err := messageSender.Send(ctx); err != nil {
			return err
		}
		s.transfer.MessageHash = s.transfer.Message.Hash
	}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: s.namespace,
			Type:      fftypes.TransactionTypeTokenTransfer,
			Signer:    s.transfer.Key,
			Reference: s.transfer.LocalID,
		},
		Created: fftypes.Now(),
		Status:  fftypes.OpStatusPending,
	}
	tx.Hash = tx.Subject.Hash()
	err = s.mgr.database.UpsertTransaction(ctx, tx, false /* should be new, or idempotent replay */)
	if err != nil {
		return err
	}

	s.transfer.TX.ID = tx.ID
	s.transfer.TX.Type = tx.Subject.Type

	op := fftypes.NewTXOperation(
		plugin,
		s.namespace,
		tx.ID,
		"",
		fftypes.OpTypeTokenTransfer,
		fftypes.OpStatusPending,
		"")
	addTokenTransferInputs(op, &s.transfer.TokenTransfer)
	if err := s.mgr.database.UpsertOperation(ctx, op, false); err != nil {
		return err
	}

	if s.sendCallback != nil {
		if err := s.sendCallback(ctx); err != nil {
			return err
		}
	}

	switch s.transfer.Type {
	case fftypes.TokenTransferTypeMint:
		return plugin.MintTokens(ctx, op.ID, &s.transfer.TokenTransfer)
	case fftypes.TokenTransferTypeTransfer:
		return plugin.TransferTokens(ctx, op.ID, &s.transfer.TokenTransfer)
	case fftypes.TokenTransferTypeBurn:
		return plugin.BurnTokens(ctx, op.ID, &s.transfer.TokenTransfer)
	default:
		panic(fmt.Sprintf("unknown transfer type: %v", s.transfer.Type))
	}
}

func (s *transferSender) sendSync(ctx context.Context) error {
	out, err := s.mgr.syncasync.SendConfirmTokenTransfer(ctx, s.namespace, s.transfer.LocalID, s.Send)
	if out != nil {
		s.transfer.TokenTransfer = *out
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
