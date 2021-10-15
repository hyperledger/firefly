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

	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
)

type Manager interface {
	CreateTokenPool(ctx context.Context, ns, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error)
	GetTokenPools(ctx context.Context, ns, typeName string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error)
	GetTokenPool(ctx context.Context, ns, typeName, poolName string) (*fftypes.TokenPool, error)
	GetTokenAccounts(ctx context.Context, ns, typeName, poolName string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error)
	ValidateTokenPoolTx(ctx context.Context, pool *fftypes.TokenPool, protocolTxID string) error
	GetTokenTransfers(ctx context.Context, ns, typeName, poolName string, filter database.AndFilter) ([]*fftypes.TokenTransfer, *database.FilterResult, error)
	NewTransfer(ns, typeName, poolName string, transfer *fftypes.TokenTransferInput) sysmessaging.MessageSender
	MintTokens(ctx context.Context, ns, typeName, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	BurnTokens(ctx context.Context, ns, typeName, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)
	TransferTokens(ctx context.Context, ns, typeName, poolName string, transfer *fftypes.TokenTransferInput, waitConfirm bool) (*fftypes.TokenTransfer, error)

	// Bound token callbacks
	TokenPoolCreated(tk tokens.Plugin, pool *fftypes.TokenPool, protocolTxID string, additionalInfo fftypes.JSONObject) error

	Start() error
	WaitStop()
}

type assetManager struct {
	ctx       context.Context
	database  database.Plugin
	identity  identity.Manager
	data      data.Manager
	syncasync syncasync.Bridge
	broadcast broadcast.Manager
	messaging privatemessaging.Manager
	tokens    map[string]tokens.Plugin
	retry     retry.Retry
	txhelper  txcommon.Helper
}

func NewAssetManager(ctx context.Context, di database.Plugin, im identity.Manager, dm data.Manager, sa syncasync.Bridge, bm broadcast.Manager, pm privatemessaging.Manager, ti map[string]tokens.Plugin) (Manager, error) {
	if di == nil || im == nil || sa == nil || bm == nil || pm == nil || ti == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	am := &assetManager{
		ctx:       ctx,
		database:  di,
		identity:  im,
		data:      dm,
		syncasync: sa,
		broadcast: bm,
		messaging: pm,
		tokens:    ti,
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.AssetManagerRetryInitialDelay),
			MaximumDelay: config.GetDuration(config.AssetManagerRetryMaxDelay),
			Factor:       config.GetFloat64(config.AssetManagerRetryFactor),
		},
		txhelper: txcommon.NewTransactionHelper(di),
	}
	return am, nil
}

func (am *assetManager) selectTokenPlugin(ctx context.Context, name string) (tokens.Plugin, error) {
	for pluginName, plugin := range am.tokens {
		if pluginName == name {
			return plugin, nil
		}
	}
	return nil, i18n.NewError(ctx, i18n.MsgUnknownTokensPlugin, name)
}

func addTokenPoolCreateInputs(op *fftypes.Operation, pool *fftypes.TokenPool) {
	op.Input = fftypes.JSONObject{
		"id":        pool.ID.String(),
		"namespace": pool.Namespace,
		"name":      pool.Name,
		"config":    pool.Config,
	}
}

func retrieveTokenPoolCreateInputs(ctx context.Context, op *fftypes.Operation, pool *fftypes.TokenPool) (err error) {
	input := &op.Input
	pool.ID, err = fftypes.ParseUUID(ctx, input.GetString("id"))
	if err != nil {
		return err
	}
	pool.Namespace = input.GetString("namespace")
	pool.Name = input.GetString("name")
	if pool.Namespace == "" || pool.Name == "" {
		return fmt.Errorf("namespace or name missing from inputs")
	}
	pool.Config = input.GetObject("config")
	return nil
}

// Note: the counterpart to below (retrieveTokenTransferInputs) lives in the events package
func addTokenTransferInputs(op *fftypes.Operation, transfer *fftypes.TokenTransfer) {
	op.Input = fftypes.JSONObject{
		"id": transfer.LocalID.String(),
	}
}

func (am *assetManager) CreateTokenPool(ctx context.Context, ns string, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	return am.createTokenPoolWithID(ctx, fftypes.NewUUID(), ns, typeName, pool, waitConfirm)
}

