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

package broadcast

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/mocks/dataexchangemocks"
	"github.com/kaleido-io/firefly/mocks/identitymocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleSystemBroadcastNodeOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x12345").Return(nil, nil)
	mdi.On("GetNodeByID", mock.Anything, node.ID).Return(nil, nil)
	mdi.On("UpsertNode", mock.Anything, mock.Anything, true).Return(nil)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("AddPeer", mock.Anything, mock.Anything).Return(nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.True(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeUpsertFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x12345").Return(nil, nil)
	mdi.On("GetNodeByID", mock.Anything, node.ID).Return(nil, nil)
	mdi.On("UpsertNode", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeAddPeerFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x12345").Return(nil, nil)
	mdi.On("GetNodeByID", mock.Anything, node.ID).Return(nil, nil)
	mdi.On("UpsertNode", mock.Anything, mock.Anything, true).Return(nil)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("AddPeer", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeDupMismatch(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x12345").Return(&fftypes.Node{Owner: "0x99999"}, nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeDupOK(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x12345").Return(&fftypes.Node{Owner: "0x23456"}, nil)
	mdi.On("UpsertNode", mock.Anything, mock.Anything, true).Return(nil)
	mdx := bm.exchange.(*dataexchangemocks.Plugin)
	mdx.On("AddPeer", mock.Anything, mock.Anything).Return(nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.True(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeGetFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x12345").Return(nil, fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeBadAuthor(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x99999",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeResolveFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(nil, fmt.Errorf("pop"))
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeGetOrgNotFound(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(nil, nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeGetOrgFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: "my org",
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(nil, fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNodeValidateFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Identity:    "0x12345",
		Owner:       "0x23456",
		Description: string(make([]byte, 4097)),
		DX: fftypes.DXInfo{
			Peer:     "peer1",
			Endpoint: fftypes.JSONObject{"some": "info"},
		},
	}
	b, err := json.Marshal(&node)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastNodeUnmarshalFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	data := &fftypes.Data{
		Value: fftypes.Byteable(`!json`),
	}

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)
}
