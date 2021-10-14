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

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestResolveMemberListNewGroupE2E(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	nodeIDRemote := fftypes.NewUUID()
	nodeIDLocal := fftypes.NewUUID()
	orgIDLocal := fftypes.NewUUID()
	orgIDRemote := fftypes.NewUUID()

	orgNameRemote := "remoteOrg"

	signingKeyLocal := "localSigningKey"
	signingKeyRemote := "remoteSigningKey"

	orgDIDLocal := "did:firefly:org/" + orgIDLocal.String()
	orgDIDRemote := "did:firefly:org/" + orgIDRemote.String()

	var dataID *fftypes.UUID
	mdi.On("GetOrganizationByName", pm.ctx, orgNameRemote).Return(&fftypes.Organization{ID: orgIDRemote, Identity: signingKeyRemote}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: nodeIDRemote, Name: "node2", Owner: signingKeyRemote}}, nil, nil).Once()
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: nodeIDLocal, Name: "node1", Owner: signingKeyLocal}}, nil, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{}, nil, nil)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, true).Return(nil)
	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return(orgDIDLocal, nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(&fftypes.Organization{Identity: signingKeyLocal}, nil)
	ud := mdi.On("UpsertData", pm.ctx, mock.Anything, true, false).Return(nil)
	ud.RunFn = func(a mock.Arguments) {
		data := a[1].(*fftypes.Data)
		assert.Equal(t, fftypes.ValidatorTypeSystemDefinition, data.Validator)
		assert.Equal(t, "ns1", data.Namespace)
		var group fftypes.Group
		err := json.Unmarshal(data.Value, &group)
		assert.NoError(t, err)
		assert.Len(t, group.Members, 2)
		// Group identiy is sorted by group members DIDs so check them in that order
		if orgDIDLocal < orgDIDRemote {
			assert.Equal(t, orgDIDLocal, group.Members[0].Identity)
			assert.Equal(t, *nodeIDLocal, *group.Members[0].Node)
			assert.Equal(t, orgDIDRemote, group.Members[1].Identity)
			assert.Equal(t, *nodeIDRemote, *group.Members[1].Node)
			assert.Nil(t, group.Ledger)
		} else {
			assert.Equal(t, orgDIDRemote, group.Members[0].Identity)
			assert.Equal(t, *nodeIDRemote, *group.Members[0].Node)
			assert.Equal(t, orgDIDLocal, group.Members[1].Identity)
			assert.Equal(t, *nodeIDLocal, *group.Members[1].Node)
			assert.Nil(t, group.Ledger)
		}

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

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Namespace: "ns1",
				Identity: fftypes.Identity{
					Author: "org1",
				},
			},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: orgNameRemote},
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
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"}}, nil, nil)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil, nil)
	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(&fftypes.Organization{Identity: "localorg"}, nil)

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "org1",
				},
			},
		},
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
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"}}, nil, nil)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))
	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(&fftypes.Organization{Identity: "localorg"}, nil)

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "org1",
				},
			},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.EqualError(t, err, "pop")
	mdi.AssertExpectations(t)

}

func TestResolveMemberListLocalOrgUnregistered(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("", fmt.Errorf("pop"))

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "org1",
				},
			},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestResolveMemberListLocalOrgLookupFailed(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("", nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(nil, fmt.Errorf("pop"))

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "org1",
				},
			},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestResolveMemberListMissingLocalMemberLookupFailed(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(&fftypes.Organization{ID: fftypes.NewUUID()}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: fftypes.NewUUID(), Name: "node2", Owner: "org1"}}, nil, nil).Once()
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop")).Once()
	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(&fftypes.Organization{Identity: "localorg"}, nil)

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "org1",
				},
			},
		},
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
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{}, nil, nil)
	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(&fftypes.Organization{Identity: "localorg"}, nil)

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "org1",
				},
			},
		},
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
	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(&fftypes.Organization{Identity: "localorg"}, nil)

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "org1",
				},
			},
		},
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
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{}, nil, nil).Once()
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"}}, nil, nil)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{{Hash: fftypes.NewRandB32()}}, nil, nil)
	mim := pm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveLocalOrgDID", pm.ctx).Return("localorg", nil)
	mim.On("GetLocalOrganization", pm.ctx).Return(&fftypes.Organization{Identity: "localorg"}, nil)

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Identity: fftypes.Identity{
					Author: "org1",
				},
			},
		},
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
	mdi.On("GetOrganizationByID", pm.ctx, orgID).Return(&fftypes.Organization{ID: orgID}, nil)

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

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{
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

	err := pm.resolveRecipientList(pm.ctx, &fftypes.MessageInOut{})
	assert.Regexp(t, "FF10219", err)
}

func TestResolveLocalNodeCached(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	pm.localNodeID = fftypes.NewUUID()

	ni, err := pm.resolveLocalNode(pm.ctx, "localorg")
	assert.NoError(t, err)
	assert.Equal(t, pm.localNodeID, ni)
}

func TestResolveLocalNodeNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{}, nil, nil)

	_, err := pm.resolveLocalNode(pm.ctx, "localorg")
	assert.Regexp(t, "FF10225", err)
}

func TestResolveLocalNodeNotError(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := pm.resolveLocalNode(pm.ctx, "localorg")
	assert.EqualError(t, err, "pop")
}
