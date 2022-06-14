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
	"sync"

	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-common/pkg/retry"
	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/batchpin"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/identity"
	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/operations"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/internal/sysmessaging"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/karlseguin/ccache"
)

const pinnedPrivateDispatcherName = "pinned_private"
const unpinnedPrivateDispatcherName = "unpinned_private"

type Manager interface {
	core.Named
	GroupManager

	Start() error
	NewMessage(ns string, msg *core.MessageInOut) sysmessaging.MessageSender
	SendMessage(ctx context.Context, ns string, in *core.MessageInOut, waitConfirm bool) (out *core.Message, err error)
	RequestReply(ctx context.Context, ns string, request *core.MessageInOut) (reply *core.MessageInOut, err error)

	// From operations.OperationHandler
	PrepareOperation(ctx context.Context, op *core.Operation) (*core.PreparedOperation, error)
	RunOperation(ctx context.Context, op *core.PreparedOperation) (outputs fftypes.JSONObject, complete bool, err error)
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
	maxBatchPayloadLength int64
	metrics               metrics.Manager
	operations            operations.Manager
	orgFirstNodes         map[fftypes.UUID]*core.Identity
}

type blobTransferTracker struct {
	dataID   *fftypes.UUID
	blobHash *fftypes.Bytes32
	op       *core.PreparedOperation
}

func NewPrivateMessaging(ctx context.Context, di database.Plugin, im identity.Manager, dx dataexchange.Plugin, bi blockchain.Plugin, ba batch.Manager, dm data.Manager, sa syncasync.Bridge, bp batchpin.Submitter, mm metrics.Manager, om operations.Manager) (Manager, error) {
	if di == nil || im == nil || dx == nil || bi == nil || ba == nil || dm == nil || mm == nil || om == nil {
		return nil, i18n.NewError(ctx, coremsgs.MsgInitializationNilDepError, "PrivateMessaging")
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
		localNodeName: config.GetString(coreconfig.NodeName),
		groupManager: groupManager{
			database:      di,
			data:          dm,
			groupCacheTTL: config.GetDuration(coreconfig.GroupCacheTTL),
		},
		retry: retry.Retry{
			InitialDelay: config.GetDuration(coreconfig.PrivateMessagingRetryInitDelay),
			MaximumDelay: config.GetDuration(coreconfig.PrivateMessagingRetryMaxDelay),
			Factor:       config.GetFloat64(coreconfig.PrivateMessagingRetryFactor),
		},
		maxBatchPayloadLength: config.GetByteSize(coreconfig.PrivateMessagingBatchPayloadLimit),
		metrics:               mm,
		operations:            om,
		orgFirstNodes:         make(map[fftypes.UUID]*core.Identity),
	}
	pm.groupManager.groupCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(coreconfig.GroupCacheSize)),
	)

	bo := batch.DispatcherOptions{
		BatchType:      core.BatchTypePrivate,
		BatchMaxSize:   config.GetUint(coreconfig.PrivateMessagingBatchSize),
		BatchMaxBytes:  pm.maxBatchPayloadLength,
		BatchTimeout:   config.GetDuration(coreconfig.PrivateMessagingBatchTimeout),
		DisposeTimeout: config.GetDuration(coreconfig.PrivateMessagingBatchAgentTimeout),
	}

	ba.RegisterDispatcher(pinnedPrivateDispatcherName,
		core.TransactionTypeBatchPin,
		[]core.MessageType{
			core.MessageTypeGroupInit,
			core.MessageTypePrivate,
			core.MessageTypeTransferPrivate,
		},
		pm.dispatchPinnedBatch, bo)

	ba.RegisterDispatcher(unpinnedPrivateDispatcherName,
		core.TransactionTypeUnpinned,
		[]core.MessageType{
			core.MessageTypePrivate,
		},
		pm.dispatchUnpinnedBatch, bo)

	om.RegisterHandler(ctx, pm, []core.OpType{
		core.OpTypeDataExchangeSendBlob,
		core.OpTypeDataExchangeSendBatch,
	})

	return pm, nil
}

func (pm *privateMessaging) Name() string {
	return "PrivateMessaging"
}

func (pm *privateMessaging) Start() error {
	return pm.exchange.Start()
}

func (pm *privateMessaging) dispatchPinnedBatch(ctx context.Context, state *batch.DispatchState) error {
	err := pm.dispatchBatchCommon(ctx, state)
	if err != nil {
		return err
	}

	log.L(ctx).Infof("Pinning private batch %s with author=%s key=%s group=%s", state.Persisted.ID, state.Persisted.Author, state.Persisted.Key, state.Persisted.Group)
	return pm.batchpin.SubmitPinnedBatch(ctx, &state.Persisted, state.Pins, "" /* no payloadRef for private */)
}

func (pm *privateMessaging) dispatchUnpinnedBatch(ctx context.Context, state *batch.DispatchState) error {
	return pm.dispatchBatchCommon(ctx, state)
}

