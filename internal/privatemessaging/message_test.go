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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/batch"
	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/operationmocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func TestSendConfirmMessageE2EOk(t *testing.T) {

	pm, cancel := newTestPrivateMessagingWithMetrics(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	rootOrg := newTestOrg("rootorg")
	intermediateOrg := newTestOrg("localorg")
	intermediateOrg.Parent = rootOrg.ID
	localNode := newTestNode("node1", intermediateOrg)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(nil)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(intermediateOrg, nil)
	mim.On("CachedIdentityLookup", pm.ctx, "org1").Return(intermediateOrg, false, nil)
	mim.On("CachedIdentityLookupByID", pm.ctx, rootOrg.ID).Return(rootOrg, nil)

	dataID := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataArray{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentities", pm.ctx, mock.Anything).Return([]*fftypes.Identity{}, nil, nil).Once()
	mdi.On("GetIdentities", pm.ctx, mock.Anything).Return([]*fftypes.Identity{localNode}, nil, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil, nil).Once()

	retMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	msa := pm.syncasync.(*syncasyncmocks.Bridge)
	msa.On("WaitForMessage", pm.ctx, "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(pm.ctx)
		}).
		Return(retMsg, nil).Once()
	mdi.On("UpsertMessage", pm.ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil).Once()

	msg, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, true)
	assert.NoError(t, err)
	assert.Equal(t, retMsg, msg)

	mim.AssertExpectations(t)
	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestSendUnpinnedMessageE2EOk(t *testing.T) {

	pm, cancel := newTestPrivateMessagingWithMetrics(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Run(func(args mock.Arguments) {
		identity := args[2].(*fftypes.SignerRef)
		identity.Author = "localorg"
		identity.Key = "localkey"
	}).Return(nil)

	dataID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataArray{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertMessage", pm.ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil).Once()
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{Hash: groupID}, nil)

	msg, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeUnpinned,
				Group:  groupID,
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, *dataID, *msg.Data[0].ID)
	assert.NotNil(t, msg.Header.Group)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestSendMessageBadGroup(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(nil)

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{},
	}, true)
	assert.Regexp(t, "FF10219", err)

	mim.AssertExpectations(t)

}

func TestSendMessageBadIdentity(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(fmt.Errorf("pop"))

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, false)
	assert.Regexp(t, "FF10206.*pop", err)

	mim.AssertExpectations(t)

}

func TestSendMessageFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	localOrg := newTestOrg("localorg")
	localNode := newTestNode("node1", localOrg)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(nil)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Run(func(args mock.Arguments) {
		identity := args[2].(*fftypes.SignerRef)
		identity.Author = "localorg"
		identity.Key = "localkey"
	}).Return(nil)
	mim.On("CachedIdentityLookup", pm.ctx, "localorg").Return(localOrg, false, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentities", pm.ctx, mock.Anything).Return([]*fftypes.Identity{localNode}, nil, nil)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil, nil)
	mdi.On("UpsertMessage", pm.ctx, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	dataID := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataArray{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "localorg"},
			},
		},
	}, false)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)

}

func TestResolveAndSendBadInlineData(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	localOrg := newTestOrg("localorg")
	localNode := newTestNode("node1", localOrg)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(nil)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Run(func(args mock.Arguments) {
		identity := args[2].(*fftypes.SignerRef)
		identity.Author = "localorg"
		identity.Key = "localkey"
	}).Return(nil)
	mim.On("CachedIdentityLookup", pm.ctx, "localorg").Return(localOrg, false, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentities", pm.ctx, mock.Anything).Return([]*fftypes.Identity{localNode}, nil, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil, nil).Once()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(nil, fmt.Errorf("pop"))

	message := &messageSender{
		mgr:       pm,
		namespace: "ns1",
		msg: &fftypes.MessageInOut{
			Message: fftypes.Message{Header: fftypes.MessageHeader{Namespace: "ns1"}},
			Group: &fftypes.InputGroup{
				Members: []fftypes.MemberInput{
					{Identity: "localorg"},
				},
			},
		},
	}

	_, err := message.resolve(pm.ctx)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)

}

func TestSendUnpinnedMessageTooLarge(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	pm.maxBatchPayloadLength = 100000
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Run(func(args mock.Arguments) {
		identity := args[2].(*fftypes.SignerRef)
		identity.Author = "localorg"
		identity.Key = "localkey"
	}).Return(nil)

	dataID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataArray{
		{ID: dataID, Hash: fftypes.NewRandB32(), ValueSize: 100001},
	}, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{Hash: groupID}, nil)

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeUnpinned,
				Group:  groupID,
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, false)
	assert.Regexp(t, "FF10328", err)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestSealFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	id1 := fftypes.NewUUID()
	message := pm.NewMessage("ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Data: fftypes.DataRefs{
				{ID: id1, Hash: fftypes.NewRandB32()},
				{ID: id1, Hash: fftypes.NewRandB32()}, // duplicate ID
			},
		},
	})

	err := message.(*messageSender).sendInternal(pm.ctx, methodSend)
	assert.Regexp(t, "FF10145", err)

}

