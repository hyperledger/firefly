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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestResolveMemberListNewGroupE2E(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	nodeIDRemote := fftypes.NewUUID()
	nodeIDLocal := fftypes.NewUUID()
	orgID := fftypes.NewUUID()
	var dataID *fftypes.UUID
	mdi.On("GetOrganizationByName", pm.ctx, mock.Anything).Return(&fftypes.Organization{ID: orgID, Identity: "remoteorg"}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: nodeIDRemote, Name: "node2", Owner: "remoteorg"}}, nil).Once()
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: nodeIDLocal, Name: "node1", Owner: "localorg"}}, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{}, nil)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, true).Return(nil)
	ud := mdi.On("UpsertData", pm.ctx, mock.Anything, true, false).Return(nil)
	ud.RunFn = func(a mock.Arguments) {
		data := a[1].(*fftypes.Data)
		assert.Equal(t, fftypes.ValidatorTypeSystemDefinition, data.Validator)
		assert.Equal(t, "ns1", data.Namespace)
		var group fftypes.Group
		err := json.Unmarshal(data.Value, &group)
		assert.NoError(t, err)
		assert.Len(t, group.Members, 2)
		// Note localorg comes first, as we sort groups before hashing
		assert.Equal(t, "localorg", group.Members[0].Identity)
		assert.Equal(t, *nodeIDLocal, *group.Members[0].Node)
		assert.Equal(t, "remoteorg", group.Members[1].Identity)
		assert.Equal(t, *nodeIDRemote, *group.Members[1].Node)
		assert.Nil(t, group.Ledger)
		dataID = data.ID
	}
	um := mdi.On("InsertMessageLocal", pm.ctx, mock.Anything).Return(nil).Once()
	um.RunFn = func(a mock.Arguments) {
		msg := a[1].(*fftypes.Message)
		assert.Equal(t, fftypes.MessageTypeGroupInit, msg.Header.Type)
		assert.Equal(t, "ns1", msg.Header.Namespace)
		assert.Len(t, msg.Data, 1)
		assert.Equal(t, *dataID, *msg.Data[0].ID)
	}

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{Identifier: "0x12345"}, &fftypes.MessageInput{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Namespace: "ns1",
			},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "remoteorg"},
			},
		},
	})
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

}

func TestResolveMemberListExistingGroup(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(&fftypes.Organization{ID: fftypes.NewUUID()}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"}}, nil)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil)

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{Identifier: "0x12345"}, &fftypes.MessageInput{
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

}

func TestResolveMemberListGetGroupsFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(&fftypes.Organization{ID: fftypes.NewUUID()}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"}}, nil)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{Identifier: "0x12345"}, &fftypes.MessageInput{
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)

}

func TestResolveMemberListMissingLocalMemberLookupFailed(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(&fftypes.Organization{ID: fftypes.NewUUID()}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: fftypes.NewUUID(), Name: "node2", Owner: "org1"}}, nil).Once()
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop")).Once()

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{Identifier: "0x12345"}, &fftypes.MessageInput{
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.Regexp(t, "pop", err)
	mdi.AssertExpectations(t)

}

func TestResolveMemberListNodeNotFound(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(&fftypes.Organization{ID: fftypes.NewUUID()}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{}, nil)

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{Identifier: "0x12345"}, &fftypes.MessageInput{
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.Regexp(t, "FF10233", err)
	mdi.AssertExpectations(t)

}

func TestResolveMemberOrgNameNotFound(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(nil, nil)
	mdi.On("GetOrganizationByIdentity", pm.ctx, "org1").Return(nil, nil)

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{Identifier: "0x12345"}, &fftypes.MessageInput{
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.Regexp(t, "FF10223", err)
	mdi.AssertExpectations(t)

}

func TestResolveMemberNodeOwnedParentOrg(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	orgID := fftypes.NewUUID()
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Parent: "id-org2"}, nil)
	mdi.On("GetOrganizationByIdentity", pm.ctx, "id-org2").Return(&fftypes.Organization{ID: orgID}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{}, nil).Once()
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"}}, nil)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{{Hash: fftypes.NewRandB32()}}, nil)

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{Identifier: "0x12345"}, &fftypes.MessageInput{
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.NoError(t, err)
	mdi.AssertExpectations(t)

}

func TestResolveOrgFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(nil, fmt.Errorf("pop"))

	_, err := pm.resolveOrg(pm.ctx, "org1")
	assert.Regexp(t, "pop", err)
	mdi.AssertExpectations(t)

}

func TestResolveOrgByIDFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	orgID := fftypes.NewUUID()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByID", pm.ctx, uuidMatches(orgID)).Return(&fftypes.Organization{ID: orgID}, nil)

	org, err := pm.resolveOrg(pm.ctx, orgID.String())
	assert.NoError(t, err)
	assert.Equal(t, *orgID, *org.ID)

}

func TestGetNodeFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNode", pm.ctx, "org1", "id-node1").Return(nil, fmt.Errorf("pop"))

	_, err := pm.resolveNode(pm.ctx, &fftypes.Organization{Identity: "org1"}, "id-node1")
	assert.Regexp(t, "pop", err)
	mdi.AssertExpectations(t)

}

func TestResolveNodeByIDNoResult(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodeByID", pm.ctx, mock.Anything).Return(nil, nil)

	_, err := pm.resolveNode(pm.ctx, &fftypes.Organization{}, fftypes.NewUUID().String())
	assert.Regexp(t, "FF10224", err)
	mdi.AssertExpectations(t)

}

func TestResolveReceipientListExisting(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{}, &fftypes.MessageInput{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Group: fftypes.NewRandB32(),
			},
		},
	})
	assert.NoError(t, err)
}

func TestResolveReceipientListEmptyList(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{}, &fftypes.MessageInput{})
	assert.Regexp(t, "FF10219", err)
}

func TestResolveLocalNodeCached(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	pm.localNodeID = fftypes.NewUUID()

	ni, err := pm.resolveLocalNode(pm.ctx)
	assert.NoError(t, err)
	assert.Equal(t, pm.localNodeID, ni)
}

func TestResolveLocalNodeNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{}, nil)

	_, err := pm.resolveLocalNode(pm.ctx)
	assert.Regexp(t, "FF10225", err)
}

func TestResolveLocalNodeNotError(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := pm.resolveLocalNode(pm.ctx)
	assert.EqualError(t, err, "pop")
}
