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

package privatemessaging

import (
	"context"
	"encoding/json"

	"github.com/kaleido-io/firefly/internal/batch"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/data"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/blockchain"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/dataexchange"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/kaleido-io/firefly/pkg/identity"
	"github.com/karlseguin/ccache"
)

type Manager interface {
	GroupManager
	dataexchange.Callbacks
}

type privateMessaging struct {
	groupManager

	ctx          context.Context
	database     database.Plugin
	identity     identity.Plugin
	exchange     dataexchange.Plugin
	blockchain   blockchain.Plugin
	batch        batch.Manager
	data         data.Manager
	nodeIdentity string
}

func NewPrivateMessaging(ctx context.Context, di database.Plugin, ii identity.Plugin, dx dataexchange.Plugin, bi blockchain.Plugin, ba batch.Manager, dm data.Manager) (Manager, error) {
	if di == nil || ii == nil || dx == nil || bi == nil || ba == nil || dm == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}

	pm := &privateMessaging{
		ctx:        ctx,
		database:   di,
		identity:   ii,
		exchange:   dx,
		blockchain: bi,
		batch:      ba,
		data:       dm,
		groupManager: groupManager{
			database:      di,
			data:          dm,
			groupCacheTTL: config.GetDuration(config.GroupCacheTTL),
		},
		nodeIdentity: config.GetString(config.NodeIdentity),
	}
	pm.groupManager.groupCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(config.GroupCacheSize)),
	)

	bo := batch.Options{
		BatchMaxSize:   config.GetUint(config.PrivateBatchSize),
		BatchTimeout:   config.GetDuration(config.PrivateBatchTimeout),
		DisposeTimeout: config.GetDuration(config.PrivateBatchAgentTimeout),
	}

	ba.RegisterDispatcher([]fftypes.MessageType{
		fftypes.MessageTypeGroupInit,
		fftypes.MessageTypePrivate,
	}, pm.dispatchBatch, bo)

	return pm, nil
}

func (pm *privateMessaging) dispatchBatch(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {

	// Serialize the full payload, which has already been sealed for us by the BatchManager
	payload, err := json.Marshal(batch)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Retrieve the group
	nodes, err := pm.groupManager.getGroupNodes(ctx, batch.Group)
	if err != nil {
		return err
	}

	return pm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return pm.sendAndSubmitBatch(ctx, batch, nodes, payload, contexts)
	})
}

func (pm *privateMessaging) sendAndSubmitBatch(ctx context.Context, batch *fftypes.Batch, nodes []*fftypes.Node, payload fftypes.Byteable, contexts []*fftypes.Bytes32) (err error) {
	l := log.L(ctx)

	id, err := pm.identity.Resolve(ctx, batch.Author)
	if err == nil {
		err = pm.blockchain.VerifyIdentitySyntax(ctx, id)
	}
	if err != nil {
		log.L(ctx).Errorf("Invalid signing identity '%s': %s", batch.Author, err)
		return err
	}

	// Write it to the dataexchange for each member
	for i, node := range nodes {
		l.Infof("Sending batch %s:%s to group=%s node=%s (%d/%d)", batch.Namespace, batch.ID, batch.Group, node.ID, i+1, len(nodes))

		trackingID, err := pm.exchange.SendMessage(ctx, node, payload)
		if err != nil {
			return err
		}

		op := fftypes.NewTXOperation(
			pm.exchange,
			batch.Payload.TX.ID,
			trackingID,
			fftypes.OpTypeDataExchangeBatchSend,
			fftypes.OpStatusPending,
			node.ID.String())
		if err = pm.database.UpsertOperation(ctx, op, false); err != nil {
			return err
		}

	}

	return pm.writeTransaction(ctx, id, batch, contexts)
}

func (pm *privateMessaging) writeTransaction(ctx context.Context, signingID *fftypes.Identity, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {

	tx := &fftypes.Transaction{
		ID: batch.Payload.TX.ID,
		Subject: fftypes.TransactionSubject{
			Type:      fftypes.TransactionTypeBatchPin,
			Signer:    signingID.OnChain,
			Namespace: batch.Namespace,
			Reference: batch.ID,
		},
		Created: fftypes.Now(),
		Status:  fftypes.OpStatusPending,
	}
	tx.Hash = tx.Subject.Hash()
	err := pm.database.UpsertTransaction(ctx, tx, true, false /* should be new, or idempotent replay */)
	if err != nil {
		return err
	}

	// Write the batch pin to the blockchain
	blockchainTrackingID, err := pm.blockchain.SubmitBatchPin(ctx, nil, signingID, &blockchain.BatchPin{
		Namespace:      batch.Namespace,
		TransactionID:  batch.Payload.TX.ID,
		BatchID:        batch.ID,
		BatchPaylodRef: batch.PayloadRef,
		BatchHash:      batch.Hash,
		Contexts:       contexts,
	})
	if err != nil {
		return err
	}

	// The pending blockchain transaction
	op := fftypes.NewTXOperation(
		pm.blockchain,
		batch.Payload.TX.ID,
		blockchainTrackingID,
		fftypes.OpTypeBlockchainBatchPin,
		fftypes.OpStatusPending,
		"")
	if err := pm.database.UpsertOperation(ctx, op, false); err != nil {
		return err
	}

	return pm.database.UpsertOperation(ctx, op, false)
}
