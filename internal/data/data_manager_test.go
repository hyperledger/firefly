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

package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestDataManager(t *testing.T, mdi *databasemocks.Plugin) *dataManager {
	dm, err := NewDataManager(context.Background(), mdi)
	assert.NoError(t, err)
	return dm.(*dataManager)
}

func TestValidateE2E(t *testing.T) {

	config.Reset()
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	data := &fftypes.Data{
		Namespace: "ns1",
		Validator: fftypes.ValidatorTypeJSON,
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
		Value: fftypes.Byteable(`{"some":"json"}`),
	}
	data.Seal(context.Background())
	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Value: fftypes.Byteable(`{
			"properties": {
				"field1": {
					"type": "string"
				}
			},
			"additionalProperties": false
		}`),
		Name:      "customer",
		Namespace: "0.0.1",
	}
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(dt, nil)
	v, err := dm.GetValidator(context.Background(), data)
	assert.NoError(t, err)

	err = v.Validate(context.Background(), data)
	assert.Regexp(t, "FF10198", err)

	data.Value = fftypes.Byteable(`{"field1":"value1"}`)
	data.Seal(context.Background())
	err = v.Validate(context.Background(), data)
	assert.NoError(t, err)

}

func TestInitBadDeps(t *testing.T) {
	_, err := NewDataManager(context.Background(), nil)
	assert.Regexp(t, "FF10128", err)
}

func TestValidatorForWrongType(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)

	_, err := dm.GetValidator(context.Background(), &fftypes.Data{
		Validator: fftypes.ValidatorType("wrong"),
	})
	assert.Regexp(t, "FF10200.*wrong", err)

}

func TestValidatorForMissingName(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)

	_, err := dm.GetValidator(context.Background(), &fftypes.Data{
		Validator: fftypes.ValidatorTypeJSON,
	})
	assert.Regexp(t, "FF10195.*null", err)

}

func TestValidatorUnknown(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(nil, nil)
	_, err := dm.GetValidator(context.Background(), &fftypes.Data{
		Namespace: "ns1",
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
	})
	assert.Regexp(t, "FF10195", err)

}

func TestValidatorLookupError(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(nil, fmt.Errorf("pop"))
	data := &fftypes.Data{
		Namespace: "ns1",
		Validator: fftypes.ValidatorTypeJSON,
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
		Value: fftypes.Byteable(`anything`),
	}
	data.Seal(context.Background())
	_, err := dm.GetValidator(context.Background(), data)
	assert.Regexp(t, "pop", err)

}

func TestValidatorLookupCached(t *testing.T) {

	config.Reset()
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	data := &fftypes.Data{
		Namespace: "ns1",
		Validator: fftypes.ValidatorTypeJSON,
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
	}
	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Value:     fftypes.Byteable(`{}`),
		Name:      "customer",
		Namespace: "0.0.1",
	}
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(dt, nil).Once()
	lookup1, err := dm.GetValidator(context.Background(), data)
	assert.NoError(t, err)
	assert.Equal(t, "customer", lookup1.(*jsonValidator).datatype.Name)

	lookup2, err := dm.GetValidator(context.Background(), data)
	assert.Equal(t, lookup1, lookup2)

}

func TestValidateBadHash(t *testing.T) {

	config.Reset()
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	data := &fftypes.Data{
		Namespace: "ns1",
		Validator: fftypes.ValidatorTypeJSON,
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
		Value: fftypes.Byteable(`{}`),
		Hash:  fftypes.NewRandB32(),
	}
	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Value:     fftypes.Byteable(`{}`),
		Name:      "customer",
		Namespace: "0.0.1",
	}
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(dt, nil).Once()
	v, err := dm.GetValidator(context.Background(), data)
	assert.NoError(t, err)
	err = v.Validate(context.Background(), data)
	assert.Regexp(t, "FF10201", err)

}

func TestGetMessageDataDBError(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(nil, fmt.Errorf("pop"))
	data, foundAll, err := dm.GetMessageData(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()}},
	}, true)
	assert.Nil(t, data)
	assert.False(t, foundAll)
	assert.EqualError(t, err, "pop")

}

func TestGetMessageDataNilEntry(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(nil, nil)
	data, foundAll, err := dm.GetMessageData(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{nil},
	}, true)
	assert.Empty(t, data)
	assert.False(t, foundAll)
	assert.NoError(t, err)

}

func TestGetMessageDataNotFound(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(nil, nil)
	data, foundAll, err := dm.GetMessageData(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()}},
	}, true)
	assert.Empty(t, data)
	assert.False(t, foundAll)
	assert.NoError(t, err)

}

func TestGetMessageDataHashMismatch(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	dataID := fftypes.NewUUID()
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(&fftypes.Data{
		ID:   dataID,
		Hash: fftypes.NewRandB32(),
	}, nil)
	data, foundAll, err := dm.GetMessageData(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: dataID, Hash: fftypes.NewRandB32()}},
	}, true)
	assert.Empty(t, data)
	assert.False(t, foundAll)
	assert.NoError(t, err)

}

func TestGetMessageDataOk(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	dataID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(&fftypes.Data{
		ID:   dataID,
		Hash: hash,
	}, nil)
	data, foundAll, err := dm.GetMessageData(context.Background(), &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: dataID, Hash: hash}},
	}, true)
	assert.NotEmpty(t, data)
	assert.Equal(t, *dataID, *data[0].ID)
	assert.True(t, foundAll)
	assert.NoError(t, err)

}

