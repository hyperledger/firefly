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

package definitions

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleDefinitionBroadcastNodeOk(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x23456", "node1").Return(nil, nil)
	mdi.On("GetNodeByID", mock.Anything, node.ID).Return(nil, nil)
	mdi.On("UpsertNode", mock.Anything, mock.Anything, true).Return(nil)
	mdx := dh.exchange.(*dataexchangemocks.Plugin)
	mdx.On("AddPeer", mock.Anything, node.DX).Return(nil)
	action, ba, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	err = ba.PreFinalize(context.Background())
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNodeUpsertFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x23456", "node1").Return(nil, nil)
	mdi.On("GetNodeByID", mock.Anything, node.ID).Return(nil, nil)
	mdi.On("UpsertNode", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNodeAddPeerFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x23456", "node1").Return(nil, nil)
	mdi.On("GetNodeByID", mock.Anything, node.ID).Return(nil, nil)
	mdi.On("UpsertNode", mock.Anything, mock.Anything, true).Return(nil)
	mdx := dh.exchange.(*dataexchangemocks.Plugin)
	mdx.On("AddPeer", mock.Anything, node.DX).Return(fmt.Errorf("pop"))
	action, ba, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)
	err = ba.PreFinalize(context.Background())
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNodeDupMismatch(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x23456", "node1").Return(&fftypes.Node{Owner: "0x99999"}, nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNodeDupOK(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x23456", "node1").Return(&fftypes.Node{Owner: "0x23456"}, nil)
	mdi.On("UpsertNode", mock.Anything, mock.Anything, true).Return(nil)
	mdx := dh.exchange.(*dataexchangemocks.Plugin)
	mdx.On("AddPeer", mock.Anything, node.DX).Return(nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNodeGetFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetNode", mock.Anything, "0x23456", "node1").Return(nil, fmt.Errorf("pop"))
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNodeBadAuthor(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x12345",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNodeGetOrgNotFound(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(nil, nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNodeGetOrgFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(nil, fmt.Errorf("pop"))
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNodeValidateFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	node := &fftypes.Node{
		ID:          fftypes.NewUUID(),
		Name:        "node1",
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
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleDefinitionBroadcastNodeUnmarshalFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtr(`!json`),
	}

	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			IdentityRef: fftypes.IdentityRef{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineNode),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}
