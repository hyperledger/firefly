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
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleDefinitionBroadcastChildOrgOk(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	parentOrg := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org2",
		Identity:    "0x23456",
		Description: "parent org",
	}

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Parent:      "0x23456",
		Description: "my org",
		Profile:     fftypes.JSONObject{"some": "info"},
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(parentOrg, nil)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(nil, nil)
	mdi.On("GetOrganizationByName", mock.Anything, "org1").Return(nil, nil)
	mdi.On("GetOrganizationByID", mock.Anything, org.ID).Return(nil, nil)
	mdi.On("UpsertOrganization", mock.Anything, mock.Anything, true).Return(nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastChildOrgDupOk(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	parentOrg := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org2",
		Identity:    "0x23456",
		Description: "parent org",
	}

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Parent:      "0x23456",
		Description: "my org",
		Profile:     fftypes.JSONObject{"some": "info"},
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(parentOrg, nil)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(org, nil)
	mdi.On("UpsertOrganization", mock.Anything, mock.Anything, true).Return(nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastChildOrgBadKey(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	parentOrg := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org2",
		Identity:    "0x23456",
		Description: "parent org",
	}

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Parent:      "0x23456",
		Description: "my org",
		Profile:     fftypes.JSONObject{"some": "info"},
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(parentOrg, nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "did:firefly:org/0x23456",
				Key:    "0x34567",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastOrgDupMismatch(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Parent:      "", // the mismatch
		Description: "my org",
		Profile:     fftypes.JSONObject{"some": "info"},
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x12345", Parent: "0x9999"}, nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastOrgUpsertFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Description: "my org",
		Profile:     fftypes.JSONObject{"some": "info"},
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(nil, nil)
	mdi.On("GetOrganizationByName", mock.Anything, "org1").Return(nil, nil)
	mdi.On("GetOrganizationByID", mock.Anything, org.ID).Return(nil, nil)
	mdi.On("UpsertOrganization", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "0x12345",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastOrgGetOrgFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Description: "my org",
		Profile:     fftypes.JSONObject{"some": "info"},
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(nil, fmt.Errorf("pop"))
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "0x12345",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastOrgAuthorMismatch(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Description: "my org",
		Profile:     fftypes.JSONObject{"some": "info"},
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x12345", Parent: "0x9999"}, nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastGetParentFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Parent:      "0x23456",
		Description: "my org",
		Profile:     fftypes.JSONObject{"some": "info"},
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(nil, fmt.Errorf("pop"))
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastGetParentNotFound(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Parent:      "0x23456",
		Description: "my org",
		Profile:     fftypes.JSONObject{"some": "info"},
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(nil, nil)
	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastValidateFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Description: string(make([]byte, 4097)),
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleDefinitionBroadcastUnmarshalFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtr(`!json`),
	}

	action, _, err := dh.HandleDefinitionBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Identity: fftypes.Identity{
				Author: "did:firefly:org/0x23456",
				Key:    "0x23456",
			},
			Tag: string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}
