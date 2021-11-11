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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/syshandlersmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMessageReceiveOK(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "signingOrg",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = batch.Payload.Hash()
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "0x12345").Return(&fftypes.Organization{
		Identity: "0x12345", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(&fftypes.Organization{
		Identity: "parentOrg",
	}, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything, false).Return(nil, nil)
	err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveOkBadBatchIgnored(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{
		ID: nil, // so that we only test up to persistBatch which will return a non-retry error
		Identity: fftypes.Identity{
			Author: "signingOrg",
			Key:    "0x12345",
		},
	}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "0x12345").Return(&fftypes.Organization{
		Identity: "0x12345", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(&fftypes.Organization{
		Identity: "parentOrg",
	}, nil)
	err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceivePersistBatchError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error

	batch := &fftypes.Batch{
		ID: fftypes.NewUUID(),
		Identity: fftypes.Identity{
			Author: "signingOrg",
			Key:    "0x12345",
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = batch.Payload.Hash()
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "0x12345").Return(&fftypes.Organization{
		Identity: "0x12345", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(&fftypes.Organization{
		Identity: "parentOrg",
	}, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything, false).Return(fmt.Errorf("pop"))
	err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceivedBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	err := em.MessageReceived(mdx, "peer1", []byte(`!{}`))
	assert.NoError(t, err)

}

func TestMessageReceivedUnknownType(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	err := em.MessageReceived(mdx, "peer1", []byte(`{
		"type": "unknown"
	}`))
	assert.NoError(t, err)

}

func TestMessageReceivedNilBatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	err := em.MessageReceived(mdx, "peer1", []byte(`{
		"type": "batch"
	}`))
	assert.NoError(t, err)

}

func TestMessageReceivedNilMessage(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	err := em.MessageReceived(mdx, "peer1", []byte(`{
		"type": "message"
	}`))
	assert.NoError(t, err)

}

func TestMessageReceivedNilGroup(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	err := em.MessageReceived(mdx, "peer1", []byte(`{
		"type": "message",
		"message": {}
	}`))
	assert.NoError(t, err)
}

func TestMessageReceiveNodeLookupError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to stop retry

	batch := &fftypes.Batch{}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))
	err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
}

func TestMessageReceiveNodeNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return(nil, nil, nil)
	err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
}

func TestMessageReceiveAuthorLookupError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to stop retry

	batch := &fftypes.Batch{}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "org1"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))
	err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)
}

func TestMessageReceiveAuthorNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "org1"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, mock.Anything).Return(nil, nil)
	err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)
}

func TestMessageReceiveGetCandidateOrgFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error so we need to break the loop

	batch := &fftypes.Batch{
		ID: nil, // so that we only test up to persistBatch which will return a non-retry error
		Identity: fftypes.Identity{
			Author: "signingOrg",
			Key:    "0x12345",
		},
	}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "0x12345").Return(&fftypes.Organization{
		Identity: "0x12345", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(nil, fmt.Errorf("pop"))
	err := em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{
		ID: nil, // so that we only test up to persistBatch which will return a non-retry error
		Identity: fftypes.Identity{
			Author: "signingOrg",
			Key:    "0x12345",
		},
	}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "0x12345").Return(&fftypes.Organization{
		Identity: "0x12345", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(nil, nil)
	err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotMatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{
		ID: nil, // so that we only test up to persistBatch which will return a non-retry error
		Identity: fftypes.Identity{
			Author: "signingOrg",
			Key:    "0x12345",
		},
	}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:  fftypes.TransportPayloadTypeBatch,
		Batch: batch,
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "another"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "0x12345").Return(&fftypes.Organization{
		Identity: "0x12345", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(&fftypes.Organization{
		Identity: "parentOrg",
	}, nil)
	err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)

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

	err := em.BLOBReceived(mdx, "peer1", *hash, "ns1/path1")
	assert.NoError(t, err)

	bid := <-em.aggregator.offchainBatches
	assert.Equal(t, *batchID, *bid)

	mdi.AssertExpectations(t)
}

func TestBLOBReceivedBadEvent(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	err := em.BLOBReceived(nil, "", fftypes.Bytes32{}, "")
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

	err := em.BLOBReceived(mdx, "peer1", *hash, "ns1/path1")
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

	err := em.BLOBReceived(mdx, "peer1", *hash, "ns1/path1")
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

	err := em.BLOBReceived(mdx, "peer1", *hash, "ns1/path1")
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
			ID:        id,
			BackendID: "tracking12345",
		},
	}, nil, nil)
	mdi.On("UpdateOperation", mock.Anything, id, mock.Anything).Return(nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, "error info", fftypes.JSONObject{"extra": "info"})
	assert.NoError(t, err)

}

func TestTransferResultNotCorrelated(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	id := fftypes.NewUUID()
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)
	mdi.On("UpdateOperation", mock.Anything, id, mock.Anything).Return(nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, "error info", fftypes.JSONObject{"extra": "info"})
	assert.NoError(t, err)

}

func TestTransferResultNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // avoid retries

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil, nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, "error info", fftypes.JSONObject{"extra": "info"})
	assert.Regexp(t, "FF10158", err)

}

func TestTransferGetOpFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, "error info", fftypes.JSONObject{"extra": "info"})
	assert.Regexp(t, "FF10158", err)

}

func TestTransferUpdateFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error

	mdi := em.database.(*databasemocks.Plugin)
	id := fftypes.NewUUID()
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{
			ID:        id,
			BackendID: "tracking12345",
		},
	}, nil, nil)
	mdi.On("UpdateOperation", mock.Anything, id, mock.Anything).Return(fmt.Errorf("pop"))

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	err := em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, "error info", fftypes.JSONObject{"extra": "info"})
	assert.Regexp(t, "FF10158", err)

}

func TestMessageReceiveMessageWrongType(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			TxType: fftypes.TransactionTypeBatchPin,
		},
	}
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: msg,
		Group:   &fftypes.Group{},
	})

	mdx := &dataexchangemocks.Plugin{}
	err := em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)

	mdx.AssertExpectations(t)
}

func TestMessageReceiveMessageIdentityFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "signingOrg",
				Key:    "0x12345",
			},
			ID:     fftypes.NewUUID(),
			TxType: fftypes.TransactionTypeNone,
		},
	}
	err := msg.Seal(em.ctx)
	assert.NoError(t, err)
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: msg,
		Group:   &fftypes.Group{},
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*syshandlersmocks.SystemHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	mdi.On("GetNodes", em.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	err = em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveMessageIdentityIncorrect(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "signingOrg",
				Key:    "0x12345",
			},
			ID:     fftypes.NewUUID(),
			TxType: fftypes.TransactionTypeNone,
		},
	}
	err := msg.Seal(em.ctx)
	assert.NoError(t, err)
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: msg,
		Group:   &fftypes.Group{},
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*syshandlersmocks.SystemHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{}, nil, nil)

	err = em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveMessagePersistMessageFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "signingOrg",
				Key:    "0x12345",
			},
			ID:     fftypes.NewUUID(),
			TxType: fftypes.TransactionTypeNone,
		},
	}
	err := msg.Seal(em.ctx)
	assert.NoError(t, err)
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: msg,
		Group:   &fftypes.Group{},
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*syshandlersmocks.SystemHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "0x12345"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "0x12345").Return(&fftypes.Organization{
		Identity: "0x12345",
	}, nil)
	mdi.On("UpsertMessage", em.ctx, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err = em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveMessagePersistDataFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "signingOrg",
				Key:    "0x12345",
			},
			ID:     fftypes.NewUUID(),
			TxType: fftypes.TransactionTypeNone,
		},
	}
	data := &fftypes.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.Byteable(`{}`),
	}
	err := msg.Seal(em.ctx)
	assert.NoError(t, err)
	err = data.Seal(em.ctx)
	assert.NoError(t, err)
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: msg,
		Data:    []*fftypes.Data{data},
		Group:   &fftypes.Group{},
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*syshandlersmocks.SystemHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "0x12345"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "0x12345").Return(&fftypes.Organization{
		Identity: "0x12345",
	}, nil)
	mdi.On("UpsertData", em.ctx, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err = em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveMessagePersistEventFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "signingOrg",
				Key:    "0x12345",
			},
			ID:     fftypes.NewUUID(),
			TxType: fftypes.TransactionTypeNone,
		},
	}
	data := &fftypes.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.Byteable(`{}`),
	}
	err := msg.Seal(em.ctx)
	assert.NoError(t, err)
	err = data.Seal(em.ctx)
	assert.NoError(t, err)
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: msg,
		Data:    []*fftypes.Data{data},
		Group:   &fftypes.Group{},
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*syshandlersmocks.SystemHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(true, nil)

	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "0x12345"},
	}, nil, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "0x12345").Return(&fftypes.Organization{
		Identity: "0x12345",
	}, nil)
	mdi.On("UpsertData", em.ctx, mock.Anything, true, false).Return(nil)
	mdi.On("UpsertMessage", em.ctx, mock.Anything, true, false).Return(nil)
	mdi.On("InsertEvent", em.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	err = em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveMessageEnsureLocalGroupFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "signingOrg",
				Key:    "0x12345",
			},
			ID:     fftypes.NewUUID(),
			TxType: fftypes.TransactionTypeNone,
		},
	}
	data := &fftypes.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.Byteable(`{}`),
	}
	err := msg.Seal(em.ctx)
	assert.NoError(t, err)
	err = data.Seal(em.ctx)
	assert.NoError(t, err)
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: msg,
		Data:    []*fftypes.Data{data},
		Group:   &fftypes.Group{},
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*syshandlersmocks.SystemHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(false, fmt.Errorf("pop"))

	err = em.MessageReceived(mdx, "peer1", b)
	assert.Regexp(t, "FF10158", err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveMessageEnsureLocalGroupReject(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to avoid infinite retry

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "signingOrg",
				Key:    "0x12345",
			},
			ID:     fftypes.NewUUID(),
			TxType: fftypes.TransactionTypeNone,
		},
	}
	data := &fftypes.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.Byteable(`{}`),
	}
	err := msg.Seal(em.ctx)
	assert.NoError(t, err)
	err = data.Seal(em.ctx)
	assert.NoError(t, err)
	b, _ := json.Marshal(&fftypes.TransportWrapper{
		Type:    fftypes.TransportPayloadTypeMessage,
		Message: msg,
		Data:    []*fftypes.Data{data},
		Group:   &fftypes.Group{},
	})

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}

	msh := em.definitions.(*syshandlersmocks.SystemHandlers)
	msh.On("EnsureLocalGroup", em.ctx, mock.Anything).Return(false, nil)

	err = em.MessageReceived(mdx, "peer1", b)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}
