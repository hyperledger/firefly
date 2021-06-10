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
	"github.com/hyperledger-labs/firefly/mocks/datamocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGroupInitSealFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	err := pm.groupInit(pm.ctx, &fftypes.Identity{}, nil)
	assert.Regexp(t, "FF10137", err)
}

func TestGroupInitWriteGroupFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))

	err := pm.groupInit(pm.ctx, &fftypes.Identity{}, &fftypes.Group{})
	assert.Regexp(t, "pop", err)
}

func TestGroupInitWriteDataFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", mock.Anything, mock.Anything, true).Return(nil)
	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := pm.groupInit(pm.ctx, &fftypes.Identity{}, &fftypes.Group{})
	assert.Regexp(t, "pop", err)
}

func TestResolveInitGroupMissingData(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{}, false, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     fftypes.NewRandB32(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupBadData(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`!json`)},
	}, true, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     fftypes.NewRandB32(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupBadValidation(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`{}`)},
	}, true, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     fftypes.NewRandB32(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupBadGroupID(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	group := &fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Name:      "group1",
			Namespace: "ns1",
			Members: fftypes.Members{
				{Identity: "abce12345", Node: fftypes.NewUUID()},
			},
		},
	}
	group.Seal()
	assert.NoError(t, group.Validate(pm.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(b)},
	}, true, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     fftypes.NewRandB32(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupUpsertFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	group := &fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Name:      "group1",
			Namespace: "ns1",
			Members: fftypes.Members{
				{Identity: "abce12345", Node: fftypes.NewUUID()},
			},
		},
	}
	group.Seal()
	assert.NoError(t, group.Validate(pm.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(b)},
	}, true, nil)
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, true).Return(fmt.Errorf("pop"))

	_, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     group.Hash,
			Author:    "author1",
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestResolveInitGroupNewOk(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	group := &fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Name:      "group1",
			Namespace: "ns1",
			Members: fftypes.Members{
				{Identity: "abce12345", Node: fftypes.NewUUID()},
			},
		},
	}
	group.Seal()
	assert.NoError(t, group.Validate(pm.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(b)},
	}, true, nil)
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, true).Return(nil)
	mdi.On("UpsertEvent", pm.ctx, mock.Anything, false).Return(nil)

	group, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     group.Hash,
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupNewEventFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	group := &fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Name:      "group1",
			Namespace: "ns1",
			Members: fftypes.Members{
				{Identity: "abce12345", Node: fftypes.NewUUID()},
			},
		},
	}
	group.Seal()
	assert.NoError(t, group.Validate(pm.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageData", pm.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(b)},
	}, true, nil)
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, true).Return(nil)
	mdi.On("UpsertEvent", pm.ctx, mock.Anything, false).Return(fmt.Errorf("pop"))

	_, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     group.Hash,
			Author:    "author1",
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestResolveInitGroupExistingOK(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, true).Return(nil)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(&fftypes.Group{}, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Tag:       "mytag",
			Group:     fftypes.NewRandB32(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)
}

func TestResolveInitGroupExistingFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Tag:       "mytag",
			Group:     fftypes.NewRandB32(),
			Author:    "author1",
		},
	})
	assert.EqualError(t, err, "pop")
}

func TestResolveInitGroupExistingNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, nil)

	group, err := pm.ResolveInitGroup(pm.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Tag:       "mytag",
			Group:     fftypes.NewRandB32(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)
	assert.Nil(t, group)
}

func TestGetGroupByIDOk(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewRandB32()
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(&fftypes.Group{Hash: groupID}, nil)

	group, err := pm.GetGroupByID(pm.ctx, groupID.String())
	assert.NoError(t, err)
	assert.Equal(t, *groupID, *group.Hash)
}

func TestGetGroupByIDBadID(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()
	_, err := pm.GetGroupByID(pm.ctx, "!wrong")
	assert.Regexp(t, "FF10232", err)
}

func TestGetGroupsOk(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{}, nil)

	fb := database.GroupQueryFactory.NewFilter(pm.ctx)
	groups, err := pm.GetGroups(pm.ctx, fb.And(fb.Eq("description", "mygroup")))
	assert.NoError(t, err)
	assert.Empty(t, groups)
}

func TestGetGroupNodesCache(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				&fftypes.Member{Node: node1},
			},
		},
	}
	group.Seal()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(group, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, mock.Anything).Return(&fftypes.Node{
		ID: node1,
	}, nil).Once()

	nodes, err := pm.getGroupNodes(pm.ctx, group.Hash)
	assert.NoError(t, err)
	assert.Equal(t, *node1, *nodes[0].ID)

	// Note this validates the cache as we only mocked the calls once
	nodes, err = pm.getGroupNodes(pm.ctx, group.Hash)
	assert.NoError(t, err)
	assert.Equal(t, *node1, *nodes[0].ID)
}

func TestGetGroupNodesGetGroupFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewRandB32()
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := pm.getGroupNodes(pm.ctx, groupID)
	assert.EqualError(t, err, "pop")
}

func TestGetGroupNodesGetGroupNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewRandB32()
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, nil)

	_, err := pm.getGroupNodes(pm.ctx, groupID)
	assert.Regexp(t, "FF10226", err)
}

func TestGetGroupNodesNodeLookupFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				&fftypes.Member{Node: node1},
			},
		},
	}
	group.Seal()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(group, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, uuidMatches(node1)).Return(nil, fmt.Errorf("pop")).Once()

	_, err := pm.getGroupNodes(pm.ctx, group.Hash)
	assert.EqualError(t, err, "pop")
}

func TestGetGroupNodesNodeLookupNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &fftypes.Group{
		GroupIdentity: fftypes.GroupIdentity{
			Members: fftypes.Members{
				&fftypes.Member{Node: node1},
			},
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(group, nil).Once()
	mdi.On("GetNodeByID", pm.ctx, uuidMatches(node1)).Return(nil, nil).Once()

	_, err := pm.getGroupNodes(pm.ctx, group.Hash)
	assert.Regexp(t, "FF10224", err)
}
