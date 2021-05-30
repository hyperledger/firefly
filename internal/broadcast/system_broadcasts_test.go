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
	"github.com/kaleido-io/firefly/mocks/datamocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleSystemBroadcastUnknown(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: "uknown",
		},
	}, []*fftypes.Data{})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestGetSystemBroadcastPayloadMissingData(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)
	valid := bm.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: "uknown",
		},
	}, nil, []*fftypes.Data{})
	assert.False(t, valid)
}

func TestGetSystemBroadcastPayloadBadJSON(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)
	valid := bm.getSystemBroadcastPayload(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: "uknown",
		},
	}, nil, []*fftypes.Data{})
	assert.False(t, valid)
}

func TestHandleSystemBroadcastDatatypeOk(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "name1",
		Version:   "ver1",
		Value:     fftypes.Byteable(`{}`),
	}
	dt.Hash = dt.Value.Hash()
	b, err := json.Marshal(&dt)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdm := bm.data.(*datamocks.Manager)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	mbi := bm.database.(*databasemocks.Plugin)
	mbi.On("GetDatatypeByName", mock.Anything, "ns1", "name1", "ver1").Return(nil, nil)
	mbi.On("UpsertDatatype", mock.Anything, mock.Anything, false).Return(nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Topic:     fftypes.SystemTopicBroadcastDatatype,
		},
	}, []*fftypes.Data{data})
	assert.True(t, valid)
	assert.NoError(t, err)

	mdm.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestHandleSystemBroadcastDatatypeMissingID(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	dt := &fftypes.Datatype{
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "name1",
		Version:   "ver1",
		Value:     fftypes.Byteable(`{}`),
	}
	dt.Hash = dt.Value.Hash()
	b, err := json.Marshal(&dt)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: fftypes.SystemTopicBroadcastDatatype,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastBadSchema(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "name1",
		Version:   "ver1",
		Value:     fftypes.Byteable(`{}`),
	}
	dt.Hash = dt.Value.Hash()
	b, err := json.Marshal(&dt)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdm := bm.data.(*datamocks.Manager)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Topic:     fftypes.SystemTopicBroadcastDatatype,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mdm.AssertExpectations(t)
}

func TestHandleSystemBroadcastMissingData(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "name1",
		Version:   "ver1",
		Value:     fftypes.Byteable(`{}`),
	}
	dt.Hash = dt.Value.Hash()

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: fftypes.SystemTopicBroadcastDatatype,
		},
	}, []*fftypes.Data{})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastDatatypeLookupFail(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "name1",
		Version:   "ver1",
		Value:     fftypes.Byteable(`{}`),
	}
	dt.Hash = dt.Value.Hash()
	b, err := json.Marshal(&dt)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdm := bm.data.(*datamocks.Manager)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	mbi := bm.database.(*databasemocks.Plugin)
	mbi.On("GetDatatypeByName", mock.Anything, "ns1", "name1", "ver1").Return(nil, fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Topic:     fftypes.SystemTopicBroadcastDatatype,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestHandleSystemBroadcastUpsertFail(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "name1",
		Version:   "ver1",
		Value:     fftypes.Byteable(`{}`),
	}
	dt.Hash = dt.Value.Hash()
	b, err := json.Marshal(&dt)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdm := bm.data.(*datamocks.Manager)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	mbi := bm.database.(*databasemocks.Plugin)
	mbi.On("GetDatatypeByName", mock.Anything, "ns1", "name1", "ver1").Return(nil, nil)
	mbi.On("UpsertDatatype", mock.Anything, mock.Anything, false).Return(fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Topic:     fftypes.SystemTopicBroadcastDatatype,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mdm.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestHandleSystemBroadcastDatatypeDuplicate(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "name1",
		Version:   "ver1",
		Value:     fftypes.Byteable(`{}`),
	}
	dt.Hash = dt.Value.Hash()
	b, err := json.Marshal(&dt)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mdm := bm.data.(*datamocks.Manager)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	mbi := bm.database.(*databasemocks.Plugin)
	mbi.On("GetDatatypeByName", mock.Anything, "ns1", "name1", "ver1").Return(dt, nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: "ns1",
			Topic:     fftypes.SystemTopicBroadcastDatatype,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mdm.AssertExpectations(t)
	mbi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNSOk(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mbi := bm.database.(*databasemocks.Plugin)
	mbi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mbi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: fftypes.SystemTopicBroadcastNamespace,
		},
	}, []*fftypes.Data{data})
	assert.True(t, valid)
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNSUpsertFail(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mbi := bm.database.(*databasemocks.Plugin)
	mbi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	mbi.On("UpsertNamespace", mock.Anything, mock.Anything, true).Return(fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: fftypes.SystemTopicBroadcastNamespace,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
}

func TestHandleSystemBroadcastNSMissingData(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: fftypes.SystemTopicBroadcastNamespace,
		},
	}, []*fftypes.Data{})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastNSBadID(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	ns := &fftypes.Namespace{}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: fftypes.SystemTopicBroadcastNamespace,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastNSBadData(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(`!{json`),
	}

	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: fftypes.SystemTopicBroadcastNamespace,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHandleSystemBroadcastDuplicate(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mbi := bm.database.(*databasemocks.Plugin)
	mbi.On("GetNamespace", mock.Anything, "ns1").Return(ns, nil)
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: fftypes.SystemTopicBroadcastNamespace,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.NoError(t, err)

	mbi.AssertExpectations(t)
}

func TestHandleSystemBroadcastDupCheckFail(t *testing.T) {
	bm, err := newTestBroadcast(context.Background())
	assert.NoError(t, err)

	ns := &fftypes.Namespace{
		ID:   fftypes.NewUUID(),
		Name: "ns1",
	}
	b, err := json.Marshal(&ns)
	assert.NoError(t, err)
	data := &fftypes.Data{
		Value: fftypes.Byteable(b),
	}

	mbi := bm.database.(*databasemocks.Plugin)
	mbi.On("GetNamespace", mock.Anything, "ns1").Return(nil, fmt.Errorf("pop"))
	valid, err := bm.HandleSystemBroadcast(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{
			Topic: fftypes.SystemTopicBroadcastNamespace,
		},
	}, []*fftypes.Data{data})
	assert.False(t, valid)
	assert.EqualError(t, err, "pop")

	mbi.AssertExpectations(t)
}
