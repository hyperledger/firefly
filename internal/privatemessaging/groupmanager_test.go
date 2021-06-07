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

	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/datamocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGroupInitSealFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	err := pm.groupInit(pm.ctx, &fftypes.Identity{}, nil)
	assert.Regexp(t, "FF10137", err)
}

func TestGroupInitWriteFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))

	err := pm.groupInit(pm.ctx, &fftypes.Identity{}, &fftypes.Group{})
	assert.Regexp(t, "pop", err)
}

func TestResolveInitGroupMissingData(t *testing.T) {
	ag, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{}, false, nil)

	_, err := ag.ResolveInitGroup(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     fftypes.NewUUID(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupBadData(t *testing.T) {
	ag, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`!json`)},
	}, true, nil)

	_, err := ag.ResolveInitGroup(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     fftypes.NewUUID(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupBadValidation(t *testing.T) {
	ag, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(`{}`)},
	}, true, nil)

	_, err := ag.ResolveInitGroup(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     fftypes.NewUUID(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupBadGroupID(t *testing.T) {
	ag, cancel := newTestPrivateMessaging(t)
	defer cancel()

	group := &fftypes.Group{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Members: fftypes.Members{
			{Identity: "abce12345", Node: fftypes.NewUUID()},
		},
	}
	assert.NoError(t, group.Validate(ag.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(b)},
	}, true, nil)

	_, err := ag.ResolveInitGroup(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     fftypes.NewUUID(),
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}

func TestResolveInitGroupUpsertFail(t *testing.T) {
	ag, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewUUID()
	group := &fftypes.Group{
		ID:        groupID,
		Namespace: "ns1",
		Members: fftypes.Members{
			{Identity: "abce12345", Node: fftypes.NewUUID()},
		},
	}
	assert.NoError(t, group.Validate(ag.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(b)},
	}, true, nil)
	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", ag.ctx, mock.Anything, true).Return(fmt.Errorf("pop"))

	_, err := ag.ResolveInitGroup(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     groupID,
			Author:    "author1",
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestResolveInitGroupNewOk(t *testing.T) {
	ag, cancel := newTestPrivateMessaging(t)
	defer cancel()

	groupID := fftypes.NewUUID()
	group := &fftypes.Group{
		ID:        groupID,
		Namespace: "ns1",
		Members: fftypes.Members{
			{Identity: "abce12345", Node: fftypes.NewUUID()},
		},
	}
	assert.NoError(t, group.Validate(ag.ctx, true))
	b, _ := json.Marshal(&group)

	mdm := ag.data.(*datamocks.Manager)
	mdm.On("GetMessageData", ag.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: fftypes.NewUUID(), Value: fftypes.Byteable(b)},
	}, true, nil)
	mdi := ag.database.(*databasemocks.Plugin)
	mdi.On("UpsertGroup", ag.ctx, mock.Anything, true).Return(nil)

	_, err := ag.ResolveInitGroup(ag.ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:        fftypes.NewUUID(),
			Namespace: fftypes.SystemNamespace,
			Tag:       string(fftypes.SystemTagDefineGroup),
			Group:     groupID,
			Author:    "author1",
		},
	})
	assert.NoError(t, err)

}