func (pm *privateMessaging) dispatchBatchCommon(ctx context.Context, state *batch.DispatchState) error {
	batch := state.Persisted.GenInflight(state.Messages, state.Data)
	tw := &core.TransportWrapper{
		Batch: batch,
	}

	// Retrieve the group
	group, nodes, err := pm.groupManager.getGroupNodes(ctx, batch.Group, false /* fail if not found */)
	if err != nil {
		return err
	}

	if batch.Payload.TX.Type == core.TransactionTypeUnpinned {
		// In the case of an un-pinned message we cannot be sure the group has been broadcast via the blockchain.
		// So we have to take the hit of sending it along with every message.
		tw.Group = group
	}

	return pm.sendData(ctx, tw, nodes)
}

func (pm *privateMessaging) prepareBlobTransfers(ctx context.Context, data core.DataArray, txid *fftypes.UUID, node *core.Identity) ([]*blobTransferTracker, error) {

	operations := make([]*blobTransferTracker, 0)

	// Build all the operations needed to send the blobs in a single DB transaction
	err := pm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		for _, d := range data {
			if d.Blob != nil {
				if d.Blob.Hash == nil {
					return i18n.NewError(ctx, coremsgs.MsgDataMissingBlobHash, d.ID)
				}

				blob, err := pm.database.GetBlobMatchingHash(ctx, d.Blob.Hash)
				if err != nil {
					return err
				}
				if blob == nil {
					return i18n.NewError(ctx, coremsgs.MsgBlobNotFound, d.Blob)
				}

				op := core.NewOperation(
					pm.exchange,
					d.Namespace,
					txid,
					core.OpTypeDataExchangeSendBlob)
				addTransferBlobInputs(op, node.ID, blob.Hash)
				if err = pm.operations.AddOrReuseOperation(ctx, op); err != nil {
					return err
				}

				operations = append(operations, &blobTransferTracker{
					dataID:   d.ID,
					blobHash: blob.Hash,
					op:       opSendBlob(op, node, blob),
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return operations, err
}

func (pm *privateMessaging) submitBlobTransfersToDX(ctx context.Context, trackers []*blobTransferTracker) error {
	// Initiate all the sends. We use parallel go routines here as these are blocking API calls
	wg := sync.WaitGroup{}
	wg.Add(len(trackers))
	var firstError error
	for _, tracker := range trackers {
		go func(tracker *blobTransferTracker) {
			defer wg.Done()
			log.L(ctx).Debugf("Initiating DX transfer blob=%s data=%s operation=%s", tracker.blobHash, tracker.dataID, tracker.op.ID)
			if _, err := pm.operations.RunOperation(ctx, tracker.op); err != nil {
				log.L(ctx).Errorf("Failed to initiate DX transfer blob=%s data=%s operation=%s", tracker.blobHash, tracker.dataID, tracker.op.ID)
				if firstError == nil {
					firstError = err
				}
			}
		}(tracker)
	}
	wg.Wait()
	return firstError
}

func (pm *privateMessaging) sendData(ctx context.Context, tw *core.TransportWrapper, nodes []*core.Identity) (err error) {
	l := log.L(ctx)
	batch := tw.Batch

	// Lookup the local org
	localOrg, err := pm.identity.GetMultipartyRootOrg(ctx)
	if err != nil {
		return err
	}

	// Write it to the dataexchange for each member
	for i, node := range nodes {

		if node.Parent.Equals(localOrg.ID) {
			l.Debugf("Skipping send of batch for local node %s:%s for group=%s node=%s (%d/%d)", batch.Namespace, batch.ID, batch.Group, node.ID, i+1, len(nodes))
			continue
		}

		l.Debugf("Sending batch %s:%s to group=%s node=%s (%d/%d)", batch.Namespace, batch.ID, batch.Group, node.ID, i+1, len(nodes))

		var blobTrackers []*blobTransferTracker
		var sendBatchOp *core.PreparedOperation

		// Use a DB group for preparing all the operations needed for this batch
		err := pm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
			blobTrackers, err = pm.prepareBlobTransfers(ctx, batch.Payload.Data, batch.Payload.TX.ID, node)
			if err != nil {
				return err
			}

			op := core.NewOperation(
				pm.exchange,
				batch.Namespace,
				batch.Payload.TX.ID,
				core.OpTypeDataExchangeSendBatch)
			addBatchSendInputs(op, node.ID, batch.Group, batch.ID)
			if err = pm.operations.AddOrReuseOperation(ctx, op); err != nil {
				return err
			}
			sendBatchOp = opSendBatch(op, node, tw)
			return nil
		})
		if err != nil {
			return err
		}

		// Initiate transfer of any blobs first
		if len(blobTrackers) > 0 {
			if err = pm.submitBlobTransfersToDX(ctx, blobTrackers); err != nil {
				return err
			}
		}

		// Then initiate the batch transfer
		if _, err = pm.operations.RunOperation(ctx, sendBatchOp); err != nil {
			return err
		}
	}

	return nil
}