func TestCheckDatatypeVerifiesTheSchema(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	err := dm.CheckDatatype(context.Background(), "ns1", &fftypes.Datatype{})
	assert.Regexp(t, "FF10196", err)
}

func TestResolveInputDataEmpty(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	refs, err := dm.ResolveInputData(context.Background(), "ns1", fftypes.InputData{})
	assert.NoError(t, err)
	assert.Empty(t, refs)

}

func TestResolveInputDataRefIDOnlyOK(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)

	ctx := context.Background()
	dataID := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()

	mdi.On("GetDataByID", ctx, dataID, false).Return(&fftypes.Data{
		ID:        dataID,
		Namespace: "ns1",
		Hash:      dataHash,
	}, nil)

	refs, err := dm.ResolveInputData(ctx, "ns1", fftypes.InputData{
		{DataRef: fftypes.DataRef{ID: dataID}},
	})
	assert.NoError(t, err)
	assert.Len(t, refs, 1)
	assert.Equal(t, dataID, refs[0].ID)
	assert.Equal(t, dataHash, refs[0].Hash)
}

func TestResolveInputDataRefBadNamespace(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)

	ctx := context.Background()
	dataID := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()

	mdi.On("GetDataByID", ctx, dataID, false).Return(&fftypes.Data{
		ID:        dataID,
		Namespace: "ns2",
		Hash:      dataHash,
	}, nil)

	refs, err := dm.ResolveInputData(ctx, "ns1", fftypes.InputData{
		{DataRef: fftypes.DataRef{ID: dataID, Hash: dataHash}},
	})
	assert.Regexp(t, "FF10204", err)
	assert.Empty(t, refs)
}

func TestResolveInputDataRefBadHash(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)

	ctx := context.Background()
	dataID := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()

	mdi.On("GetDataByID", ctx, dataID, false).Return(&fftypes.Data{
		ID:        dataID,
		Namespace: "ns2",
		Hash:      dataHash,
	}, nil)

	refs, err := dm.ResolveInputData(ctx, "ns1", fftypes.InputData{
		{DataRef: fftypes.DataRef{ID: dataID, Hash: fftypes.NewRandB32()}},
	})
	assert.Regexp(t, "FF10204", err)
	assert.Empty(t, refs)
}

func TestResolveInputDataRefLookkupFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)

	ctx := context.Background()
	dataID := fftypes.NewUUID()

	mdi.On("GetDataByID", ctx, dataID, false).Return(nil, fmt.Errorf("pop"))

	_, err := dm.ResolveInputData(ctx, "ns1", fftypes.InputData{
		{DataRef: fftypes.DataRef{ID: dataID, Hash: fftypes.NewRandB32()}},
	})
	assert.EqualError(t, err, "pop")
}

func TestResolveInputDataValueNoValidatorOK(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	ctx := context.Background()

	mdi.On("UpsertData", ctx, mock.Anything, false, false).Return(nil)

	refs, err := dm.ResolveInputData(ctx, "ns1", fftypes.InputData{
		{Value: fftypes.Byteable(`{"some":"json"}`)},
	})
	assert.NoError(t, err)
	assert.Len(t, refs, 1)
	assert.NotNil(t, refs[0].ID)
	assert.NotNil(t, refs[0].Hash)
}

func TestResolveInputDataValueNoValidatorStoreFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	ctx := context.Background()

	mdi.On("UpsertData", ctx, mock.Anything, false, false).Return(fmt.Errorf("pop"))

	_, err := dm.ResolveInputData(ctx, "ns1", fftypes.InputData{
		{Value: fftypes.Byteable(`{"some":"json"}`)},
	})
	assert.EqualError(t, err, "pop")
}

func TestResolveInputDataValueWithValidation(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	ctx := context.Background()

	mdi.On("UpsertData", ctx, mock.Anything, false, false).Return(nil)
	mdi.On("GetDatatypeByName", ctx, "ns1", "customer", "0.0.1").Return(&fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "customer",
		Version:   "0.0.1",
		Value: fftypes.Byteable(`{
			"properties": {
				"field1": {
					"type": "string"
				}
			},
			"additionalProperties": false
		}`),
	}, nil)

	refs, err := dm.ResolveInputData(ctx, "ns1", fftypes.InputData{
		{
			Datatype: &fftypes.DatatypeRef{
				Name:    "customer",
				Version: "0.0.1",
			},
			Value: fftypes.Byteable(`{"field1":"value1"}`),
		},
	})
	assert.NoError(t, err)
	assert.Len(t, refs, 1)
	assert.NotNil(t, refs[0].ID)
	assert.NotNil(t, refs[0].Hash)

	_, err = dm.ResolveInputData(ctx, "ns1", fftypes.InputData{
		{
			Datatype: &fftypes.DatatypeRef{
				Name:    "customer",
				Version: "0.0.1",
			},
			Value: fftypes.Byteable(`{"not_allowed":"value"}`),
		},
	})
	assert.Regexp(t, "FF10198", err)
}

func TestResolveInputDataNoRefOrValue(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	ctx := context.Background()

	_, err := dm.ResolveInputData(ctx, "ns1", fftypes.InputData{
		{ /* missing */ },
	})
	assert.Regexp(t, "FF10205", err)
}

func TestValidateAndStoreLoadDatatypeFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)
	ctx := context.Background()

	mdi.On("GetDatatypeByName", ctx, "ns1", "customer", "0.0.1").Return(nil, fmt.Errorf("pop"))
	_, err := dm.validateAndStore(ctx, "ns1", &fftypes.DataRefOrValue{
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
	})
	assert.EqualError(t, err, "pop")
}
