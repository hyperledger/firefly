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

	"github.com/hyperledger-labs/firefly/internal/batch"
	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/data"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/retry"
	"github.com/hyperledger-labs/firefly/pkg/blockchain"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/dataexchange"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/identity"
	"github.com/karlseguin/ccache"
)

type Manager interface {
	GroupManager

	Start() error
	SendMessage(ctx context.Context, ns string, in *fftypes.MessageInput) (out *fftypes.Message, err error)
	SendMessageWithID(ctx context.Context, ns string, in *fftypes.MessageInput) (out *fftypes.Message, err error)
}

type privateMessaging struct {
	groupManager

	ctx                  context.Context
	database             database.Plugin
	identity             identity.Plugin
	exchange             dataexchange.Plugin
	blockchain           blockchain.Plugin
	batch                batch.Manager
	data                 data.Manager
	retry                retry.Retry
	localNodeName        string
	localOrgIdentity     string
	opCorrelationRetries int
}

func NewPrivateMessaging(ctx context.Context, di database.Plugin, ii identity.Plugin, dx dataexchange.Plugin, bi blockchain.Plugin, ba batch.Manager, dm data.Manager) (Manager, error) {
	if di == nil || ii == nil || dx == nil || bi == nil || ba == nil || dm == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}

	pm := &privateMessaging{
		ctx:              ctx,
		database:         di,
		identity:         ii,
		exchange:         dx,
		blockchain:       bi,
		batch:            ba,
		data:             dm,
		localNodeName:    config.GetString(config.NodeName),
		localOrgIdentity: config.GetString(config.OrgIdentity),
		groupManager: groupManager{
			database:      di,
			data:          dm,
			groupCacheTTL: config.GetDuration(config.GroupCacheTTL),
		},
		retry: retry.Retry{
			InitialDelay: config.GetDuration(config.PrivateMessagingRetryInitDelay),
			MaximumDelay: config.GetDuration(config.PrivateMessagingRetryMaxDelay),
			Factor:       config.GetFloat64(config.PrivateMessagingRetryFactor),
		},
		opCorrelationRetries: config.GetInt(config.PrivateMessagingOpCorrelationRetries),
	}
	pm.groupManager.groupCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(config.GroupCacheSize)),
	)

	bo := batch.Options{
		BatchMaxSize:   config.GetUint(config.PrivateMessagingBatchSize),
		BatchTimeout:   config.GetDuration(config.PrivateMessagingBatchTimeout),
		DisposeTimeout: config.GetDuration(config.PrivateMessagingBatchAgentTimeout),
	}

	ba.RegisterDispatcher([]fftypes.MessageType{
		fftypes.MessageTypeGroupInit,
		fftypes.MessageTypePrivate,
	}, pm.dispatchBatch, bo)

	return pm, nil
}

func (pm *privateMessaging) Start() error {
	return pm.exchange.Start()
}

func (pm *privateMessaging) dispatchBatch(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {

	// Serialize the full payload, which has already been sealed for us by the BatchManager
	payload, err := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Retrieve the group
	_, nodes, err := pm.groupManager.getGroupNodes(ctx, batch.Group)
	if err != nil {
		return err
	}

	return pm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return pm.sendAndSubmitBatch(ctx, batch, nodes, payload, contexts)
	})
}

func (pm *privateMessaging) transferBlobs(ctx context.Context, data []*fftypes.Data, txid *fftypes.UUID, node *fftypes.Node) error {
	// Send all the blobs associated with this batch
	for _, d := range data {
		// We only need to send a blob if there is one, and it's not been uploaded to the public storage
		if d.Blob != nil && d.Blob.Hash != nil && d.Blob.Public == "" {
			blob, err := pm.database.GetBlobMatchingHash(ctx, d.Blob.Hash)
			if err != nil {
				return err
			}
			if blob == nil {
				return i18n.NewError(ctx, i18n.MsgBlobNotFound, d.Blob)
			}

			trackingID, err := pm.exchange.TransferBLOB(ctx, node.DX.Peer, blob.PayloadRef)
			if err != nil {
				return err
			}

			if txid != nil {
				op := fftypes.NewTXOperation(
					pm.exchange,
					d.Namespace,
					txid,
					trackingID,
					fftypes.OpTypeDataExchangeBlobSend,
					fftypes.OpStatusPending,
					node.ID.String())
				if err = pm.database.UpsertOperation(ctx, op, false); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (pm *privateMessaging) sendData(ctx context.Context, mType string, mID *fftypes.UUID, group *fftypes.Bytes32, ns string, nodes []*fftypes.Node, payload fftypes.Byteable, txid *fftypes.UUID, data []*fftypes.Data) (err error) {
	l := log.L(ctx)

	// Write it to the dataexchange for each member
	for i, node := range nodes {

		if node.Owner == pm.localOrgIdentity {
			l.Debugf("Skipping send of %s for local node %s:%s for group=%s node=%s (%d/%d)", mType, ns, mID, group, node.ID, i+1, len(nodes))
			continue
		}

		l.Debugf("Sending %s %s:%s to group=%s node=%s (%d/%d)", mType, ns, mID, group, node.ID, i+1, len(nodes))

		// Initiate transfer of any blobs first
		if err = pm.transferBlobs(ctx, data, txid, node); err != nil {
			return err
		}

		// Send the payload itself
		trackingID, err := pm.exchange.SendMessage(ctx, node.DX.Peer, payload)
		if err != nil {
			return err
		}

		if txid != nil {
			op := fftypes.NewTXOperation(
				pm.exchange,
				ns,
				txid,
				trackingID,
				fftypes.OpTypeDataExchangeBatchSend,
				fftypes.OpStatusPending,
				node.ID.String())
			if err = pm.database.UpsertOperation(ctx, op, false); err != nil {
				return err
			}
		}

	}

	return nil
}

func (pm *privateMessaging) sendAndSubmitBatch(ctx context.Context, batch *fftypes.Batch, nodes []*fftypes.Node, payload fftypes.Byteable, contexts []*fftypes.Bytes32) (err error) {
	id, err := pm.identity.Resolve(ctx, batch.Author)
	if err == nil {
		err = pm.blockchain.VerifyIdentitySyntax(ctx, id)
	}
	if err != nil {
		log.L(ctx).Errorf("Invalid signing identity '%s': %s", batch.Author, err)
		return err
	}

	if err = pm.sendData(ctx, "batch", batch.ID, batch.Group, batch.Namespace, nodes, payload, batch.Payload.TX.ID, batch.Payload.Data); err != nil {
		return err
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
		batch.Namespace,
		batch.Payload.TX.ID,
		blockchainTrackingID,
		fftypes.OpTypeBlockchainBatchPin,
		fftypes.OpStatusPending,
		"")

	return pm.database.UpsertOperation(ctx, op, false)
}
