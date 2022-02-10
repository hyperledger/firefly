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

package broadcast

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/hyperledger/firefly/internal/metrics"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/metricsmocks"
	"github.com/hyperledger/firefly/mocks/publicstoragemocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastMessageOk(t *testing.T) {
	metrics.Registry()
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)
	mmi := bm.metrics.(*metricsmocks.Manager)

	ctx := context.Background()
	rag := mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
	mdm.On("ResolveInlineDataBroadcast", ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
	}, []*fftypes.DataAndBlob{}, nil)
	mdi.On("UpsertMessage", ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	msgInOut := &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "did:firefly:org/abcd",
					Key:    "0x12345",
				},
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"hello": "world"}`)},
		},
	}
	mmi.On("MessageSubmitted", msgInOut).Return()
	mmi.On("IsMetricsEnabled").Return(true)
	msg, err := bm.BroadcastMessage(ctx, "ns1", msgInOut, false)
	assert.NoError(t, err)
	assert.NotNil(t, msg.Data[0].ID)
	assert.NotNil(t, msg.Data[0].Hash)
	assert.Equal(t, "ns1", msg.Header.Namespace)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastRootOrg(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	ctx := context.Background()
	rag := mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}

	org := fftypes.Organization{
		ID:     fftypes.NewUUID(),
		Name:   "org1",
		Parent: "", // root
	}
	orgBytes, err := json.Marshal(&org)
	assert.NoError(t, err)

	data := &fftypes.Data{
		ID:        fftypes.NewUUID(),
		Value:     fftypes.JSONAnyPtrBytes(orgBytes),
		Validator: fftypes.MessageTypeDefinition,
	}
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("IsMetricsEnabled").Return(false)
	mdm.On("GetMessageData", ctx, mock.Anything, mock.Anything).Return([]*fftypes.Data{data}, true, nil)
	mdm.On("ResolveInlineDataBroadcast", ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
	}, []*fftypes.DataAndBlob{}, nil)
	mdi.On("UpsertMessage", ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	msg, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:   fftypes.NewUUID(),
				Type: fftypes.MessageTypeDefinition,
				Identity: fftypes.Identity{
					Author: "did:firefly:org/12345",
					Key:    "0x12345",
				},
			},
			Data: fftypes.DataRefs{
				{
					ID:   data.ID,
					Hash: data.Hash,
				},
			},
		},
	}, false)
	assert.NoError(t, err)
	assert.NotNil(t, msg.Data[0].ID)
	assert.NotNil(t, msg.Data[0].Hash)
	assert.Equal(t, "ns1", msg.Header.Namespace)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastRootOrgBadData(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	ctx := context.Background()
	data := &fftypes.Data{
		ID:        fftypes.NewUUID(),
		Value:     fftypes.JSONAnyPtr("not an org"),
		Validator: fftypes.MessageTypeDefinition,
	}
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("IsMetricsEnabled").Return(false)
	mdm.On("GetMessageData", ctx, mock.Anything, mock.Anything).Return([]*fftypes.Data{data}, true, nil)
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(errors.New("not registered"))

	_, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:   fftypes.NewUUID(),
				Type: fftypes.MessageTypeDefinition,
				Identity: fftypes.Identity{
					Author: "did:firefly:org/12345",
					Key:    "0x12345",
				},
			},
			Data: fftypes.DataRefs{
				{
					ID:   data.ID,
					Hash: data.Hash,
				},
			},
		},
	}, false)
	assert.Error(t, err, "not registered")

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastMessageWaitConfirmOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	msa := bm.syncasync.(*syncasyncmocks.Bridge)
	mim := bm.identity.(*identitymanagermocks.Manager)

	ctx := context.Background()
	rag := mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("IsMetricsEnabled").Return(false)
	mdm.On("ResolveInlineDataBroadcast", ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
	}, []*fftypes.DataAndBlob{}, nil)
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	replyMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
		},
	}
	msa.On("WaitForMessage", ctx, "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(ctx)
		}).
		Return(replyMsg, nil)
	mdi.On("UpsertMessage", ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)

	msg, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "did:firefly:org/abcd",
					Key:    "0x12345",
				},
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"hello": "world"}`)},
		},
	}, true)
	assert.NoError(t, err)
	assert.Equal(t, replyMsg, msg)
	assert.Equal(t, "ns1", msg.Header.Namespace)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastMessageWithBlobsOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mps := bm.publicstorage.(*publicstoragemocks.Plugin)
	mim := bm.identity.(*identitymanagermocks.Manager)
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("IsMetricsEnabled").Return(false)

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	ctx := context.Background()
	rag := mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
	mdm.On("ResolveInlineDataBroadcast", ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, []*fftypes.DataAndBlob{
		{
			Data: &fftypes.Data{
				ID: dataID,
				Blob: &fftypes.BlobRef{
					Hash: blobHash,
				},
			},
			Blob: &fftypes.Blob{
				Hash:       blobHash,
				PayloadRef: "blob/1",
			},
		},
	}, nil)
	mdx.On("DownloadBLOB", ctx, "blob/1").Return(ioutil.NopCloser(bytes.NewReader([]byte(`some data`))), nil)
	mps.On("PublishData", ctx, mock.MatchedBy(func(reader io.ReadCloser) bool {
		b, err := ioutil.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, "some data", string(b))
		return true
	})).Return("payload-ref", nil)
	mdi.On("UpdateData", ctx, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertMessage", ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	msg, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "did:firefly:org/abcd",
					Key:    "0x12345",
				},
			},
		},
		InlineData: fftypes.InlineData{
			{Blob: &fftypes.BlobRef{
				Hash: blobHash,
			}},
		},
	}, false)
	assert.NoError(t, err)
	assert.NotNil(t, msg.Data[0].ID)
	assert.NotNil(t, msg.Data[0].Hash)
	assert.Equal(t, "ns1", msg.Header.Namespace)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastMessageTooLarge(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	bm.maxBatchPayloadLength = 1000000
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("IsMetricsEnabled").Return(false)

	ctx := context.Background()
	rag := mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
	mdm.On("ResolveInlineDataBroadcast", ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), ValueSize: 1000001},
	}, []*fftypes.DataAndBlob{}, nil)
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	_, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "did:firefly:org/abcd",
					Key:    "0x12345",
				},
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"hello": "world"}`)},
		},
	}, true)
	assert.Regexp(t, "FF10327", err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastMessageBadInput(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("IsMetricsEnabled").Return(false)

	ctx := context.Background()
	rag := mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
	mdm.On("ResolveInlineDataBroadcast", ctx, "ns1", mock.Anything).Return(nil, nil, fmt.Errorf("pop"))
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	_, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"hello": "world"}`)},
		},
	}, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastMessageBadIdentity(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	ctx := context.Background()
	mim := bm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(fmt.Errorf("pop"))
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("IsMetricsEnabled").Return(false)

	_, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"hello": "world"}`)},
		},
	}, false)
	assert.Regexp(t, "FF10206", err)

	mim.AssertExpectations(t)
}

func TestPublishBlobsSendMessageFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mim := bm.identity.(*identitymanagermocks.Manager)
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("IsMetricsEnabled").Return(false)

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	ctx := context.Background()
	rag := mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)
	mdm.On("ResolveInlineDataBroadcast", ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, []*fftypes.DataAndBlob{
		{
			Data: &fftypes.Data{
				ID: dataID,
				Blob: &fftypes.BlobRef{
					Hash: blobHash,
				},
			},
			Blob: &fftypes.Blob{
				Hash:       blobHash,
				PayloadRef: "blob/1",
			},
		},
	}, nil)
	mdx.On("DownloadBLOB", ctx, "blob/1").Return(nil, fmt.Errorf("pop"))

	_, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "did:firefly:org/abcd",
					Key:    "0x12345",
				},
			},
		},
		InlineData: fftypes.InlineData{
			{Blob: &fftypes.BlobRef{
				Hash: blobHash,
			}},
		},
	}, false)
	assert.Regexp(t, "FF10240", err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBroadcastPrepare(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)
	mmi := bm.metrics.(*metricsmocks.Manager)
	mmi.On("IsMetricsEnabled").Return(false)

	ctx := context.Background()
	rag := mdi.On("RunAsGroup", ctx, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		var fn = a[1].(func(context.Context) error)
		rag.ReturnArguments = mock.Arguments{fn(a[0].(context.Context))}
	}
	mdm.On("ResolveInlineDataBroadcast", ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
	}, []*fftypes.DataAndBlob{}, nil)
	mim.On("ResolveInputIdentity", ctx, mock.Anything).Return(nil)

	msg := &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "did:firefly:org/abcd",
					Key:    "0x12345",
				},
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"hello": "world"}`)},
		},
	}
	sender := bm.NewBroadcast("ns1", msg)
	err := sender.Prepare(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, msg.Data[0].ID)
	assert.NotNil(t, msg.Data[0].Hash)
	assert.Equal(t, "ns1", msg.Header.Namespace)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}
