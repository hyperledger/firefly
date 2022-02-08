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

package privatemessaging

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/batchpin"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/retry"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/karlseguin/ccache"
)

type Manager interface {
	GroupManager

	Start() error
	NewMessage(ns string, msg *fftypes.MessageInOut) sysmessaging.MessageSender
	SendMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error)
	RequestReply(ctx context.Context, ns string, request *fftypes.MessageInOut) (reply *fftypes.MessageInOut, err error)
}

type privateMessaging struct {
	groupManager

	ctx                   context.Context
	database              database.Plugin
	identity              identity.Manager
	exchange              dataexchange.Plugin
	blockchain            blockchain.Plugin
	batch                 batch.Manager
	data                  data.Manager
	syncasync             syncasync.Bridge
	batchpin              batchpin.Submitter
	retry                 retry.Retry
	localNodeName         string
	localNodeID           *fftypes.UUID // lookup and cached on first use, as might not be registered at startup
	opCorrelationRetries  int
	maxBatchPayloadLength int64
}

func NewPrivateMessaging(ctx context.Context, di database.Plugin, im identity.Manager, dx dataexchange.Plugin, bi blockchain.Plugin, ba batch.Manager, dm data.Manager, sa syncasync.Bridge, bp batchpin.Submitter) (Manager, error) {
	if di == nil || im == nil || dx == nil || bi == nil || ba == nil || dm == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}

	pm := &privateMessaging{
		ctx:           ctx,
		database:      di,
		identity:      im,
		exchange:      dx,
		blockchain:    bi,
		batch:         ba,
		data:          dm,
		syncasync:     sa,
		batchpin:      bp,
		localNodeName: config.GetString(config.NodeName),
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
		opCorrelationRetries:  config.GetInt(config.PrivateMessagingOpCorrelationRetries),
		maxBatchPayloadLength: config.GetByteSize(config.PrivateMessagingBatchPayloadLimit),
	}
	pm.groupManager.groupCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(config.GroupCacheSize)),
	)

	bo := batch.Options{
		BatchMaxSize:   config.GetUint(config.PrivateMessagingBatchSize),
		BatchMaxBytes:  pm.maxBatchPayloadLength,
		BatchTimeout:   config.GetDuration(config.PrivateMessagingBatchTimeout),
		DisposeTimeout: config.GetDuration(config.PrivateMessagingBatchAgentTimeout),
	}

	ba.RegisterDispatcher([]fftypes.MessageType{
		fftypes.MessageTypeGroupInit,
		fftypes.MessageTypePrivate,
		fftypes.MessageTypeTransferPrivate,
	}, pm.dispatchBatch, bo)

	return pm, nil
}

func (pm *privateMessaging) Start() error {
	return pm.exchange.Start()
}

func (pm *privateMessaging) dispatchBatch(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {

	// Serialize the full payload, which has already been sealed for us by the BatchManager
	tw := &fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	}

	// Retrieve the group
	_, nodes, err := pm.groupManager.getGroupNodes(ctx, batch.Group)
	if err != nil {
		return err
	}

	return pm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return pm.sendAndSubmitBatch(ctx, batch, nodes, tw, contexts)
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

			opID := fftypes.NewUUID()
			if txid != nil {
				op := fftypes.NewOperation(
					pm.exchange,
					d.Namespace,
					txid,
					fftypes.OpTypeDataExchangeBlobSend)
				if err = pm.database.InsertOperation(ctx, op); err != nil {
					return err
				}
				opID = op.ID
			}

			if err := pm.exchange.TransferBLOB(ctx, opID, node.DX.Peer, blob.PayloadRef); err != nil {
				return err
			}
		}
	}
	return nil
}

func (pm *privateMessaging) sendData(ctx context.Context, mType string, mID *fftypes.UUID, group *fftypes.Bytes32, ns string, nodes []*fftypes.Node, tw *fftypes.TransportWrapper, txid *fftypes.UUID, data []*fftypes.Data) (err error) {
	l := log.L(ctx)

	payload, err := json.Marshal(tw)
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// TODO: move to using DIDs consistently as the way to reference the node/organization (i.e. node.Owner becomes a DID)
	localOrgSigingKey, err := pm.identity.GetLocalOrgKey(ctx)
	if err != nil {
		return err
	}

	// Write it to the dataexchange for each member
	for i, node := range nodes {

		if node.Owner == localOrgSigingKey {
			l.Debugf("Skipping send of %s for local node %s:%s for group=%s node=%s (%d/%d)", mType, ns, mID, group, node.ID, i+1, len(nodes))
			continue
		}

		l.Debugf("Sending %s %s:%s to group=%s node=%s (%d/%d)", mType, ns, mID, group, node.ID, i+1, len(nodes))

		// Initiate transfer of any blobs first
		if err = pm.transferBlobs(ctx, data, txid, node); err != nil {
			return err
		}

		opID := fftypes.NewUUID()
		if txid != nil {
			op := fftypes.NewOperation(
				pm.exchange,
				ns,
				txid,
				fftypes.OpTypeDataExchangeBatchSend)
			op.Input = fftypes.JSONObject{
				"manifest": tw.Manifest().String(),
			}
			if err = pm.database.InsertOperation(ctx, op); err != nil {
				return err
			}
			opID = op.ID
		}

		// Send the payload itself
		err := pm.exchange.SendMessage(ctx, opID, node.DX.Peer, payload)
		if err != nil {
			return err
		}
	}

	return nil
}

func (pm *privateMessaging) sendAndSubmitBatch(ctx context.Context, batch *fftypes.Batch, nodes []*fftypes.Node, tw *fftypes.TransportWrapper, contexts []*fftypes.Bytes32) (err error) {
	if err = pm.sendData(ctx, "batch", batch.ID, batch.Group, batch.Namespace, nodes, tw, batch.Payload.TX.ID, batch.Payload.Data); err != nil {
		return err
	}
	return pm.writeTransaction(ctx, batch, contexts)
}

func (pm *privateMessaging) writeTransaction(ctx context.Context, batch *fftypes.Batch, contexts []*fftypes.Bytes32) error {
	return pm.batchpin.SubmitPinnedBatch(ctx, batch, contexts)
}
