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

	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/datamocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSendMessageE2EOk(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mii := pm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", pm.ctx, "localorg").Return(&fftypes.Identity{
		Identifier: "localorg",
		OnChain:    "0x12345",
	}, nil)

	dataID := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInputData", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
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
	}, nil).Once()
	mdi.On("GetOrganizationByName", pm.ctx, "org1").Return(&fftypes.Organization{
		ID: fftypes.NewUUID(),
	}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{
		{ID: fftypes.NewUUID(), Name: "node1", Owner: "org1"},
	}, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil).Once()
	mdi.On("InsertMessageLocal", pm.ctx, mock.Anything).Return(nil).Once()

	msg, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInput{
		InputData: fftypes.InputData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, *dataID, *msg.Data[0].ID)
	assert.NotNil(t, msg.Header.Group)

}

func TestSendMessageBadIdentity(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mii := pm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", pm.ctx, "localorg").Return(nil, fmt.Errorf("pop"))

	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInput{
		InputData: fftypes.InputData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.Regexp(t, "FF10206.*pop", err)

}

func TestSendMessageFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mii := pm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", pm.ctx, "localorg").Return(&fftypes.Identity{
		Identifier: "localorg",
		OnChain:    "0x12345",
	}, nil)

	dataID := fftypes.NewUUID()
	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInputData", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID, Hash: fftypes.NewRandB32()},
	}, nil)

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("RunAsGroup", pm.ctx, mock.Anything).Return(fmt.Errorf("pop"))
	_, err := pm.SendMessage(pm.ctx, "ns1", &fftypes.MessageInput{
		InputData: fftypes.InputData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
		},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "org1"},
			},
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestResolveAndSendBadMembers(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	err := pm.resolveAndSend(pm.ctx, &fftypes.Identity{}, &fftypes.MessageInput{
		InputData: fftypes.InputData{
			{Value: fftypes.Byteable(`{"some": "data"}`)},
		},
	})
	assert.Regexp(t, "FF10219", err)

}

func TestResolveAndSendBadInputData(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "localorg").Return(&fftypes.Organization{
		ID: fftypes.NewUUID(),
	}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{
		{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"},
	}, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil).Once()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInputData", pm.ctx, "ns1", mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := pm.resolveAndSend(pm.ctx, &fftypes.Identity{}, &fftypes.MessageInput{
		Message: fftypes.Message{Header: fftypes.MessageHeader{Namespace: "ns1"}},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "localorg"},
			},
		},
	})
	assert.Regexp(t, "pop", err)

}

func TestResolveAndSendSealFail(t *testing.T) {

	pm, cancel := newTestPrivateMessaging(t)
	defer cancel()

	mdi := pm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByName", pm.ctx, "localorg").Return(&fftypes.Organization{
		ID: fftypes.NewUUID(),
	}, nil)
	mdi.On("GetNodes", pm.ctx, mock.Anything).Return([]*fftypes.Node{
		{ID: fftypes.NewUUID(), Name: "node1", Owner: "localorg"},
	}, nil).Once()
	mdi.On("GetGroups", pm.ctx, mock.Anything).Return([]*fftypes.Group{
		{Hash: fftypes.NewRandB32()},
	}, nil).Once()

	mdm := pm.data.(*datamocks.Manager)
	mdm.On("ResolveInputData", pm.ctx, "ns1", mock.Anything).Return(fftypes.DataRefs{
		{ /* missing */ },
	}, nil)

	err := pm.resolveAndSend(pm.ctx, &fftypes.Identity{}, &fftypes.MessageInput{
		Message: fftypes.Message{Header: fftypes.MessageHeader{Namespace: "ns1"}},
		Group: &fftypes.InputGroup{
			Members: []fftypes.MemberInput{
				{Identity: "localorg"},
			},
		},
	})
	assert.Regexp(t, "FF10144", err)

}