func TestMessagePrepare(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	localOrg := newTestOrg("localorg")
	localNode := newTestNode("node1", localOrg)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(nil)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Run(func(args mock.Arguments) {
		identity := args[2].(*fftypes.SignerRef)
		identity.Author = "localorg"
		identity.Key = "localkey"
	}).Return(nil)
	mim.On("CachedIdentityLookup", pm.ctx, "localorg").Return(localOrg, false, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentities", pm.ctx, mock.Anything).Return([]*fftypes.Identity{localNode}, nil, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil, nil).Once()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataArray{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
	}, nil)

	message := pm.NewMessage("ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Data: fftypes.DataRefs{
				{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
			},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "localorg"},
			},
		},
	})

	err := message.Prepare(pm.ctx)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)

}

func TestSendUnpinnedMessageGroupLookupFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewRandB32()
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(nil, fmt.Errorf("pop")).Once()

	err := pm.dispatchUnpinnedBatch(pm.ctx, &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				ID:    fftypes.NewUUID(),
				Group: groupID,
			},
		},
		Payload: fftypes.BatchPayload{
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						SignerRef: fftypes.SignerRef{
							Author: "org1",
						},
						TxType: fftypes.TransactionTypeUnpinned,
						Group:  groupID,
					},
				},
			},
		},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestSendUnpinnedMessageInsertFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.MatchedBy(func(identity *fftypes.SignerRef) bool {
		assert.Empty(t, identity.Author)
		return true
	})).Return(nil)

	dataID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataArray{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertMessage", pm.ctx, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop")).Once()
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{Hash: groupID}, nil)

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeUnpinned,
				Group:  groupID,
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, false)
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestSendUnpinnedMessageConfirmFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(fmt.Errorf("pop"))

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeUnpinned,
				Group:  fftypes.NewRandB32(),
			},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, true)
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)

}

func TestSendUnpinnedMessageResolveGroupFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(nil)

	groupID := fftypes.NewRandB32()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(nil, fmt.Errorf("pop")).Once()

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, mock.Anything, "peer2-remote", mock.Anything).Return("tracking1", nil).Once()

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeUnpinned,
				Group:  groupID,
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, false)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestSendUnpinnedMessageResolveGroupNotFound(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(nil)

	groupID := fftypes.NewRandB32()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(nil, nil)

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, mock.Anything, "peer2-remote", mock.Anything).Return(nil).Once()

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeUnpinned,
				Group:  groupID,
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.JSONAnyPtr(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, false)
	assert.Regexp(t, "FF10226", err)

	mdi.AssertExpectations(t)
	mim.AssertExpectations(t)

}

func TestRequestReplyMissingTag(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	msa := pm.syncasync.(*syncasyncmocks.Bridge)
	msa.On("WaitForReply", pm.ctx, "ns1", mock.Anything).Return(nil, nil)

	_, err := pm.RequestReply(pm.ctx, "ns1", &fftypes.MessageInOut{})
	assert.Regexp(t, "FF10261", err)
}

func TestRequestReplyInvalidCID(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	msa := pm.syncasync.(*syncasyncmocks.Bridge)
	msa.On("WaitForReply", pm.ctx, "ns1", mock.Anything).Return(nil, nil)

	_, err := pm.RequestReply(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Tag:   "mytag",
				CID:   fftypes.NewUUID(),
				Group: fftypes.NewRandB32(),
			},
		},
	})
	assert.Regexp(t, "FF10262", err)
}

func TestRequestReplySuccess(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.Anything).Return(nil)

	msa := pm.syncasync.(*syncasyncmocks.Bridge)
	msa.On("WaitForReply", pm.ctx, "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(pm.ctx)
		}).
		Return(nil, nil)

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataArray{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
	}, nil)
	mdm.On("UpdateMessageCache", mock.Anything, mock.Anything).Return()

	groupID := fftypes.NewRandB32()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertMessage", pm.ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil).Once()
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{Hash: groupID}, nil)

	_, err := pm.RequestReply(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Tag:   "mytag",
				Group: groupID,
				SignerRef: fftypes.SignerRef{
					Author: "org1",
				},
			},
		},
	})
	assert.NoError(t, err)
}

