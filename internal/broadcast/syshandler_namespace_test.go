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
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleSystemBroadcastNSOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpsertEvent", mock.Anything, mock.Anything, false).Return(nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.True(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNSEventFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpsertEvent", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNSUpsertFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNSMissingData(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastNSBadID(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	ns := &fftypes.Namespace{}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastNSBadData(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	data := &fftypes.Data{
		Value: fftypes.Byteable(`!{json`),
	}

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastDuplicate(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastDuplicateOverrideLocal(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	mdi.On("DeleteNamespace", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertNamespace", mock.Anything, mock.Anything, false).Return(nil)
	mdi.On("UpsertEvent", mock.Anything, mock.Anything, false).Return(nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.True(t, valid)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastDuplicateOverrideLocalFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

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

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	mdi.On("DeleteNamespace", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestHandleSystemBroadcastDupCheckFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdi := bm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Tag: string(fftypes.SystemTagDefineNamespace),
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}
