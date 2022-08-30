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

package events

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func sampleBatchTransfer(t *testing.T, txType core.TransactionType) (*core.Batch, *core.TransportWrapper) {
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypePrivate, txType, core.DataArray{data})
	b := &core.TransportWrapper{
		Batch: batch,
		Group: &core.Group{
			Hash: fftypes.NewRandB32(),
		},
	}
	return batch, b
}

func newTestOrg(name string) *core.Identity {
	identity := &core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      core.IdentityTypeOrg,
			Namespace: "ns1",
			Name:      name,
			Parent:    nil,
		},
	}
	identity.DID, _ = identity.GenerateDID(context.Background())
	return identity
}

func newTestNode(name string, owner *core.Identity) *core.Identity {
	identity := &core.Identity{
		IdentityBase: core.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      core.IdentityTypeNode,
			Namespace: "ns1",
			Name:      name,
			Parent:    owner.ID,
		},
		IdentityProfile: core.IdentityProfile{
			Profile: fftypes.JSONObject{
				"id":  fmt.Sprintf("%s-peer", name),
				"url": fmt.Sprintf("https://%s.example.com", name),
			},
		},
	}
	identity.DID, _ = identity.GenerateDID(context.Background())
	return identity
}

func newMessageReceivedNoAck(peerID string, transport *core.TransportWrapper) *dataexchangemocks.DXEvent {
	mde := &dataexchangemocks.DXEvent{}
	mde.On("MessageReceived").Return(&dataexchange.MessageReceived{
		PeerID:    peerID,
		Transport: transport,
	})
	mde.On("Type").Return(dataexchange.DXEventTypeMessageReceived).Maybe()
	return mde
}

func newMessageReceived(peerID string, transport *core.TransportWrapper, expectedManifest string) *dataexchangemocks.DXEvent {
	mde := newMessageReceivedNoAck(peerID, transport)
	mde.On("AckWithManifest", expectedManifest).Return()
	return mde
}

func newPrivateBlobReceivedNoAck(peerID string, hash *fftypes.Bytes32, size int64, payloadRef string) *dataexchangemocks.DXEvent {
	mde := &dataexchangemocks.DXEvent{}
	pathParts := strings.Split(payloadRef, "/")
	mde.On("PrivateBlobReceived").Return(&dataexchange.PrivateBlobReceived{
		Namespace:  pathParts[0],
		PeerID:     peerID,
		Hash:       *hash,
		Size:       size,
		PayloadRef: payloadRef,
	})
	mde.On("Type").Return(dataexchange.DXEventTypePrivateBlobReceived).Maybe()
	return mde
}

func newPrivateBlobReceived(peerID string, hash *fftypes.Bytes32, size int64, payloadRef string) *dataexchangemocks.DXEvent {
	mde := newPrivateBlobReceivedNoAck(peerID, hash, size, payloadRef)
	mde.On("Ack").Return()
	return mde
}

