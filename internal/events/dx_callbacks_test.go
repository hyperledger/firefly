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
	"encoding/json"
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

func sampleBatchTransfer(t *testing.T, txType core.TransactionType) (*core.Batch, []byte) {
	data := &core.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, core.BatchTypePrivate, txType, core.DataArray{data})
	b, _ := json.Marshal(&core.TransportWrapper{
		Batch: batch,
		Group: &core.Group{
			Hash: fftypes.NewRandB32(),
		},
	})
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

func newMessageReceivedNoAck(peerID string, data []byte) *dataexchangemocks.DXEvent {
	mde := &dataexchangemocks.DXEvent{}
	mde.On("MessageReceived").Return(&dataexchange.MessageReceived{
		PeerID: peerID,
		Data:   data,
	})
	mde.On("Type").Return(dataexchange.DXEventTypeMessageReceived).Maybe()
	return mde
}

func newMessageReceived(peerID string, data []byte, expectedManifest string) *dataexchangemocks.DXEvent {
	mde := newMessageReceivedNoAck(peerID, data)
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
	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org1, false, nil)
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
	b, _ := json.Marshal(&core.TransportWrapper{
		Batch: batch,
		Group: &core.Group{
			Hash: fftypes.NewRandB32(),
		},
	})

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org1, false, nil)

	mde := newMessageReceived("peer1", b, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceivePersistBatchError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error

	_, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceivedBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mde := newMessageReceived("peer1", []byte(`!{}`), "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)

}

func TestMessageReceivedUnknownType(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mde := newMessageReceived("peer1", []byte(`{
		"type": "unknown"
	}`), "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
}

func TestMessageReceivedNilBatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mde := newMessageReceived("peer1", []byte(`{
		"type": "batch"
	}`), "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
}

func TestMessageReceivedNilMessage(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mde := newMessageReceived("peer1", []byte(`{
		"type": "message"
	}`), "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
}

func TestMessageReceivedNilGroup(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mde := newMessageReceived("peer1", []byte(`{
		"type": "message",
		"message": {}
	}`), "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
}

func TestMessageReceiveNodeLookupError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to stop retry

	batch := &core.Batch{
		BatchHeader: core.BatchHeader{
			Namespace: "ns1",
		},
	}
	b, _ := json.Marshal(&core.TransportWrapper{
		Batch: batch,
	})

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(nil, fmt.Errorf("pop"))

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error so we need to break the loop

	_, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(nil, true, fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	_, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(nil, false, nil)
	mde := newMessageReceived("peer1", b, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotMatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	_, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(newTestOrg("org2"), false, nil)
	mde := newMessageReceived("peer1", b, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
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

func TestMessageReceiveMessageIdentityFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	org1 := newTestOrg("org1")
	org2 := newTestOrg("org2")
	org2.Parent = org1.ID
	node1 := newTestNode("node1", org1)
	_, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org2, false, nil)
	mim.On("CachedIdentityLookupByID", em.ctx, org2.Parent).Return(nil, fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceiveMessageIdentityParentNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	org1 := newTestOrg("org1")
	org2 := newTestOrg("org2")
	org2.Parent = org1.ID
	node1 := newTestNode("node1", org1)
	_, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org2, false, nil)
	mim.On("CachedIdentityLookupByID", em.ctx, org2.Parent).Return(nil, nil)

	mde := newMessageReceived("peer1", b, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceiveMessageIdentityIncorrect(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	org1 := newTestOrg("org1")
	org2 := newTestOrg("org2")
	org3 := newTestOrg("org3")
	org2.Parent = org1.ID
	node1 := newTestNode("node1", org1)
	_, b := sampleBatchTransfer(t, core.TransactionTypeBatchPin)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org2, false, nil)
	mim.On("CachedIdentityLookupByID", em.ctx, org2.Parent).Return(org3, nil)

	mde := newMessageReceived("peer1", b, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceiveMessagePersistMessageFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(fmt.Errorf("optimization fail"))
	mdi.On("UpsertMessage", em.ctx, mock.Anything, database.UpsertOptimizationExisting, mock.AnythingOfType("database.PostCompletionHook")).Return(fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveMessagePersistDataFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(fmt.Errorf("optimization miss"))
	mdi.On("UpsertData", em.ctx, mock.Anything, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveUnpinnedBatchOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	batch, b := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})
	mdi.On("UpdateMessages", em.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(nil)
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	mde := newMessageReceived("peer1", b, batch.Payload.Manifest(batch.ID).String())
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

	_, b := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})
	mdi.On("UpdateMessages", em.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
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

	_, b := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []core.IdentityType{core.IdentityTypeNode}, "ns1", &core.VerifierRef{
		Type:  core.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookupMustExist", em.ctx, "ns1", "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything, mock.AnythingOfType("database.PostCompletionHook")).Return(nil, nil).Run(func(args mock.Arguments) {
		args[2].(database.PostCompletionHook)()
	})
	mdi.On("UpdateMessages", em.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
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

	_, b := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(false, fmt.Errorf("pop"))

	// no ack as we are simulating termination mid retry
	mde := newMessageReceivedNoAck("peer1", b)
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mpm.AssertExpectations(t)
}

func TestMessageReceiveMessageEnsureLocalGroupReject(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, core.TransactionTypeUnpinned)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")

	mpm := em.messaging.(*privatemessagingmocks.Manager)
	mpm.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(false, nil)

	mde := newMessageReceived("peer1", b, "")
	em.messageReceived(mdx, mde)

	mde.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mpm.AssertExpectations(t)
}
