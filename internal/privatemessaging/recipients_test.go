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
	"testing"

	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestResolveReceipientListNewGroupE2E(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	orgID := fftypes.NewUUID()
	nodeID := fftypes.NewUUID()
	var dataID *fftypes.UUID
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(&fftypes.Organization{ID: orgID}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{{ID: nodeID}}, nil)
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{}, nil)
	ud := mdi.On("UpsertData", pm.ctx, mock.Anything, true, false).Return(nil)
	ud.RunFn = func(a mock.Arguments) {
		data := a[1].(*fftypes.Data)
		assert.Equal(t, fftypes.ValidatorTypeSystemDefinition, data.Validator)
		assert.Equal(t, fftypes.SystemNamespace, data.Namespace)
		var group fftypes.Group
		err := json.Unmarshal(data.Value, &group)
		assert.NoError(t, err)
		assert.Len(t, group.Recipients, 1)
		assert.Equal(t, *orgID, *group.Recipients[0].Org)
		assert.Equal(t, *nodeID, *group.Recipients[0].Node)
		assert.Nil(t, group.Ledger)
		dataID = data.ID
	}
	um := mdi.On("UpsertMessage", pm.ctx, mock.Anything, false, false).Return(nil).Once()
	um.RunFn = func(a mock.Arguments) {
		msg := a[1].(*fftypes.Message)
		assert.Equal(t, fftypes.MessageTypeGroupInit, msg.Header.Type)
		assert.Equal(t, fftypes.SystemNamespace, msg.Header.Namespace)
		assert.Len(t, msg.Data, 1)
		assert.Equal(t, *dataID, *msg.Data[0].ID)
	}

	err := pm.resolveReceipientList(pm.ctx, &fftypes.Identity{Identifier: "0x12345"}, &fftypes.MessageInput{
		Recipients: []fftypes.RecipientInput{
			{Org: "org1"},
		},
	})
	assert.NoError(t, err)

}
