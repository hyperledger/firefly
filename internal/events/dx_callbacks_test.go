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

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/definitionsmocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func sampleBatchTransfer(t *testing.T, txType fftypes.TransactionType) (*fftypes.Batch, []byte) {
	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypePrivate, txType, fftypes.DataArray{data})
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Batch: batch,
		Group: &fftypes.Group{
			Hash: fftypes.NewRandB32(),
		},
	})
	return batch, b
}

func newTestOrg(name string) *fftypes.Identity {
	identity := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.IdentityTypeOrg,
			Namespace: fftypes.SystemNamespace,
			Name:      name,
			Parent:    nil,
		},
	}
	identity.DID, _ = identity.GenerateDID(context.Background())
	return identity
}

func newTestNode(name string, owner *fftypes.Identity) *fftypes.Identity {
	identity := &fftypes.Identity{
		IdentityBase: fftypes.IdentityBase{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.IdentityTypeNode,
			Namespace: fftypes.SystemNamespace,
			Name:      name,
			Parent:    owner.ID,
		},
		IdentityProfile: fftypes.IdentityProfile{
			Profile: fftypes.JSONObject{
				"id":  fmt.Sprintf("%s-peer", name),
				"url": fmt.Sprintf("https://%s.example.com", name),
			},
		},
	}
	identity.DID, _ = identity.GenerateDID(context.Background())
	return identity
}

func TestPinnedReceiveOK(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeBatchPin)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything).Return(nil, nil)
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestMessageReceiveOkBadBatchIgnored(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	data := &fftypes.Data{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`"test"`)}
	batch := sampleBatch(t, fftypes.BatchTypePrivate, fftypes.TransactionTypeBatchPin, fftypes.DataArray{data})
	batch.Payload.TX.Type = fftypes.TransactionTypeTokenPool
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Batch: batch,
		Group: &fftypes.Group{
			Hash: fftypes.NewRandB32(),
		},
	})

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdx := &dataexchangemocks.Plugin{}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org1, false, nil)

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
	assert.Empty(t, m)

	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceivePersistBatchError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeBatchPin)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))
	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceivedBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	m, err := em.MessageReceived(mdx, "peer1", []byte(`!{}`))
	assert.NoError(t, err)
	assert.Empty(t, m)

}

func TestMessageReceivedUnknownType(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	m, err := em.MessageReceived(mdx, "peer1", []byte(`{
		"type": "unknown"
	}`))
	assert.NoError(t, err)
	assert.Empty(t, m)

}

func TestMessageReceivedNilBatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	m, err := em.MessageReceived(mdx, "peer1", []byte(`{
		"type": "batch"
	}`))
	assert.NoError(t, err)
	assert.Empty(t, m)

}

func TestMessageReceivedNilMessage(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	m, err := em.MessageReceived(mdx, "peer1", []byte(`{
		"type": "message"
	}`))
	assert.NoError(t, err)
	assert.Empty(t, m)

}

func TestMessageReceivedNilGroup(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	m, err := em.MessageReceived(mdx, "peer1", []byte(`{
		"type": "message",
		"message": {}
	}`))
	assert.NoError(t, err)
	assert.Empty(t, m)
}

func TestMessageReceiveNodeLookupError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to stop retry

	batch := &fftypes.Batch{}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Batch: batch,
	})

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(nil, fmt.Errorf("pop"))

	mdx := &dataexchangemocks.Plugin{}
	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
	assert.Empty(t, m)
}

func TestMessageReceiveGetCandidateOrgFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error so we need to break the loop

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeBatchPin)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(nil, true, fmt.Errorf("pop"))
	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeBatchPin)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(nil, false, nil)
	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotMatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeBatchPin)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(newTestOrg("org2"), false, nil)
	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestBLOBReceivedTriggersRewindOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()
	hash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()

	mdx := &dataexchangemocks.Plugin{}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertBlob", em.ctx, mock.Anything).Return(nil)
	mdi.On("GetDataRefs", em.ctx, mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID},
	}, nil, nil)
	mdi.On("GetMessagesForData", em.ctx, dataID, mock.Anything).Return([]*fftypes.Message{
		{BatchID: batchID},
	}, nil, nil)

	err := em.BLOBReceived(mdx, "peer1", *hash, 12345, "ns1/path1")
	assert.NoError(t, err)

	bid := <-em.aggregator.rewindBatches
	assert.Equal(t, *batchID, *bid)

	mdi.AssertExpectations(t)
}

