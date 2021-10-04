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

	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSendConfirmMessageE2EOk(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("ResolveInputIdentity", pm.ctx, mock.Anything).Return(nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(&fftypes.Organization{Identity: "localorg"}, nil)

	dataID := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", pm.ctx, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		err := a[1].(func(context.Context) error)(a[0].(context.Context))
		rag.ReturnArguments = mock.Arguments{err}
	}
	mdi.On("GetOrganizationByName", pm.ctx, "localorg").Return(&fftypes.Organization{
		ID: fftypes.NewUUID(),
	}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{
		{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"},
	}, nil, nil).Once()
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(&fftypes.Organization{
		ID: fftypes.NewUUID(),
	}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{
		{ID: fftypes.NewUUID(), Name: "node1", Owner: "org1"},
	}, nil, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil, nil).Once()

	requestID := fftypes.NewUUID()
	retMsg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
	}
	msa := pm.syncasync.(*syncasyncmocks.Bridge)
	msa.On("SendConfirm", pm.ctx, "ns1", mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[2].(syncasync.RequestSender)
			send(requestID)
		}).
		Return(retMsg, nil).Once()
	mdi.On("InsertMessageLocal", pm.ctx, mock.MatchedBy(func(msg *fftypes.Message) bool {
		return msg.Header.ID == requestID
	})).Return(nil).Once()

	msg, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, true)
	assert.NoError(t, err)
	assert.Equal(t, retMsg, msg)

}

func TestSendUnpinnedMessageE2EOk(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("ResolveInputIdentity", pm.ctx, mock.Anything).Run(func(args mock.Arguments) {
		identity := args[1].(*fftypes.Identity)
		identity.Author = "localorg"
		identity.Key = "localkey"
	}).Return(nil)

	dataID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	nodeID1 := fftypes.NewUUID()
	nodeID2 := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: dataID, Value: fftypes.Byteable(`{"some": "data"}`)},
	}, true, nil).Once()

	mdi := pm.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", pm.ctx, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		err := a[1].(func(context.Context) error)(a[0].(context.Context))
		rag.ReturnArguments = mock.Arguments{err}
	}
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: groupID,
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Node: nodeID1, Identity: "localorg"},
				{Node: nodeID2, Identity: "remoteorg"},
			},
		},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, nodeID1).Return(&fftypes.Node{
		ID: nodeID1, Name: "node1", Owner: "localorg", DX: fftypes.DXInfo{Peer: "peer1-local"},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, nodeID2).Return(&fftypes.Node{
		ID: nodeID2, Name: "node2", Owner: "org1", DX: fftypes.DXInfo{Peer: "peer2-remote"},
	}, nil).Once()
	mdi.On("InsertMessageLocal", pm.ctx, mock.Anything).Return(nil).Once()
	mdi.On("InsertEvent", pm.ctx, mock.Anything).Return(nil).Once()

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, "peer2-remote", mock.Anything).Return("tracking1", nil).Once()

	msg, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeNone,
				Group:  groupID,
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
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

func TestSendMessageBadIdentity(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", pm.ctx, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
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
	mim.On("ResolveInputIdentity", pm.ctx, mock.Anything).Run(func(args mock.Arguments) {
		identity := args[1].(*fftypes.Identity)
		identity.Author = "localorg"
		identity.Key = "localkey"
	}).Return(nil)

	dataID := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", pm.ctx, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	}, false)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)

}

func TestResolveAndSendBadMembers(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	err := pm.resolveMessage(pm.ctx, &fftypes.MessageInOut{
		InlineData: fftypes.InlineData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
		},
	})
	assert.Regexp(t, "FF10219", err)

}

func TestResolveAndSendBadInlineData(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(&fftypes.Organization{Identity: "localorg"}, nil)

	mdi := pm.database.(*databasemocks.Plugin)

	mdi.On("GetOrganizationByName", pm.ctx, "localorg").Return(&fftypes.Organization{
		ID: fftypes.NewUUID(),
	}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{
		{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"},
	}, nil, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil, nil).Once()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := pm.resolveMessage(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{Header: fftypes.MessageHeader{Namespace: "ns1"}},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "localorg"},
			},
		},
	})
	assert.Regexp(t, "pop", err)

}

func TestSealFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	id1 := fftypes.NewUUID()
	_, err := pm.sendOrWaitMessage(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{Namespace: "ns1"},
		Data: fftypes.DataRefs{
			{ID: id1},
			{ID: id1}, // duplicate
		},
	}, false)
	assert.Regexp(t, "FF10144", err)

}

func TestSendUnpinnedMessageMarshalFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", pm.ctx, mock.MatchedBy(func(identity *fftypes.Identity) bool {
		assert.Equal(t, "localorg", identity.Author)
		return true
	})).Return(nil)

	groupID := fftypes.NewRandB32()
	nodeID1 := fftypes.NewUUID()
	nodeID2 := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`!Invalid JSON`)},
	}, true, nil).Once()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: groupID,
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Node: nodeID1, Identity: "localorg"},
				{Node: nodeID2, Identity: "remoteorg"},
			},
		},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, nodeID1).Return(&fftypes.Node{
		ID: nodeID1, Name: "node1", Owner: "localorg", DX: fftypes.DXInfo{Peer: "peer1-local"},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, nodeID2).Return(&fftypes.Node{
		ID: nodeID2, Name: "node2", Owner: "org1", DX: fftypes.DXInfo{Peer: "peer2-remote"},
	}, nil).Once()

	err := pm.sendUnpinnedMessage(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "localorg",
			},
			TxType: fftypes.TransactionTypeNone,
			Group:  groupID,
		},
	})
	assert.Regexp(t, "FF10137", err)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestSendUnpinnedMessageGetDataFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewRandB32()
	nodeID1 := fftypes.NewUUID()
	nodeID2 := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return(nil, false, fmt.Errorf("pop")).Once()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: groupID,
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Node: nodeID1, Identity: "localorg"},
				{Node: nodeID2, Identity: "remoteorg"},
			},
		},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, nodeID1).Return(&fftypes.Node{
		ID: nodeID1, Name: "node1", Owner: "localorg", DX: fftypes.DXInfo{Peer: "peer1-local"},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, nodeID2).Return(&fftypes.Node{
		ID: nodeID2, Name: "node2", Owner: "org1", DX: fftypes.DXInfo{Peer: "peer2-remote"},
	}, nil).Once()

	err := pm.sendUnpinnedMessage(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "localorg",
			},
			TxType: fftypes.TransactionTypeNone,
			Group:  groupID,
		},
	})
	assert.Regexp(t, "pop", err)

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestSendUnpinnedMessageGroupLookupFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewRandB32()
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(nil, fmt.Errorf("pop")).Once()

	err := pm.sendUnpinnedMessage(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			Identity: fftypes.Identity{
				Author: "org1",
			},
			TxType: fftypes.TransactionTypeNone,
			Group:  groupID,
		},
	})
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)

}

func TestSendUnpinnedMessageInsertFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", pm.ctx, mock.MatchedBy(func(identity *fftypes.Identity) bool {
		assert.Empty(t, identity.Author)
		return true
	})).Return(nil)

	dataID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", pm.ctx, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		err := a[1].(func(context.Context) error)(a[0].(context.Context))
		rag.ReturnArguments = mock.Arguments{err}
	}
	mdi.On("InsertMessageLocal", pm.ctx, mock.Anything).Return(fmt.Errorf("pop")).Once()

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeNone,
				Group:  groupID,
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
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

func TestSendUnpinnedMessageResolveGroupFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", pm.ctx, mock.Anything).Return(nil)

	dataID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", pm.ctx, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		err := a[1].(func(context.Context) error)(a[0].(context.Context))
		rag.ReturnArguments = mock.Arguments{err}
	}
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(nil, fmt.Errorf("pop")).Once()
	mdi.On("InsertMessageLocal", pm.ctx, mock.Anything).Return(nil).Once()

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, "peer2-remote", mock.Anything).Return("tracking1", nil).Once()

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeNone,
				Group:  groupID,
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
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

func TestSendUnpinnedMessageEventFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputIdentity", pm.ctx, mock.Anything).Return(nil)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)

	dataID := fftypes.NewUUID()
	groupID := fftypes.NewRandB32()
	nodeID1 := fftypes.NewUUID()
	nodeID2 := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInlineDataPrivate", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: dataID, Value: fftypes.Byteable(`{"some": "data"}`)},
	}, true, nil).Once()

	mdi := pm.database.(*databasemocks.Plugin)
	rag := mdi.On("RunAsGroup", pm.ctx, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		err := a[1].(func(context.Context) error)(a[0].(context.Context))
		rag.ReturnArguments = mock.Arguments{err}
	}
	mdi.On("GetGroupByHash", pm.ctx, groupID).Return(&fftypes.Group{
		Hash: groupID,
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				{Node: nodeID1, Identity: "localorg"},
				{Node: nodeID2, Identity: "remoteorg"},
			},
		},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, nodeID1).Return(&fftypes.Node{
		ID: nodeID1, Name: "node1", Owner: "localorg", DX: fftypes.DXInfo{Peer: "peer1-local"},
	}, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, nodeID2).Return(&fftypes.Node{
		ID: nodeID2, Name: "node2", Owner: "org1", DX: fftypes.DXInfo{Peer: "peer2-remote"},
	}, nil).Once()
	mdi.On("InsertMessageLocal", pm.ctx, mock.Anything).Return(nil).Once()
	mdi.On("InsertEvent", pm.ctx, mock.Anything).Return(fmt.Errorf("pop")).Once()

	mdx := pm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("SendMessage", pm.ctx, "peer2-remote", mock.Anything).Return("tracking1", nil).Once()

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				TxType: fftypes.TransactionTypeNone,
				Group:  groupID,
			},
		},
		InlineData: fftypes.InlineData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
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
