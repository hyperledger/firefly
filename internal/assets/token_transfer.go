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
	return am.database.GetTokenTransfers(ctx, filter.Condition(filter.Builder().Eq("poolprotocolid", pool.ProtocolID)))
}

func (am *assetManager) NewTransfer(ns, connector, poolName string, transfer *fftypes.TokenTransferInput) sysmessaging.MessageSender {
	sender := &transferSender{
		mgr:       am,
		namespace: ns,
		connector: connector,
		poolName:  poolName,
		transfer:  transfer,
	}
	sender.setDefaults()
	return sender
}

type transferSender struct {
	mgr       *assetManager
	namespace string
	connector string
	poolName  string
	transfer  *fftypes.TokenTransferInput
	resolved  bool
}

type sendMethod int

const (
	methodPrepare sendMethod = iota
	methodSend
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
	s.transfer.Connector = s.connector
}

func (am *assetManager) MintTokens(ctx context.Context, ns, connector, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
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

	sender := am.NewTransfer(ns, connector, poolName, transfer)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (am *assetManager) BurnTokens(ctx context.Context, ns, connector, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
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

	sender := am.NewTransfer(ns, connector, poolName, transfer)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (am *assetManager) TransferTokens(ctx context.Context, ns, connector, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (out *fftypes.TokenTransfer, err error) {
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

	sender := am.NewTransfer(ns, connector, poolName, transfer)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &transfer.TokenTransfer, err
}

func (s *transferSender) resolveAndSend(ctx context.Context, method sendMethod) (err error) {
	var messageSender sysmessaging.MessageSender
	if !s.resolved {
		if messageSender, err = s.resolve(ctx); err != nil {
			return err
		}
		s.resolved = true
	}

	if messageSender != nil {
		if method == methodSendAndWait {
			if err = s.sendInternal(ctx, method); err != nil {
				return err
			}
			return messageSender.SendAndWait(ctx)
		}

		if err := messageSender.Send(ctx); err != nil {
			return err
		}
	}
	return s.sendInternal(ctx, method)
}

func (s *transferSender) resolve(ctx context.Context) (sender sysmessaging.MessageSender, err error) {
	// Resolve the attached message
	if s.transfer.Message != nil {
		if sender, err = s.buildTransferMessage(ctx, s.namespace, s.transfer.Message); err != nil {
			return nil, err
		}
		if err = sender.Prepare(ctx); err != nil {
			return nil, err
		}
		s.transfer.MessageHash = s.transfer.Message.Hash
	}
	return sender, nil
}

func (s *transferSender) sendInternal(ctx context.Context, method sendMethod) error {
	if method == methodSendAndWait {
		out, err := s.mgr.syncasync.WaitForTokenTransfer(ctx, s.namespace, s.transfer.LocalID, s.Send)
		if out != nil {
			s.transfer.TokenTransfer = *out
		}
		return err
	}

	plugin, err := s.mgr.selectTokenPlugin(ctx, s.connector)
	if err != nil {
		return err
	}

	if method == methodPrepare {
		return nil
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

	err = s.mgr.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		pool, err := s.mgr.GetTokenPool(ctx, s.namespace, s.connector, s.poolName)
		if err != nil {
			return err
		}
		s.transfer.PoolProtocolID = pool.ProtocolID

		err = s.mgr.database.UpsertTransaction(ctx, tx, false /* should be new, or idempotent replay */)
		if err == nil {
			err = s.mgr.database.UpsertOperation(ctx, op, false)
		}
		return err
	})
	if err != nil {
		return err
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