func TestBLOBReceivedBadEvent(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	err := em.BLOBReceived(nil, "", fftypes.Bytes32{}, 12345, "")
	assert.NoError(t, err)
}

func TestBLOBReceivedGetMessagesFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error
	hash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	mdx := &dataexchangemocks.Plugin{}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertBlob", em.ctx, mock.Anything).Return(nil)
	mdi.On("GetDataRefs", em.ctx, mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID},
	}, nil, nil)
	mdi.On("GetMessagesForData", em.ctx, dataID, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err := em.BLOBReceived(mdx, "peer1", *hash, 12345, "ns1/path1")
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
}

func TestBLOBReceivedGetDataRefsFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error
	hash := fftypes.NewRandB32()

	mdx := &dataexchangemocks.Plugin{}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertBlob", em.ctx, mock.Anything).Return(nil)
	mdi.On("GetDataRefs", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err := em.BLOBReceived(mdx, "peer1", *hash, 12345, "ns1/path1")
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
}

func TestBLOBReceivedInsertBlobFails(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error
	hash := fftypes.NewRandB32()

	mdx := &dataexchangemocks.Plugin{}

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("InsertBlob", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	err := em.BLOBReceived(mdx, "peer1", *hash, 12345, "ns1/path1")
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
}

func TestTransferResultOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	id := fftypes.NewUUID()
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{
			ID: id,
		},
	}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, id, fftypes.OpStatusFailed, "error info", fftypes.JSONObject{
		"extra": "info",
	}).Return(nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, id.String(), fftypes.OpStatusFailed, fftypes.TransportStatusUpdate{
		Error: "error info",
		Info:  fftypes.JSONObject{"extra": "info"},
	})
	assert.NoError(t, err)
}

func TestTransferResultManifestMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	id := fftypes.NewUUID()
	mdi.On("GetBatchByID", mock.Anything, mock.Anything).Return(&fftypes.BatchPersisted{
		Manifest: fftypes.JSONAnyPtr("my-manifest"),
	}, nil)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{
			ID:   id,
			Type: "dataexchange_batch_send",
			Input: fftypes.JSONObject{
				"batch": fftypes.NewUUID().String(),
			},
		},
	}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, id, fftypes.OpStatusFailed, mock.MatchedBy(func(errorMsg string) bool {
		return strings.Contains(errorMsg, "FF10329")
	}), fftypes.JSONObject{
		"extra": "info",
	}).Return(nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mdx.On("Capabilities").Return(&dataexchange.Capabilities{
		Manifest: true,
	})
	err := em.TransferResult(mdx, id.String(), fftypes.OpStatusSucceeded, fftypes.TransportStatusUpdate{
		Info:     fftypes.JSONObject{"extra": "info"},
		Manifest: "Sally",
	})
	assert.NoError(t, err)

}

func TestTransferResultManifestFamil(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel()

	mdi := em.database.(*databasemocks.Plugin)
	id := fftypes.NewUUID()
	mdi.On("GetBatchByID", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{
			ID:   id,
			Type: "dataexchange_batch_send",
			Input: fftypes.JSONObject{
				"batch": fftypes.NewUUID().String(),
			},
		},
	}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, id, fftypes.OpStatusFailed, mock.MatchedBy(func(errorMsg string) bool {
		return strings.Contains(errorMsg, "FF10329")
	}), fftypes.JSONObject{
		"extra": "info",
	}).Return(nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mdx.On("Capabilities").Return(&dataexchange.Capabilities{
		Manifest: true,
	})
	err := em.TransferResult(mdx, id.String(), fftypes.OpStatusSucceeded, fftypes.TransportStatusUpdate{
		Info:     fftypes.JSONObject{"extra": "info"},
		Manifest: "Sally",
	})
	assert.Regexp(t, "FF10158", err)

}

func TestTransferResultHashtMismatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	id := fftypes.NewUUID()
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{
			ID:   id,
			Type: "dataexchange_blob_send",
			Input: fftypes.JSONObject{
				"hash": "Bob",
			},
		},
	}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, id, fftypes.OpStatusFailed, mock.MatchedBy(func(errorMsg string) bool {
		return strings.Contains(errorMsg, "FF10348")
	}), fftypes.JSONObject{
		"extra": "info",
	}).Return(nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	mdx.On("Capabilities").Return(&dataexchange.Capabilities{
		Manifest: true,
	})
	err := em.TransferResult(mdx, id.String(), fftypes.OpStatusSucceeded, fftypes.TransportStatusUpdate{
		Info: fftypes.JSONObject{"extra": "info"},
		Hash: "Sally",
	})
	assert.NoError(t, err)

}

