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

	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/identitymocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleSystemBroadcastOrgOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(nil, nil)
	mdi.On("GetOrganizationByName", mock.Anything, "org1").Return(nil, nil)
	mdi.On("GetOrganizationByID", mock.Anything, org.ID).Return(nil, nil)
	mdi.On("UpsertOrganization", mock.Anything, mock.Anything, true).Return(nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.True(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastOrgDupOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x12345", Parent: "0x23456"}, nil)
	mdi.On("UpsertOrganization", mock.Anything, mock.Anything, true).Return(nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.True(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastOrgDupMismatch(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x23456").Return(&fftypes.Identity{OnChain: "0x23456"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x23456"}, nil)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(&fftypes.Organization{ID: fftypes.NewUUID(), Identity: "0x12345", Parent: "0x9999"}, nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastOrgUpsertFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x12345").Return(&fftypes.Identity{OnChain: "0x12345"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(nil, nil)
	mdi.On("GetOrganizationByName", mock.Anything, "org1").Return(nil, nil)
	mdi.On("GetOrganizationByID", mock.Anything, org.ID).Return(nil, nil)
	mdi.On("UpsertOrganization", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x12345",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastOrgGetOrgFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x12345").Return(&fftypes.Identity{OnChain: "0x12345"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x12345").Return(nil, fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x12345",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastOrgAuthorMismatch(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x12345").Return(&fftypes.Identity{OnChain: "0x12345"}, nil)
	mdi := bm.database.(*databasemocks.Plugin)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastOrgResolveFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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
		Value: fftypes.Byteable(b),
	}

	mii := bm.identity.(*identitymocks.Plugin)
	mii.On("Resolve", mock.Anything, "0x12345").Return(nil, fmt.Errorf("pop"))
	mdi := bm.database.(*databasemocks.Plugin)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mii.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastGetParentFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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
		Value: fftypes.Byteable(b),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(nil, fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastGetParentNotFound(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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
		Value: fftypes.Byteable(b),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetOrganizationByIdentity", mock.Anything, "0x23456").Return(nil, nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastValidateFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	org := &fftypes.Organization{
		ID:          fftypes.NewUUID(),
		Name:        "org1",
		Identity:    "0x12345",
		Description: string(make([]byte, 4097)),
	}
	b, err := json.Marshal(&org)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastUnmarshalFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	data := &fftypes.Data{
		Value: fftypes.Byteable(`!json`),
	}

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Author:    "0x23456",
			Tag:       string(fftypes.SystemTagDefineOrganization),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)
}
