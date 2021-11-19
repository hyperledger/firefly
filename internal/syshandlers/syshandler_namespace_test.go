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

package syshandlers

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

func TestHandleSystemBroadcastNSOk(t *testing.T) {
	sh := newTestSystemHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNSEventFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNSUpsertFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))
	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNSMissingData(t *testing.T) {
	sh := newTestSystemHandlers(t)

	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastNSBadID(t *testing.T) {
	sh := newTestSystemHandlers(t)

	ns := &fftypes.Namespace{}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastNSBadData(t *testing.T) {
	sh := newTestSystemHandlers(t)

	data := &fftypes.Data{
		Value: fftypes.Byteable(`!{json`),
	}

	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastDuplicate(t *testing.T) {
	sh := newTestSystemHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionReject, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastDuplicateOverrideLocal(t *testing.T) {
	sh := newTestSystemHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
		Type: fftypes.NamespaceTypeLocal,
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	mdi.On("DeleteNamespace", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil)
	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionConfirm, action)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastDuplicateOverrideLocalFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
		Type: fftypes.NamespaceTypeLocal,
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	mdi.On("DeleteNamespace", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastDupCheckFail(t *testing.T) {
	sh := newTestSystemHandlers(t)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := sh.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, fmt.Errorf("pop"))
	action, err := sh.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.Equal(t, ActionRetry, action)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}
