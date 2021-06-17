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

	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/mock"
)

func TestMessageReceiveOK(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "signingOrg",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = batch.Payload.Hash()
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "signingOrg").Return(&fftypes.Organization{
		Identity: "signingOrg", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(&fftypes.Organization{
		Identity: "parentOrg",
	}, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything, true, false).Return(nil, nil)
	em.MessageReceived(mdx, "peer1", b)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveOkBadBatchIgnored(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{
		ID:     nil, // so that we only test up to persistBatch which will return a non-retry error
		Author: "signingOrg",
	}
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "signingOrg").Return(&fftypes.Organization{
		Identity: "signingOrg", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(&fftypes.Organization{
		Identity: "parentOrg",
	}, nil)
	em.MessageReceived(mdx, "peer1", b)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceivePersistBatchError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error

	batch := &fftypes.Batch{
		ID:     fftypes.NewUUID(),
		Author: "signingOrg",
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID: fftypes.NewUUID(),
			},
		},
	}
	batch.Hash = batch.Payload.Hash()
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "signingOrg").Return(&fftypes.Organization{
		Identity: "signingOrg", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(&fftypes.Organization{
		Identity: "parentOrg",
	}, nil)
	mdi.On("UpsertBatch", em.ctx, mock.Anything, true, false).Return(fmt.Errorf("pop"))
	em.MessageReceived(mdx, "peer1", b)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceivedBadData(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	em.MessageReceived(mdx, "peer1", []byte(`!{}`))
}

func TestMessageReceiveNodeLookupError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to stop retry

	batch := &fftypes.Batch{}
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))
	em.MessageReceived(mdx, "peer1", b)
}

func TestMessageReceiveNodeNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{}
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return(nil, nil)
	em.MessageReceived(mdx, "peer1", b)
}

func TestMessageReceiveAuthorLookupError(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // to stop retry

	batch := &fftypes.Batch{}
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "org1"},
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))
	em.MessageReceived(mdx, "peer1", b)
}

func TestMessageReceiveAuthorNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{}
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "org1"},
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, mock.Anything).Return(nil, nil)
	em.MessageReceived(mdx, "peer1", b)
}

func TestMessageReceiveGetCandidateOrgFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // retryable error so we need to break the loop

	batch := &fftypes.Batch{
		ID:     nil, // so that we only test up to persistBatch which will return a non-retry error
		Author: "signingOrg",
	}
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "signingOrg").Return(&fftypes.Organization{
		Identity: "signingOrg", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(nil, fmt.Errorf("pop"))
	em.MessageReceived(mdx, "peer1", b)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{
		ID:     nil, // so that we only test up to persistBatch which will return a non-retry error
		Author: "signingOrg",
	}
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "parentOrg"},
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "signingOrg").Return(&fftypes.Organization{
		Identity: "signingOrg", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(nil, nil)
	em.MessageReceived(mdx, "peer1", b)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestMessageReceiveGetCandidateOrgNotMatch(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	batch := &fftypes.Batch{
		ID:     nil, // so that we only test up to persistBatch which will return a non-retry error
		Author: "signingOrg",
	}
	b, _ := json.Marshal(batch)

	mdi := em.database.(*databasemocks.Plugin)
	mdx := &dataexchangemocks.Plugin{}
	mdi.On("GetNodes", em.ctx, mock.Anything).Return([]*fftypes.Node{
		{Name: "node1", Owner: "another"},
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "signingOrg").Return(&fftypes.Organization{
		Identity: "signingOrg", Parent: "parentOrg",
	}, nil)
	mdi.On("GetOrganizationByIdentity", em.ctx, "parentOrg").Return(&fftypes.Organization{
		Identity: "parentOrg",
	}, nil)
	em.MessageReceived(mdx, "peer1", b)

	mdi.AssertExpectations(t)
	mdx.AssertExpectations(t)
}

func TestBLOBReceivedNoop(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdx := &dataexchangemocks.Plugin{}
	u := fftypes.NewUUID()
	em.BLOBReceived(mdx, "peer1", fftypes.NewRandB32(), fmt.Sprintf("ns1/%s", u))
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
	}, nil)
	mdi.On("UpdateOperation", mock.Anything, id, mock.Anything).Return(nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, "error info", fftypes.JSONObject{"extra": "info"})

}

func TestTransferResultNotFound(t *testing.T) {
	em, cancel := newTestEventManager(t)
	cancel() // avoid retries

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{}, nil)

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, "error info", fftypes.JSONObject{"extra": "info"})

}

func TestTransferUpdateFail(t *testing.T) {
	em, cancel := newTestEventManager(t)
	defer cancel()

	mdi := em.database.(*databasemocks.Plugin)
	id := fftypes.NewUUID()
	mdi.On("GetOperations", mock.Anything, mock.Anything).Return([]*fftypes.Operation{
		{
			ID:        id,
			BackendID: "tracking12345",
		},
	}, nil)
	mdi.On("UpdateOperation", mock.Anything, id, mock.Anything).Return(fmt.Errorf("pop"))

	mdx := &dataexchangemocks.Plugin{}
	mdx.On("Name").Return("utdx")
	em.TransferResult(mdx, "tracking12345", fftypes.OpStatusFailed, "error info", fftypes.JSONObject{"extra": "info"})

}
