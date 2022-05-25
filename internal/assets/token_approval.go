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

func (am *assetManager) GetTokenApprovals(ctx context.Context, ns string, filter database.AndFilter) ([]*core.TokenApproval, *database.FilterResult, error) {
	return am.database.GetTokenApprovals(ctx, am.scopeNS(ns, filter))
}

type approveSender struct {
	mgr       *assetManager
	namespace string
	approval  *core.TokenApprovalInput
}

func (s *approveSender) Prepare(ctx context.Context) error {
	return s.sendInternal(ctx, methodPrepare)
}

func (s *approveSender) Send(ctx context.Context) error {
	return s.sendInternal(ctx, methodSend)
}

func (s *approveSender) SendAndWait(ctx context.Context) error {
	return s.sendInternal(ctx, methodSendAndWait)
}

func (s *approveSender) setDefaults() {
	s.approval.LocalID = fftypes.NewUUID()
}

func (am *assetManager) NewApproval(ns string, approval *core.TokenApprovalInput) sysmessaging.MessageSender {
	sender := &approveSender{
		mgr:       am,
		namespace: ns,
		approval:  approval,
	}
	sender.setDefaults()
	return sender
}

func (am *assetManager) TokenApproval(ctx context.Context, ns string, approval *core.TokenApprovalInput, waitConfirm bool) (out *core.TokenApproval, err error) {
	sender := am.NewApproval(ns, approval)
	if waitConfirm {
		err = sender.SendAndWait(ctx)
	} else {
		err = sender.Send(ctx)
	}
	return &approval.TokenApproval, err
}

func (s *approveSender) sendInternal(ctx context.Context, method sendMethod) (err error) {
	if method == methodSendAndWait {
		out, err := s.mgr.syncasync.WaitForTokenApproval(ctx, s.namespace, s.approval.LocalID, s.Send)
		if out != nil {
			s.approval.TokenApproval = *out
		}
		return err
	}

	var op *core.Operation
	var pool *core.TokenPool
	err = s.mgr.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		pool, err = s.mgr.validateApproval(ctx, s.namespace, s.approval)
		if err != nil {
			return err
		}

		plugin, err := s.mgr.selectTokenPlugin(ctx, s.approval.Connector)
		if err != nil {
			return err
		}

		if method == methodPrepare {
			return nil
		}

		txid, err := s.mgr.txHelper.SubmitNewTransaction(ctx, s.namespace, core.TransactionTypeTokenApproval)
		if err != nil {
			return err
		}
		s.approval.TX.ID = txid
		s.approval.TX.Type = core.TransactionTypeTokenApproval

		op = core.NewOperation(
			plugin,
			s.namespace,
			txid,
			core.TransactionTypeTokenApproval)
		if err = txcommon.AddTokenApprovalInputs(op, &s.approval.TokenApproval); err == nil {
			err = s.mgr.database.InsertOperation(ctx, op)
		}
		return err
	})
	if err != nil {
		return err
	} else if method == methodPrepare {
		return nil
	}

	_, err = s.mgr.operations.RunOperation(ctx, opApproval(op, pool, &s.approval.TokenApproval))
	return err
}

func (am *assetManager) validateApproval(ctx context.Context, ns string, approval *core.TokenApprovalInput) (pool *core.TokenPool, err error) {
	if approval.Pool == "" {
		pool, err = am.getDefaultTokenPool(ctx, ns)
		if err != nil {
			return nil, err
		}
	} else {
		pool, err = am.GetTokenPoolByNameOrID(ctx, ns, approval.Pool)
		if err != nil {
			return nil, err
		}
	}
	approval.TokenApproval.Pool = pool.ID
	approval.TokenApproval.Connector = pool.Connector

	if pool.State != core.TokenPoolStateConfirmed {
		return nil, i18n.NewError(ctx, coremsgs.MsgTokenPoolNotConfirmed)
	}
	approval.Key, err = am.identity.NormalizeSigningKey(ctx, approval.Key, am.keyNormalization)
	return pool, err
}