func (am *assetManager) createTokenPoolWithID(ctx context.Context, id *fftypes.UUID, ns string, typeName string, pool *fftypes.TokenPool, waitConfirm bool) (*fftypes.TokenPool, error) {
	if err := am.data.VerifyNamespaceExists(ctx, ns); err != nil {
		return nil, err
	}

	if pool.Key == "" {
		org, err := am.identity.GetLocalOrganization(ctx)
		if err != nil {
			return nil, err
		}
		pool.Key = org.Identity
	}

	plugin, err := am.selectTokenPlugin(ctx, typeName)
	if err != nil {
		return nil, err
	}

	if waitConfirm {
		requestID := fftypes.NewUUID()
		return am.syncasync.SendConfirmTokenPool(ctx, ns, requestID, func() error {
			_, err := am.createTokenPoolWithID(ctx, requestID, ns, typeName, pool, false)
			return err
		})
	}

	tx := &fftypes.Transaction{
		ID: fftypes.NewUUID(),
		Subject: fftypes.TransactionSubject{
			Namespace: ns,
			Type:      fftypes.TransactionTypeTokenPool,
			Signer:    pool.Key,
			Reference: id,
		},
		Created: fftypes.Now(),
		Status:  fftypes.OpStatusPending,
	}
	tx.Hash = tx.Subject.Hash()
	err = am.database.UpsertTransaction(ctx, tx, false /* should be new, or idempotent replay */)
	if err != nil {
		return nil, err
	}

	pool.ID = id
	pool.Namespace = ns
	pool.TX = fftypes.TransactionRef{
		ID:   tx.ID,
		Type: tx.Subject.Type,
	}

	op := fftypes.NewTXOperation(
		plugin,
		ns,
		tx.ID,
		"",
		fftypes.OpTypeTokenCreatePool,
		fftypes.OpStatusPending,
		"")
	addTokenPoolCreateInputs(op, pool)
	err = am.database.UpsertOperation(ctx, op, false)
	if err != nil {
		return nil, err
	}

	return pool, plugin.CreateTokenPool(ctx, op.ID, pool)
}

func (am *assetManager) scopeNS(ns string, filter database.AndFilter) database.AndFilter {
	return filter.Condition(filter.Builder().Eq("namespace", ns))
}

func (am *assetManager) GetTokenPools(ctx context.Context, ns string, typeName string, filter database.AndFilter) ([]*fftypes.TokenPool, *database.FilterResult, error) {
	if _, err := am.selectTokenPlugin(ctx, typeName); err != nil {
		return nil, nil, err
	}
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, nil, err
	}
	return am.database.GetTokenPools(ctx, am.scopeNS(ns, filter))
}

func (am *assetManager) GetTokenPool(ctx context.Context, ns, typeName, poolName string) (*fftypes.TokenPool, error) {
	if _, err := am.selectTokenPlugin(ctx, typeName); err != nil {
		return nil, err
	}
	if err := fftypes.ValidateFFNameField(ctx, ns, "namespace"); err != nil {
		return nil, err
	}
	if err := fftypes.ValidateFFNameField(ctx, poolName, "name"); err != nil {
		return nil, err
	}
	pool, err := am.database.GetTokenPool(ctx, ns, poolName)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, i18n.NewError(ctx, i18n.Msg404NotFound)
	}
	return pool, nil
}

func (am *assetManager) GetTokenAccounts(ctx context.Context, ns, typeName, poolName string, filter database.AndFilter) ([]*fftypes.TokenAccount, *database.FilterResult, error) {
	pool, err := am.GetTokenPool(ctx, ns, typeName, poolName)
	if err != nil {
		return nil, nil, err
	}
	return am.database.GetTokenAccounts(ctx, filter.Condition(filter.Builder().Eq("poolprotocolid", pool.ProtocolID)))
}

func (am *assetManager) ValidateTokenPoolTx(ctx context.Context, pool *fftypes.TokenPool, protocolTxID string) error {
	// TODO: validate that the given token pool was created with the given protocolTxId
	return nil
}

func (am *assetManager) GetTokenTransfers(ctx context.Context, ns, typeName, name string, filter database.AndFilter) ([]*fftypes.TokenTransfer, *database.FilterResult, error) {
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
	sealCallback sysmessaging.SealCallback
}

func (s *transferSender) Send(ctx context.Context) error {
	return s.resolveAndSend(ctx, false)
}

func (s *transferSender) SendAndWait(ctx context.Context) error {
	return s.resolveAndSend(ctx, true)
}

func (s *transferSender) AfterSeal(cb sysmessaging.SealCallback) sysmessaging.MessageSender {
	s.sealCallback = cb
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

	if waitConfirm {
		return s.sendSync(ctx)
	}

	if s.transfer.Message != nil {
		if err := s.sendTransferMessage(ctx, s.namespace, s.transfer.Message); err != nil {
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

	if s.sealCallback != nil {
		if err := s.sealCallback(ctx); err != nil {
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
	out, err := s.mgr.syncasync.SendConfirmTokenTransfer(ctx, s.namespace, s.transfer.LocalID, func() error {
		return s.Send(ctx)
	})
	if out != nil {
		s.transfer.TokenTransfer = *out
	}
	return err
}

func (s *transferSender) sendTransferMessage(ctx context.Context, ns string, in *fftypes.MessageInOut) error {
	allowedTypes := []fftypes.FFEnum{
		fftypes.MessageTypeTransferBroadcast,
		fftypes.MessageTypeTransferPrivate,
	}
	if in.Header.Type == "" {
		in.Header.Type = fftypes.MessageTypeTransferBroadcast
	}
	switch in.Header.Type {
	case fftypes.MessageTypeTransferBroadcast:
		return s.mgr.broadcast.NewBroadcast(ns, in).Send(ctx)
	case fftypes.MessageTypeTransferPrivate:
		return s.mgr.messaging.NewMessage(ns, in).Send(ctx)
	default:
		return i18n.NewError(ctx, i18n.MsgInvalidMessageType, allowedTypes)
	}
}

func (am *assetManager) Start() error {
	return nil
}

func (am *assetManager) WaitStop() {
	// No go routines
}
