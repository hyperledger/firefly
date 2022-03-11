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
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastMessageOk(t *testing.T) {
	bm, cancel := newTestBroadcastWithMetrics(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	ctx := context.Background()
	mdm.On("ResolveInlineDataBroadcast", ctx, mock.Anything).Return(nil)
	mdm.On("WriteNewMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mim.On("ResolveInputSigningIdentity", ctx, "ns1", mock.Anything).Return(nil)

	msg, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				SignerRef: fftypes.SignerRef{
					Author: "did:firefly:org/abcd",
					Key:    "0x12345",
				},
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"hello": "world"}`)},
		},
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, "ns1", msg.Header.Namespace)

	mim.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastMessageWaitConfirmOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	msa := bm.syncasync.(*syncasyncmocks.Bridge)
	mim := bm.identity.(*identitymanagermocks.Manager)

	ctx := context.Background()
	mdm.On("ResolveInlineDataBroadcast", ctx, mock.Anything).Return(nil)
	mim.On("ResolveInputSigningIdentity", ctx, "ns1", mock.Anything).Return(nil)

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
	mdm.On("WriteNewMessage", ctx, mock.Anything, mock.Anything).Return(nil)

	msg, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				SignerRef: fftypes.SignerRef{
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

	msa.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastMessageWithBlobsOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mps := bm.sharedstorage.(*sharedstoragemocks.Plugin)
	mim := bm.identity.(*identitymanagermocks.Manager)

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	ctx := context.Background()
	mdm.On("ResolveInlineDataBroadcast", ctx, mock.Anything).
		Run(func(args mock.Arguments) {
			newMsg := args[1].(*data.NewMessage)
			newMsg.ResolvedData.AllData = fftypes.DataArray{
				{ID: dataID, Hash: fftypes.NewRandB32()},
			}
			newMsg.ResolvedData.DataToPublish = []*fftypes.DataAndBlob{
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
			}
		}).
		Return(nil)
	mdx.On("DownloadBLOB", ctx, "blob/1").Return(ioutil.NopCloser(bytes.NewReader([]byte(`some data`))), nil)
	var readStr string
	mps.On("PublishData", ctx, mock.MatchedBy(func(reader io.ReadCloser) bool {
		if readStr == "" { // called again in AssertExpectationa
			b, err := ioutil.ReadAll(reader)
			assert.NoError(t, err)
			readStr = string(b)
		}
		assert.Equal(t, "some data", readStr)
		return true
	})).Return("payload-ref", nil).Once()
	mdm.On("WriteNewMessage", ctx, mock.Anything, mock.Anything).Return(nil)
	mim.On("ResolveInputSigningIdentity", ctx, "ns1", mock.Anything).Return(nil)

	msg, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				SignerRef: fftypes.SignerRef{
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
	assert.Equal(t, "ns1", msg.Header.Namespace)

	mdx.AssertExpectations(t)
	mps.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBroadcastMessageTooLarge(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	bm.maxBatchPayloadLength = 1000000
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	ctx := context.Background()
	mdm.On("ResolveInlineDataBroadcast", ctx, mock.Anything).Run(
		func(args mock.Arguments) {
			newMsg := args[1].(*data.NewMessage)
			newMsg.Message.Data = fftypes.DataRefs{
				{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32(), ValueSize: 1000001},
			}
		}).
		Return(nil)
	mim.On("ResolveInputSigningIdentity", ctx, "ns1", mock.Anything).Return(nil)

	_, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				SignerRef: fftypes.SignerRef{
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

	mdm.AssertExpectations(t)
}

func TestBroadcastMessageBadInput(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	ctx := context.Background()
	mdm.On("ResolveInlineDataBroadcast", ctx, mock.Anything).Return(fmt.Errorf("pop"))
	mim.On("ResolveInputSigningIdentity", ctx, "ns1", mock.Anything).Return(nil)

	_, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"hello": "world"}`)},
		},
	}, false)
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
}

func TestBroadcastMessageBadIdentity(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	ctx := context.Background()
	mim := bm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", ctx, "ns1", mock.Anything).Return(fmt.Errorf("pop"))

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
	mdm := bm.data.(*datamocks.Manager)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mim := bm.identity.(*identitymanagermocks.Manager)

	blobHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()

	ctx := context.Background()
	mim.On("ResolveInputSigningIdentity", ctx, "ns1", mock.Anything).Return(nil)
	mdm.On("ResolveInlineDataBroadcast", ctx, mock.Anything).
		Run(func(args mock.Arguments) {
			newMsg := args[1].(*data.NewMessage)
			newMsg.ResolvedData.AllData = fftypes.DataArray{
				{ID: dataID, Hash: fftypes.NewRandB32()},
			}
			newMsg.ResolvedData.DataToPublish = []*fftypes.DataAndBlob{
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
			}
		}).
		Return(nil)
	mdx.On("DownloadBLOB", ctx, "blob/1").Return(nil, fmt.Errorf("pop"))

	_, err := bm.BroadcastMessage(ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				SignerRef: fftypes.SignerRef{
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

	mdm.AssertExpectations(t)
	mdx.AssertExpectations(t)
	mim.AssertExpectations(t)
}

func TestBroadcastPrepare(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	ctx := context.Background()
	mdm.On("ResolveInlineDataBroadcast", ctx, mock.Anything).Return(nil)
	mim.On("ResolveInputSigningIdentity", ctx, "ns1", mock.Anything).Return(nil)

	msg := &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				SignerRef: fftypes.SignerRef{
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
	assert.Equal(t, "ns1", msg.Header.Namespace)

	mdm.AssertExpectations(t)
}
