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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGroupInitSealFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	err := pm.groupInit(pm.ctx, &core.SignerRef{}, &core.Group{})
	assert.Regexp(t, "FF10137", err)
}

func TestGroupInitWriteGroupFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Namespace: "ns1",
			Members: core.Members{
				{Identity: "id1", Node: fftypes.NewUUID()},
			},
		},
	}
	group.Seal()
	err := pm.groupInit(pm.ctx, &core.SignerRef{}, group)
	assert.Regexp(t, "pop", err)
}

func TestGroupInitWriteDataFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Namespace: "ns1",
			Members: core.Members{
				{Identity: "id1", Node: fftypes.NewUUID()},
			},
		},
	}
	group.Seal()
	err := pm.groupInit(pm.ctx, &core.SignerRef{}, group)
	assert.Regexp(t, "pop", err)
}

func TestResolveInitGroupMissingData(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", pm.ctx, mock.Anything).Return(core.DataArray{}, false, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: core.SystemNamespace,
			Tag:       core.SystemTagDefineGroup,
			Group:     fftypes.NewRandB32(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupBadData(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", pm.ctx, mock.Anything).Return(core.DataArray{
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`!json`)},
	}, true, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: core.SystemNamespace,
			Tag:       core.SystemTagDefineGroup,
			Group:     fftypes.NewRandB32(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupBadValidation(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", pm.ctx, mock.Anything).Return(core.DataArray{
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtr(`{}`)},
	}, true, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: core.SystemNamespace,
			Tag:       core.SystemTagDefineGroup,
			Group:     fftypes.NewRandB32(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupBadGroupID(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Name:      "group1",
			Namespace: "ns1",
			Members: core.Members{
				{Identity: "abce12345", Node: fftypes.NewUUID()},
			},
		},
	}
	group.Seal()
	assert.NoError(t, group.Validate(pm.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", pm.ctx, mock.Anything).Return(core.DataArray{
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtrBytes(b)},
	}, true, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: core.SystemNamespace,
			Tag:       core.SystemTagDefineGroup,
			Group:     fftypes.NewRandB32(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupUpsertFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Name:      "group1",
			Namespace: "ns1",
			Members: core.Members{
				{Identity: "abce12345", Node: fftypes.NewUUID()},
			},
		},
	}
	group.Seal()
	assert.NoError(t, group.Validate(pm.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", pm.ctx, mock.Anything).Return(core.DataArray{
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtrBytes(b)},
	}, true, nil)
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	_, err := pm.ResolveInitGroup(pm.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: core.SystemNamespace,
			Tag:       core.SystemTagDefineGroup,
			Group:     group.Hash,
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestResolveInitGroupNewOk(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Name:      "group1",
			Namespace: "ns1",
			Members: core.Members{
				{Identity: "abce12345", Node: fftypes.NewUUID()},
			},
		},
	}
	group.Seal()
	assert.NoError(t, group.Validate(pm.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("GetMessageDataCached", pm.ctx, mock.Anything).Return(core.DataArray{
		{ID: fftypes.NewUUID(), Value: fftypes.JSONAnyPtrBytes(b)},
	}, true, nil)
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdi.On("InsertEvent", pm.ctx, mock.Anything).Return(nil)

	group, err := pm.ResolveInitGroup(pm.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: core.SystemNamespace,
			Tag:       core.SystemTagDefineGroup,
			Group:     group.Hash,
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupExistingOK(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(&core.Group{}, nil)

	_, err := pm.ResolveInitGroup(pm.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Tag:       "mytag",
			Group:     fftypes.NewRandB32(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	})
	assert.NoError(t, err)
}

func TestResolveInitGroupExistingFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := pm.ResolveInitGroup(pm.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Tag:       "mytag",
			Group:     fftypes.NewRandB32(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
		},
	})
	assert.EqualError(t, err, "pop")
}

func TestResolveInitGroupExistingNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, nil)

	group, err := pm.ResolveInitGroup(pm.ctx, &core.Message{
		Header: core.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: "ns1",
			Tag:       "mytag",
			Group:     fftypes.NewRandB32(),
			SignerRef: core.SignerRef{
				Author: "author1",
				Key:    "0x12345",
			},
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
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(&core.Group{Hash: groupID}, nil)

	group, err := pm.GetGroupByID(pm.ctx, groupID.String())
	assert.NoError(t, err)
	assert.Equal(t, *groupID, *group.Hash)
}

func TestGetGroupByIDBadID(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()
	_, err := pm.GetGroupByID(pm.ctx, "!wrong")
	assert.Regexp(t, "FF00107", err)
}

func TestGetGroupsOk(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*core.Group{}, nil, nil)

	fb := database.GroupQueryFactory.NewFilter(pm.ctx)
	groups, _, err := pm.GetGroups(pm.ctx, fb.And(fb.Eq("description", "mygroup")))
	assert.NoError(t, err)
	assert.Empty(t, groups)
}

