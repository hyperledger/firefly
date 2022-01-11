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

func TestHandleDefinitionBroadcastNSOk(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNSEventFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNSUpsertFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastNSMissingData(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleDefinitionBroadcastNSBadID(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	ns := &fftypes.Namespace{}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleDefinitionBroadcastNSBadData(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtr(`!{json`),
	}

	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleDefinitionBroadcastDuplicate(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastDuplicateOverrideLocal(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
		Type: fftypes.NamespaceTypeLocal,
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	mdi.On("DeleteNamespace", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastDuplicateOverrideLocalFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
		Type: fftypes.NamespaceTypeLocal,
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	mdi.On("DeleteNamespace", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleDefinitionBroadcastDupCheckFail(t *testing.T) {
	dh := newTestDefinitionHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.JSONAnyPtrBytes(b),
	}

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, fmt.Errorf("pop"))
	action, err := dh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}