func TestUnknownEvent(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	done := make(chan struct{})
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx").Maybe()
	mde := &dataexchangemocks.DXEvent{}
	mde.On("Type").Return(dataexchange.DXEventType(99)).Maybe()
	mde.On("Ack").Run(func(args mock.Arguments) {
		close(done)
	})
	em.DXEvent(mdx, mde)
	<-done

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestPinnedReceiveOK(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	batch.Node = node1.ID

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(org1, false, nil)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)
	mim.On("ValidateNodeOwner", em.ctx, mock.Anything, mock.Anything).Return(true, nil)

	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})
	mdx.On("Name").Return("utdx").Maybe()
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	done := make(chan struct{})
	mde := newMessageReceivedNoAck("peer1", b)
	mde.On("AckWithManifest", batch.Payload.Manifest(batch.ID).String()).Run(func(args mock.Arguments) {
		close(done)
	})
	em.DXEvent(mdx, mde)
	<-done

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestMessageReceiveOkBadBatchIgnored(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypePrivate, core.TransactionTypeBatchPin, core.DataArray{data})
	batch.Payload.TX.Type = core.TransactionTypeTokenPool
	b := &core.TransportWrapper{
		Batch: batch,
		Group: &core.Group{
			Hash: fftypes.NewRandB32(),
		},
	}

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)

	mde := newMessageReceived("peer1", b, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceivePersistBatchError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error

	batch, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	batch.Node = node1.ID

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(org1, false, nil)
	mim.On("ValidateNodeOwner", em.ctx, mock.Anything, mock.Anything).Return(true, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceivedWrongNS(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	em.namespace.NetworkName = "ns2"

	_, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mde := newMessageReceived("peer1", b, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)

}

func TestMessageReceivedNonMultiparty(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	em.multiparty = nil

	_, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mde := newMessageReceived("peer1", b, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)

}

func TestMessageReceiveNodeLookupError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to stop retry

	groupID := fftypes.NewRandB32()
	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			Namespace: "ns1",
			Group:     groupID,
			Node:      fftypes.NewUUID(),
			SignerRef: core.SignerRef{
				Author: "org1",
			},
		},
		Payload: core.BatchPayload{
			TX: core.TransactionRef{
				Type: core.TransactionTypeUnpinned,
			},
		},
	}
	b := &core.TransportWrapper{
		Batch: batch,
	}

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(nil, fmt.Errorf("pop"))

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mim.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error so we need to break the loop

	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	b.Node = node1.ID
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(nil, true, fmt.Errorf("pop"))

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", tw)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mpm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	b.Node = node1.ID
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(nil, false, nil)

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	mde := newMessageReceived("peer1", tw, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotMatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	b.Node = node1.ID
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(newTestOrg("org2"), false, nil)
	mim.On("ValidateNodeOwner", em.ctx, mock.Anything, mock.Anything).Return(false, nil)

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	mde := newMessageReceived("peer1", tw, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestPrivateBlobReceivedTriggersRewindOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	hash := fftypes.NewRandB32()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobs", em.ctx, mock.Anything).Return([]*core.Blob{}, nil, nil)
	mdi.On("InsertBlobs", em.ctx, mock.Anything).Return(nil)

	done := make(chan struct{})
	mde := newPrivateBlobReceivedNoAck("peer1", hash, 12345, "ns1/path1")
	mde.On("Ack").Run(func(args mock.Arguments) {
		close(done)
	})
	em.DXEvent(mdx, mde)
	<-done

	brw := <-em.aggregator.rewinder.rewindRequests
	assert.Equal(t, rewind{hash: *hash, rewindType: rewindBlob}, brw)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPrivateBlobReceivedBadEvent(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mde := newPrivateBlobReceived("", fftypes.NewRandB32(), 12345, "")
	em.privateBlobReceived(mdx, mde)
	mde.AssertExpectations(t)
}

func TestPrivateBlobReceivedInsertBlobFails(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error
	hash := fftypes.NewRandB32()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobs", em.ctx, mock.Anything).Return([]*core.Blob{}, nil, nil)
	mdi.On("InsertBlobs", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newPrivateBlobReceivedNoAck("peer1", hash, 12345, "ns1/path1")
	em.privateBlobReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPrivateBlobReceivedGetBlobsFails(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error
	hash := fftypes.NewRandB32()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobs", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newPrivateBlobReceivedNoAck("peer1", hash, 12345, "ns1/path1")
	em.privateBlobReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestPrivateBlobReceivedWrongNS(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error
	em.namespace.NetworkName = "ns2"
	hash := fftypes.NewRandB32()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mde := newPrivateBlobReceived("peer1", hash, 12345, "ns1/path1")
	em.privateBlobReceived(mdx, mde)

	mde.AssertExpectations(t)
}

func TestMessageReceiveMessageIdentityFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	org1 := newTestOrg("org1")
	org2 := newTestOrg("org2")
	org2.Parent = org1.ID
	node1 := newTestNode("node1", org1)
	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)
	b.Node = node1.ID
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(org2, false, nil)
	mim.On("ValidateNodeOwner", em.ctx, mock.Anything, mock.Anything).Return(false, fmt.Errorf("pop"))

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", tw)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveNodeNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	org1 := newTestOrg("org1")
	org2 := newTestOrg("org2")
	org2.Parent = org1.ID
	node1 := newTestNode("node1", org1)
	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)
	b.Node = node1.ID
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(nil, nil)

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	mde := newMessageReceived("peer1", tw, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveMessagePersistMessageFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	b.Node = node1.ID
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(org1, false, nil)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)
	mim.On("ValidateNodeOwner", em.ctx, mock.Anything, mock.Anything).Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(fmt.Errorf("optimization fail"))
	mdi.On("UpsertMessage", em.ctx, mock.Anything, database.UpsertOptimizationExisting, mock.AnythingOfType("database.PostCompletionHook")).Return(fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", tw)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mpm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceiveMessagePersistDataFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	b.Node = node1.ID
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(org1, false, nil)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)
	mim.On("ValidateNodeOwner", em.ctx, mock.Anything, mock.Anything).Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(fmt.Errorf("optimization miss"))
	mdi.On("UpsertData", em.ctx, mock.Anything, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", tw)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveUnpinnedBatchOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	batch, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	batch.Node = node1.ID
	creator := &core.Member{
		Identity: batch.Author,
		Node:     batch.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(org1, false, nil)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)
	mim.On("ValidateNodeOwner", em.ctx, mock.Anything, mock.Anything).Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})
	mdi.On("UpdateMessages", em.ctx, "ns1", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(nil)
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	mde := newMessageReceived("peer1", tw, batch.Payload.Manifest(batch.ID).String())
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveUnpinnedBatchConfirmMessagesFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	b.Node = node1.ID
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(org1, false, nil)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)
	mim.On("ValidateNodeOwner", em.ctx, mock.Anything, mock.Anything).Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})
	mdi.On("UpdateMessages", em.ctx, "ns1", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", tw)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveUnpinnedBatchPersistEventFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	b.Node = node1.ID
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(true, nil)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "signingOrg").Return(org1, false, nil)
	mim.On("GetLocalNode", mock.Anything).Return(testNode, nil)
	mim.On("ValidateNodeOwner", em.ctx, mock.Anything, mock.Anything).Return(true, nil)

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})
	mdi.On("UpdateMessages", em.ctx, "ns1", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", tw)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveMessageEnsureLocalGroupFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(false, fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", tw)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveMessageEnsureLocalGroupReject(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	b, tw := sampleBatchTransfer(t, core.TransactionTypeUnpinned)
	creator := &core.Member{
		Identity: b.Author,
		Node:     b.Node,
	}

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything, creator).Return(false, nil)

	mde := newMessageReceived("peer1", tw, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mpm.AssertExpectations(t)
}