func TestGetGroupsNSOk(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroups", pm.ctx, mock.MatchedBy(func(filter database.AndFilter) bool {
		f, err := filter.Finalize()
		assert.NoError(t, err)
		assert.Contains(t, f.String(), "namespace")
		assert.Contains(t, f.String(), "ns1")
		return true
	})).Return([]*core.Group{}, nil, nil)

	fb := database.GroupQueryFactory.NewFilter(pm.ctx)
	groups, _, err := pm.GetGroupsNS(pm.ctx, "ns1", fb.And(fb.Eq("description", "mygroup")))
	assert.NoError(t, err)
	assert.Empty(t, groups)
}

func TestGetGroupNodesCache(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				&core.Member{Node: node1},
			},
		},
	}
	group.Seal()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(group, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, mock.Anything).Return(&core.Identity{
		IdentityBase: core.IdentityBase{
			ID:   node1,
			Type: core.IdentityTypeNode,
		},
	}, nil).Once()

	g, nodes, err := pm.getGroupNodes(pm.ctx, group.Hash, false)
	assert.NoError(t, err)
	assert.Equal(t, *node1, *nodes[0].ID)
	assert.Equal(t, *group.Hash, *g.Hash)

	// Note this validates the cache as we only mocked the calls once
	g, nodes, err = pm.getGroupNodes(pm.ctx, group.Hash, false)
	assert.NoError(t, err)
	assert.Equal(t, *node1, *nodes[0].ID)
	assert.Equal(t, *group.Hash, *g.Hash)
}

func TestGetGroupNodesGetGroupFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewRandB32()
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, _, err := pm.getGroupNodes(pm.ctx, groupID, false)
	assert.EqualError(t, err, "pop")
}

func TestGetGroupNodesGetGroupNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewRandB32()
	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, nil)

	_, _, err := pm.getGroupNodes(pm.ctx, groupID, false)
	assert.Regexp(t, "FF10226", err)
}

func TestGetGroupNodesNodeLookupFail(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				&core.Member{Node: node1},
			},
		},
	}
	group.Seal()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(group, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, node1).Return(nil, fmt.Errorf("pop")).Once()

	_, _, err := pm.getGroupNodes(pm.ctx, group.Hash, false)
	assert.EqualError(t, err, "pop")
}

func TestGetGroupNodesNodeLookupNotFound(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				&core.Member{Node: node1},
			},
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(group, nil).Once()
	mdi.On("GetIdentityByID", pm.ctx, node1).Return(nil, nil).Once()

	_, _, err := pm.getGroupNodes(pm.ctx, group.Hash, false)
	assert.Regexp(t, "FF10224", err)
}

func TestEnsureLocalGroupNewOk(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Namespace: "ns1",
			Members: core.Members{
				&core.Member{Node: node1, Identity: "id1"},
			},
		},
	}
	group.Seal()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, nil)
	mdi.On("UpsertGroup", pm.ctx, group, database.UpsertOptimizationNew).Return(nil)

	ok, err := pm.EnsureLocalGroup(pm.ctx, group)
	assert.NoError(t, err)
	assert.True(t, ok)

	mdi.AssertExpectations(t)
}

func TestEnsureLocalGroupNil(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	ok, err := pm.EnsureLocalGroup(pm.ctx, nil)
	assert.Regexp(t, "FF10344", err)
	assert.False(t, ok)
}

func TestEnsureLocalGroupExistingOk(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				&core.Member{Node: node1},
			},
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(group, nil)

	ok, err := pm.EnsureLocalGroup(pm.ctx, group)
	assert.NoError(t, err)
	assert.True(t, ok)

	mdi.AssertExpectations(t)
}

func TestEnsureLocalGroupLookupErr(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Members: core.Members{
				&core.Member{Node: node1},
			},
		},
	}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	ok, err := pm.EnsureLocalGroup(pm.ctx, group)
	assert.EqualError(t, err, "pop")
	assert.False(t, ok)

	mdi.AssertExpectations(t)
}

func TestEnsureLocalGroupInsertErr(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	node1 := fftypes.NewUUID()
	group := &core.Group{
		GroupIdentity: core.GroupIdentity{
			Namespace: "ns1",
			Members: core.Members{
				&core.Member{Node: node1, Identity: "id1"},
			},
		},
	}
	group.Seal()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, nil)
	mdi.On("UpsertGroup", pm.ctx, mock.Anything, database.UpsertOptimizationNew).Return(fmt.Errorf("pop"))

	ok, err := pm.EnsureLocalGroup(pm.ctx, group)
	assert.EqualError(t, err, "pop")
	assert.False(t, ok)

	mdi.AssertExpectations(t)
}

func TestEnsureLocalGroupBadGroup(t *testing.T) {
	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	group := &core.Group{}

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetGroupByHash", pm.ctx, mock.Anything).Return(nil, nil)

	ok, err := pm.EnsureLocalGroup(pm.ctx, group)
	assert.NoError(t, err)
	assert.False(t, ok)

	mdi.AssertExpectations(t)
}