func TestDispatchedUnpinnedMessageOK(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	localOrg := newTestOrg("localorg")
	groupID := fftypes.NewRandB32()
	node1 := newTestNode("node1", localOrg)
	node2 := newTestNode("node2", newTestOrg("remoteorg"))

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", pm.ctx, "ns1", mock.MatchedBy(func(identity *fftypes.SignerRef) bool {
		assert.Equal(t, "localorg", identity.Author)
		return true
	})).Return(nil)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, mock.Anything, "node2-peer", mock.Anything).Return(nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mom := pm.operations.(*operationmocks.Manager)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: groupID,
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Node: node1.ID, Identity: "localorg"},
				{Node: node2.ID, Identity: "remoteorg"},
			},
		},
	}, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, node1.ID).Return(node1, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, node2.ID).Return(node2, nil).Once()

	mom.On("AddOrReuseOperation", pm.ctx, mock.Anything).Return(nil)
	mom.On("RunOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(batchSendData)
		return op.Type == fftypes.OpTypeDataExchangeBatchSend && *data.Node.ID == *node2.ID
	})).Return(nil)

	err := pm.dispatchUnpinnedBatch(pm.ctx, &batch.DispatchState{
		Persisted: fftypes.BatchPersisted{
			BatchHeader: fftypes.BatchHeader{
				ID:    fftypes.NewUUID(),
				Group: groupID,
			},
		},
		Payload: fftypes.BatchPayload{
			TX: fftypes.TransactionRef{
				ID:   fftypes.NewUUID(),
				Type: fftypes.TransactionTypeUnpinned,
			},
			Messages: []*fftypes.Message{
				{
					Header: fftypes.MessageHeader{
						Tag:   "mytag",
						Group: groupID,
						SignerRef: fftypes.SignerRef{
							Author: "org1",
						},
					},
				},
			},
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mom.AssertExpectations(t)

}

func TestSendDataTransferBlobsFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	localOrg := newTestOrg("localorg")
	groupID := fftypes.NewRandB32()
	node2 := newTestNode("node2", newTestOrg("remoteorg"))
	nodes := []*fftypes.Identity{node2}

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", pm.ctx, mock.MatchedBy(func(identity *fftypes.SignerRef) bool {
		assert.Equal(t, "localorg", identity.Author)
		return true
	})).Return(nil)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetBlobMatchingHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := pm.sendData(pm.ctx, &fftypes.TransportWrapper{
		Batch: &fftypes.Batch{
			BatchHeader: fftypes.BatchHeader{
				ID:    fftypes.NewUUID(),
				Group: groupID,
			},
			Payload: fftypes.BatchPayload{
				Messages: []*fftypes.Message{
					{
						Header: fftypes.MessageHeader{
							Tag:   "mytag",
							Group: groupID,
							SignerRef: fftypes.SignerRef{
								Author: "org1",
							},
						},
					},
				},
				Data: fftypes.DataArray{
					{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr("{}"), Blob: &fftypes.BlobRef{
						Hash: fftypes.NewRandB32(),
					}},
				},
			},
		},
	}, nodes, "manifest-data")
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestSendDataTransferFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	localOrg := newTestOrg("localorg")
	groupID := fftypes.NewRandB32()
	node2 := newTestNode("node2", newTestOrg("remoteorg"))
	nodes := []*fftypes.Identity{node2}

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)

	mom := pm.operations.(*operationmocks.Manager)
	mom.On("AddOrReuseOperation", pm.ctx, mock.Anything).Return(nil)
	mom.On("RunOperation", pm.ctx, mock.MatchedBy(func(op *fftypes.PreparedOperation) bool {
		data := op.Data.(batchSendData)
		return op.Type == fftypes.OpTypeDataExchangeBatchSend && *data.Node.ID == *node2.ID
	})).Return(fmt.Errorf("pop"))

	err := pm.sendData(pm.ctx, &fftypes.TransportWrapper{
		Batch: &fftypes.Batch{
			BatchHeader: fftypes.BatchHeader{
				ID:    fftypes.NewUUID(),
				Group: groupID,
			},
			Payload: fftypes.BatchPayload{
				Messages: []*fftypes.Message{
					{
						Header: fftypes.MessageHeader{
							Tag:   "mytag",
							Group: groupID,
							SignerRef: fftypes.SignerRef{
								Author: "org1",
							},
						},
					},
				},
			},
		},
	}, nodes, "manifest-data")
	assert.Regexp(t, "pop", err)

	mim.AssertExpectations(t)
	mom.AssertExpectations(t)

}

func TestSendDataTransferInsertOperationFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	localOrg := newTestOrg("localorg")
	groupID := fftypes.NewRandB32()
	node2 := newTestNode("node2", newTestOrg("remoteorg"))
	nodes := []*fftypes.Identity{node2}

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", pm.ctx, mock.MatchedBy(func(identity *fftypes.SignerRef) bool {
		assert.Equal(t, "localorg", identity.Author)
		return true
	})).Return(nil)
	mim.On("GetNodeOwnerOrg", pm.ctx).Return(localOrg, nil)

	mom := pm.operations.(*operationmocks.Manager)
	mom.On("AddOrReuseOperation", pm.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	err := pm.sendData(pm.ctx, &fftypes.TransportWrapper{
		Batch: &fftypes.Batch{
			BatchHeader: fftypes.BatchHeader{
				ID:    fftypes.NewUUID(),
				Group: groupID,
			},
			Payload: fftypes.BatchPayload{
				Messages: []*fftypes.Message{
					{
						Header: fftypes.MessageHeader{
							Tag:   "mytag",
							Group: groupID,
							SignerRef: fftypes.SignerRef{
								Author: "org1",
							},
						},
					},
				},
			},
		},
	}, nodes, "manifest-data")
	assert.Regexp(t, "pop", err)

}