func TestTransferResultNotCorrelated(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, fftypes.TransportStatusUpdate{
		Error: "error info",
		Info:  fftypes.JSONObject{"extra": "info"},
	})
	assert.NoError(t, err)

}

func TestTransferResultNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel() // we want to retry until the count

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, fftypes.TransportStatusUpdate{
		Error: "error info",
		Info:  fftypes.JSONObject{"extra": "info"},
	})
	assert.NoError(t, err)

}

func TestTransferGetOpFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, fftypes.TransportStatusUpdate{
		Error: "error info",
		Info:  fftypes.JSONObject{"extra": "info"},
	})
	assert.Regexp(t, "FF10158", err)

}

func TestTransferUpdateFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error

	mdi := em.database.(*databasemocks.Plugin)
	id := fftypes.NewUUID()
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{
			ID: id,
		},
	}, nil, nil)
	mdi.On("ResolveOperation", mock.Anything, id, fftypes.OpStatusFailed, "error info", fftypes.JSONObject{
		"extra": "info",
	}).Return(fmt.Errorf("pop"))

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, id.String(), fftypes.OpStatusFailed, fftypes.TransportStatusUpdate{
		Error: "error info",
		Info:  fftypes.JSONObject{"extra": "info"},
	})
	assert.Regexp(t, "FF10158", err)

}

func TestMessageReceiveMessageIdentityFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	org1 := newTestOrg("org1")
	org2 := newTestOrg("org2")
	org2.Parent = org1.ID
	node1 := newTestNode("node1", org1)
	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeBatchPin)

	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org2, false, nil)
	mim.On("CachedIdentityLookupByID", em.ctx, org2.Parent).Return(nil, fmt.Errorf("pop"))

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
	assert.Empty(t, m)

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
	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeBatchPin)

	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org2, false, nil)
	mim.On("CachedIdentityLookupByID", em.ctx, org2.Parent).Return(nil, nil)

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
	assert.Empty(t, m)

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
	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeBatchPin)

	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org2, false, nil)
	mim.On("CachedIdentityLookupByID", em.ctx, org2.Parent).Return(org3, nil)

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
	assert.Empty(t, m)

	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestMessageReceiveMessagePersistMessageFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything).Return(fmt.Errorf("optimization fail"))
	mdi.On("UpsertMessage", em.ctx, mock.Anything, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveMessagePersistDataFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(fmt.Errorf("optimization miss"))
	mdi.On("UpsertData", em.ctx, mock.Anything, database.UpsertOptimizationExisting).Return(fmt.Errorf("pop"))

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveUnpinnedBatchOk(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything).Return(nil)
	mdi.On("UpdateMessages", em.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(nil)
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
	assert.NotEmpty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestMessageReceiveUnpinnedBatchConfirmMessagesFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything).Return(nil)
	mdi.On("UpdateMessages", em.ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestMessageReceiveUnpinnedBatchPersistEventFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	org1 := newTestOrg("org1")
	node1 := newTestNode("node1", org1)

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)
	mim := em.identity.(*identitymanagermocks.Manager)
	mim.On("FindIdentityForVerifier", em.ctx, []fftypes.IdentityType{fftypes.IdentityTypeNode}, fftypes.SystemNamespace, &fftypes.VerifierRef{
		Type:  fftypes.VerifierTypeFFDXPeerID,
		Value: "peer1",
	}).Return(node1, nil)
	mim.On("CachedIdentityLookup", em.ctx, "signingOrg").Return(org1, false, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything).Return(nil, nil)
	mdi.On("InsertDataArray", em.ctx, mock.Anything).Return(nil)
	mdi.On("InsertMessages", em.ctx, mock.Anything).Return(nil)
	mdi.On("UpdateMessages", em.ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))
	mdm := em.data.(*datamocks.Manager)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestMessageReceiveMessageEnsureLocalGroupFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(false, fmt.Errorf("pop"))

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "pop", err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveMessageEnsureLocalGroupReject(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	_, b := sampleBatchTransfer(t, fftypes.TransactionTypeUnpinned)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*definitionsmocks.DefinitionHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(false, nil)

	m, err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
	assert.Empty(t, m)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}
