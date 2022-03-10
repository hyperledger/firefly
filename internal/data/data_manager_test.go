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

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/dataexchangemocks"
	"github.com/hyperledger/firefly/mocks/sharedstoragemocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestDataManager(t *testing.T) (*dataManager, context.Context, func()) {
	config.Reset()
	ctx, cancel := context.WithCancel(context.Background())
	mdi := &databasemocks.Plugin{}
	mdi.On("Capabilities").Return(&database.Capabilities{
		Concurrency: true,
	})
	mdx := &dataexchangemocks.Plugin{}
	mps := &sharedstoragemocks.Plugin{}
	dm, err := NewDataManager(ctx, mdi, mps, mdx)
	assert.NoError(t, err)
	return dm.(*dataManager), ctx, func() {
		cancel()
		dm.Close()
	}
}

func testNewMessage() (*fftypes.UUID, *fftypes.Bytes32, *NewMessage) {
	dataID := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()
	return dataID, dataHash, &NewMessage{
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:        fftypes.NewUUID(),
				Namespace: "ns1",
			},
		},
		InData: fftypes.InlineData{
			{DataRef: fftypes.DataRef{ID: dataID, Hash: dataHash}},
		},
	}
}

func TestValidateE2E(t *testing.T) {

	config.Reset()
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	data := &fftypes.Data{
		Namespace: "ns1",
		Validator: fftypes.ValidatorTypeJSON,
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
		Value: fftypes.JSONAnyPtr(`{"some":"json"}`),
	}
	data.Seal(ctx, nil)
	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Value: fftypes.JSONAnyPtr(`{
			"properties": {
				"field1": {
					"type": "string"
				}
			},
			"additionalProperties": false
		}`),
		Namespace: "ns1",
		Name:      "customer",
		Version:   "0.0.1",
	}
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(dt, nil)
	isValid, err := dm.ValidateAll(ctx, fftypes.DataArray{data})
	assert.Regexp(t, "FF10198", err)
	assert.False(t, isValid)

	v, err := dm.getValidatorForDatatype(ctx, data.Namespace, data.Validator, data.Datatype)
	err = v.Validate(ctx, data)
	assert.Regexp(t, "FF10198", err)

	data.Value = fftypes.JSONAnyPtr(`{"field1":"value1"}`)
	data.Seal(context.Background(), nil)
	err = v.Validate(ctx, data)
	assert.NoError(t, err)

	isValid, err = dm.ValidateAll(ctx, fftypes.DataArray{data})
	assert.NoError(t, err)
	assert.True(t, isValid)

}

func TestWriteNewMessageE2E(t *testing.T) {

	config.Reset()
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Value:     fftypes.JSONAnyPtr(`{}`),
		Name:      "customer",
		Namespace: "0.0.1",
	}

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(dt, nil).Once()
	mdi.On("RunAsGroup", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		err := args[1].(func(context.Context) error)(ctx)
		assert.NoError(t, err)
	}).Return(nil)
	mdi.On("InsertDataArray", mock.Anything, mock.Anything).Return(nil).Once()

	data1, err := dm.UploadJSON(ctx, "ns1", &fftypes.DataRefOrValue{
		Value:     fftypes.JSONAnyPtr(`"message 1 - data A"`),
		Validator: fftypes.ValidatorTypeJSON,
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
	})
	assert.NoError(t, err)

	mdi.On("GetDataByID", mock.Anything, data1.ID, true).Return(data1, nil).Once()

	_, _, newMsg1 := testNewMessage()
	newMsg1.InData = fftypes.InlineData{
		{DataRef: fftypes.DataRef{
			ID:   data1.ID,
			Hash: data1.Hash,
		}},
		{Value: fftypes.JSONAnyPtr(`"message 1 - data B"`)},
		{Value: fftypes.JSONAnyPtr(`"message 1 - data C"`)},
	}
	_, _, newMsg2 := testNewMessage()
	newMsg2.InData = fftypes.InlineData{
		{Value: fftypes.JSONAnyPtr(`"message 2 - data B"`)},
		{Value: fftypes.JSONAnyPtr(`"message 2 - data C"`)},
	}

	err = dm.ResolveInlineDataPrivate(ctx, newMsg1)
	assert.NoError(t, err)
	err = dm.ResolveInlineDataPrivate(ctx, newMsg2)
	assert.NoError(t, err)

	allData := append(append(fftypes.DataArray{}, newMsg1.ResolvedData.NewData...), newMsg2.ResolvedData.NewData...)
	assert.Len(t, allData, 4)

	mdi.On("InsertMessages", mock.Anything, mock.MatchedBy(func(msgs []*fftypes.Message) bool {
		msgsByID := make(map[fftypes.UUID]bool)
		for _, msg := range msgs {
			msgsByID[*msg.Header.ID] = true
		}
		return len(msgs) == 2 &&
			msgsByID[*newMsg1.Message.Header.ID] &&
			msgsByID[*newMsg2.Message.Header.ID]
	})).Return(nil).Once()
	mdi.On("InsertDataArray", mock.Anything, mock.MatchedBy(func(dataArray fftypes.DataArray) bool {
		dataByID := make(map[fftypes.UUID]bool)
		for _, data := range dataArray {
			dataByID[*data.ID] = true
		}
		return len(dataArray) == 4 &&
			dataByID[*newMsg1.ResolvedData.AllData[1].ID] &&
			dataByID[*newMsg1.ResolvedData.AllData[2].ID] &&
			dataByID[*newMsg2.ResolvedData.AllData[0].ID] &&
			dataByID[*newMsg2.ResolvedData.AllData[1].ID]
	})).Return(nil).Once()

	results := make(chan error)

	go func() {
		results <- dm.WriteNewMessage(ctx, newMsg1)
	}()
	go func() {
		results <- dm.WriteNewMessage(ctx, newMsg2)
	}()

	assert.NoError(t, <-results)
	assert.NoError(t, <-results)

	mdi.AssertExpectations(t)
}
func TestInitBadDeps(t *testing.T) {
	_, err := NewDataManager(context.Background(), nil, nil, nil)
	assert.Regexp(t, "FF10128", err)
}

func TestValidatorLookupCached(t *testing.T) {

	config.Reset()
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	ref := &fftypes.DatatypeRef{
		Name:    "customer",
		Version: "0.0.1",
	}
	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Value:     fftypes.JSONAnyPtr(`{}`),
		Name:      "customer",
		Namespace: "0.0.1",
	}
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(dt, nil).Once()
	lookup1, err := dm.getValidatorForDatatype(ctx, "ns1", fftypes.ValidatorTypeJSON, ref)
	assert.NoError(t, err)
	assert.Equal(t, "customer", lookup1.(*jsonValidator).datatype.Name)

	lookup2, err := dm.getValidatorForDatatype(ctx, "ns1", fftypes.ValidatorTypeJSON, ref)
	assert.NoError(t, err)
	assert.Equal(t, lookup1, lookup2)

}

func TestValidateBadHash(t *testing.T) {

	config.Reset()
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	data := &fftypes.Data{
		Namespace: "ns1",
		Validator: fftypes.ValidatorTypeJSON,
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
		Value: fftypes.JSONAnyPtr(`{}`),
		Hash:  fftypes.NewRandB32(),
	}
	dt := &fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Value:     fftypes.JSONAnyPtr(`{}`),
		Name:      "customer",
		Namespace: "0.0.1",
	}
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(dt, nil).Once()
	_, err := dm.ValidateAll(ctx, fftypes.DataArray{data})
	assert.Regexp(t, "FF10201", err)

}

func TestGetMessageDataDBError(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(nil, fmt.Errorf("pop"))
	data, foundAll, err := dm.GetMessageDataCached(ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()}},
	})
	assert.Nil(t, data)
	assert.False(t, foundAll)
	assert.EqualError(t, err, "pop")

}

func TestGetMessageDataNilEntry(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(nil, nil)
	data, foundAll, err := dm.GetMessageDataCached(ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{nil},
	})
	assert.Empty(t, data)
	assert.False(t, foundAll)
	assert.NoError(t, err)

}

func TestGetMessageDataNotFound(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(nil, nil)
	data, foundAll, err := dm.GetMessageDataCached(ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()}},
	})
	assert.Empty(t, data)
	assert.False(t, foundAll)
	assert.NoError(t, err)

}

func TestGetMessageDataHashMismatch(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	dataID := fftypes.NewUUID()
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(&fftypes.Data{
		ID:   dataID,
		Hash: fftypes.NewRandB32(),
	}, nil)
	data, foundAll, err := dm.GetMessageDataCached(ctx, &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: dataID, Hash: fftypes.NewRandB32()}},
	})
	assert.Empty(t, data)
	assert.False(t, foundAll)
	assert.NoError(t, err)

}

func TestGetMessageDataOk(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	dataID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: dataID, Hash: hash}},
	}

	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(&fftypes.Data{
		ID:   dataID,
		Hash: hash,
	}, nil).Once()
	data, foundAll, err := dm.GetMessageDataCached(ctx, msg)
	assert.NotEmpty(t, data)
	assert.Equal(t, *dataID, *data[0].ID)
	assert.True(t, foundAll)
	assert.NoError(t, err)

	// Check cache kicks in for second call
	data, foundAll, err = dm.GetMessageDataCached(ctx, msg)
	assert.NotEmpty(t, data)
	assert.Equal(t, *dataID, *data[0].ID)
	assert.True(t, foundAll)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestCheckDatatypeVerifiesTheSchema(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	err := dm.CheckDatatype(ctx, "ns1", &fftypes.Datatype{})
	assert.Regexp(t, "FF10196", err)
}

func TestResolveInlineDataEmpty(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	newMsg := &NewMessage{
		Message: &fftypes.Message{
			Header: fftypes.MessageHeader{
				ID:        fftypes.NewUUID(),
				Namespace: "ns1",
			},
		},
		InData: fftypes.InlineData{},
	}

	err := dm.ResolveInlineDataPrivate(ctx, newMsg)
	assert.NoError(t, err)
	assert.Empty(t, newMsg.ResolvedData.AllData)
	assert.Empty(t, newMsg.Message.Data)

}

func TestResolveInlineDataRefIDOnlyOK(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)

	dataID, dataHash, newMsg := testNewMessage()

	mdi.On("GetDataByID", ctx, dataID, true).Return(&fftypes.Data{
		ID:        dataID,
		Namespace: "ns1",
		Hash:      dataHash,
	}, nil)

	err := dm.ResolveInlineDataPrivate(ctx, newMsg)
	assert.NoError(t, err)
	assert.Len(t, newMsg.ResolvedData.AllData, 1)
	assert.Len(t, newMsg.Message.Data, 1)
	assert.Equal(t, dataID, newMsg.ResolvedData.AllData[0].ID)
	assert.Equal(t, dataHash, newMsg.ResolvedData.AllData[0].Hash)
	assert.Empty(t, newMsg.ResolvedData.NewData)
	assert.Empty(t, newMsg.ResolvedData.DataToPublish)
}

func TestResolveInlineDataBroadcastDataToPublish(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)

	dataID, dataHash, newMsg := testNewMessage()
	blobHash := fftypes.NewRandB32()

	mdi.On("GetDataByID", ctx, dataID, true).Return(&fftypes.Data{
		ID:        dataID,
		Namespace: "ns1",
		Hash:      dataHash,
		Blob: &fftypes.BlobRef{
			Hash: blobHash,
		},
	}, nil)
	mdi.On("GetBlobMatchingHash", ctx, blobHash).Return(&fftypes.Blob{
		Hash:       blobHash,
		PayloadRef: "blob/1",
	}, nil)

	err := dm.ResolveInlineDataBroadcast(ctx, newMsg)
	assert.NoError(t, err)
	assert.Len(t, newMsg.ResolvedData.AllData, 1)
	assert.Len(t, newMsg.Message.Data, 1)
	assert.Empty(t, newMsg.ResolvedData.NewData)
	assert.Len(t, newMsg.ResolvedData.DataToPublish, 1)
	assert.Equal(t, dataID, newMsg.ResolvedData.AllData[0].ID)
	assert.Equal(t, dataHash, newMsg.ResolvedData.AllData[0].Hash)
	assert.Len(t, newMsg.ResolvedData.DataToPublish, 1)
	assert.Equal(t, newMsg.ResolvedData.AllData[0].ID, newMsg.ResolvedData.DataToPublish[0].Data.ID)
	assert.Equal(t, "blob/1", newMsg.ResolvedData.DataToPublish[0].Blob.PayloadRef)
}

func TestResolveInlineDataBroadcastResolveBlobFail(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)

	dataID, dataHash, newMsg := testNewMessage()
	blobHash := fftypes.NewRandB32()

	mdi.On("GetDataByID", ctx, dataID, true).Return(&fftypes.Data{
		ID:        dataID,
		Namespace: "ns1",
		Hash:      dataHash,
		Blob: &fftypes.BlobRef{
			Hash: blobHash,
		},
	}, nil)
	mdi.On("GetBlobMatchingHash", ctx, blobHash).Return(nil, fmt.Errorf("pop"))

	err := dm.ResolveInlineDataBroadcast(ctx, newMsg)
	assert.EqualError(t, err, "pop")
}

func TestResolveInlineDataRefBadNamespace(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)

	dataID, dataHash, newMsg := testNewMessage()

	mdi.On("GetDataByID", ctx, dataID, true).Return(&fftypes.Data{
		ID:        dataID,
		Namespace: "ns2",
		Hash:      dataHash,
	}, nil)

	err := dm.ResolveInlineDataPrivate(ctx, newMsg)
	assert.Regexp(t, "FF10204", err)
}

func TestResolveInlineDataRefBadHash(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)

	dataID, dataHash, newMsg := testNewMessage()

	mdi.On("GetDataByID", ctx, dataID, true).Return(&fftypes.Data{
		ID:        dataID,
		Namespace: "ns2",
		Hash:      dataHash,
	}, nil)

	err := dm.ResolveInlineDataPrivate(ctx, newMsg)
	assert.Regexp(t, "FF10204", err)
}

func TestResolveInlineDataRefLookkupFail(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)

	dataID, _, newMsg := testNewMessage()

	mdi.On("GetDataByID", ctx, dataID, true).Return(nil, fmt.Errorf("pop"))

	err := dm.ResolveInlineDataPrivate(ctx, newMsg)
	assert.EqualError(t, err, "pop")
}

func TestResolveInlineDataValueNoValidatorOK(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)

	mdi.On("UpsertData", ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)

	_, _, newMsg := testNewMessage()
	newMsg.InData = fftypes.InlineData{
		{Value: fftypes.JSONAnyPtr(`{"some":"json"}`)},
	}

	err := dm.ResolveInlineDataPrivate(ctx, newMsg)
	assert.NoError(t, err)
	assert.Len(t, newMsg.ResolvedData.AllData, 1)
	assert.Len(t, newMsg.ResolvedData.NewData, 1)
	assert.Len(t, newMsg.Message.Data, 1)
	assert.NotNil(t, newMsg.ResolvedData.AllData[0].ID)
	assert.NotNil(t, newMsg.ResolvedData.AllData[0].Hash)
}

func TestResolveInlineDataValueWithValidation(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)

	mdi.On("UpsertData", ctx, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdi.On("GetDatatypeByName", ctx, "ns1", "customer", "0.0.1").Return(&fftypes.Datatype{
		ID:        fftypes.NewUUID(),
		Validator: fftypes.ValidatorTypeJSON,
		Namespace: "ns1",
		Name:      "customer",
		Version:   "0.0.1",
		Value: fftypes.JSONAnyPtr(`{
			"properties": {
				"field1": {
					"type": "string"
				}
			},
			"additionalProperties": false
		}`),
	}, nil)

	_, _, newMsg := testNewMessage()
	newMsg.InData = fftypes.InlineData{
		{
			Datatype: &fftypes.DatatypeRef{
				Name:    "customer",
				Version: "0.0.1",
			},
			Value: fftypes.JSONAnyPtr(`{"field1":"value1"}`),
		},
	}

	err := dm.ResolveInlineDataPrivate(ctx, newMsg)
	assert.NoError(t, err)
	assert.Len(t, newMsg.ResolvedData.AllData, 1)
	assert.Len(t, newMsg.ResolvedData.NewData, 1)
	assert.NotNil(t, newMsg.ResolvedData.AllData[0].ID)
	assert.NotNil(t, newMsg.ResolvedData.AllData[0].Hash)

	newMsg.InData = fftypes.InlineData{
		{
			Datatype: &fftypes.DatatypeRef{
				Name:    "customer",
				Version: "0.0.1",
			},
			Value: fftypes.JSONAnyPtr(`{"not_allowed":"value"}`),
		},
	}
	err = dm.ResolveInlineDataPrivate(ctx, newMsg)
	assert.Regexp(t, "FF10198", err)
}

func TestResolveInlineDataNoRefOrValue(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	_, _, newMsg := testNewMessage()
	newMsg.InData = fftypes.InlineData{
		{ /* missing */ },
	}

	err := dm.ResolveInlineDataPrivate(ctx, newMsg)
	assert.Regexp(t, "FF10205", err)
}

func TestUploadJSONLoadDatatypeFail(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)

	mdi.On("GetDatatypeByName", ctx, "ns1", "customer", "0.0.1").Return(nil, fmt.Errorf("pop"))
	_, err := dm.UploadJSON(ctx, "ns1", &fftypes.DataRefOrValue{
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
	})
	assert.EqualError(t, err, "pop")
}

func TestUploadJSONLoadInsertDataFail(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	dm.messageWriter.close()
	_, err := dm.UploadJSON(ctx, "ns1", &fftypes.DataRefOrValue{
		Value: fftypes.JSONAnyPtr(`{}`),
	})
	assert.Regexp(t, "FF10158", err)
}

func TestValidateAndStoreLoadNilRef(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	_, _, err := dm.validateInputData(ctx, "ns1", &fftypes.DataRefOrValue{
		Validator: fftypes.ValidatorTypeJSON,
		Datatype:  nil,
	})
	assert.Regexp(t, "FF10199", err)
}

func TestValidateAndStoreLoadValidatorUnknown(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(nil, nil)
	_, _, err := dm.validateInputData(ctx, "ns1", &fftypes.DataRefOrValue{
		Validator: "wrong!",
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
	})
	assert.Regexp(t, "FF10200.*wrong", err)

}

func TestValidateAndStoreLoadBadRef(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(nil, nil)
	_, _, err := dm.validateInputData(ctx, "ns1", &fftypes.DataRefOrValue{
		Datatype: &fftypes.DatatypeRef{
			// Missing name
		},
	})
	assert.Regexp(t, "FF10195", err)
}

func TestValidateAndStoreNotFound(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(nil, nil)
	_, _, err := dm.validateInputData(ctx, "ns1", &fftypes.DataRefOrValue{
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
	})
	assert.Regexp(t, "FF10195", err)
}

func TestValidateAndStoreBlobError(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	blobHash := fftypes.NewRandB32()
	mdi.On("GetBlobMatchingHash", mock.Anything, blobHash).Return(nil, fmt.Errorf("pop"))
	_, _, err := dm.validateInputData(ctx, "ns1", &fftypes.DataRefOrValue{
		Blob: &fftypes.BlobRef{
			Hash: blobHash,
		},
	})
	assert.Regexp(t, "pop", err)
}

func TestValidateAndStoreBlobNotFound(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	blobHash := fftypes.NewRandB32()
	mdi.On("GetBlobMatchingHash", mock.Anything, blobHash).Return(nil, nil)
	_, _, err := dm.validateInputData(ctx, "ns1", &fftypes.DataRefOrValue{
		Blob: &fftypes.BlobRef{
			Hash: blobHash,
		},
	})
	assert.Regexp(t, "FF10239", err)
}

func TestValidateAllLookupError(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(nil, fmt.Errorf("pop"))
	data := &fftypes.Data{
		Namespace: "ns1",
		Validator: fftypes.ValidatorTypeJSON,
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
		Value: fftypes.JSONAnyPtr(`anything`),
	}
	data.Seal(ctx, nil)
	_, err := dm.ValidateAll(ctx, fftypes.DataArray{data})
	assert.Regexp(t, "pop", err)

}

func TestGetValidatorForDatatypeNilRef(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	v, err := dm.getValidatorForDatatype(ctx, "", "", nil)
	assert.Nil(t, v)
	assert.NoError(t, err)

}

func TestValidateAllStoredValidatorInvalid(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetDatatypeByName", mock.Anything, "ns1", "customer", "0.0.1").Return(&fftypes.Datatype{
		Value: fftypes.JSONAnyPtr(`{"not": "a", "schema": true}`),
	}, nil)
	data := &fftypes.Data{
		Namespace: "ns1",
		Datatype: &fftypes.DatatypeRef{
			Name:    "customer",
			Version: "0.0.1",
		},
	}
	isValid, err := dm.ValidateAll(ctx, fftypes.DataArray{data})
	assert.False(t, isValid)
	assert.NoError(t, err)
	mdi.AssertExpectations(t)
}

func TestVerifyNamespaceExistsInvalidFFName(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	err := dm.VerifyNamespaceExists(ctx, "!wrong")
	assert.Regexp(t, "FF10131", err)
}

func TestVerifyNamespaceExistsLookupErr(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, fmt.Errorf("pop"))
	err := dm.VerifyNamespaceExists(ctx, "ns1")
	assert.Regexp(t, "pop", err)
}

func TestVerifyNamespaceExistsNotFound(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(nil, nil)
	err := dm.VerifyNamespaceExists(ctx, "ns1")
	assert.Regexp(t, "FF10187", err)
}

func TestVerifyNamespaceExistsOk(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetNamespace", mock.Anything, "ns1").Return(&fftypes.Namespace{}, nil)
	err := dm.VerifyNamespaceExists(ctx, "ns1")
	assert.NoError(t, err)
}

func TestHydrateBatchOK(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	batchID := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	msgHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()
	bp := &fftypes.BatchPersisted{
		BatchHeader: fftypes.BatchHeader{
			Type:      fftypes.BatchTypeBroadcast,
			ID:        batchID,
			Namespace: "ns1",
		},
		Manifest: fftypes.JSONAnyPtr(fmt.Sprintf(`{"id":"%s","messages":[{"id":"%s","hash":"%s"}],"data":[{"id":"%s","hash":"%s"}]}`,
			batchID, msgID, msgHash, dataID, dataHash,
		)),
		TX: fftypes.TransactionRef{
			ID: fftypes.NewUUID(),
		},
	}

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", ctx, msgID).Return(&fftypes.Message{
		Header:    fftypes.MessageHeader{ID: msgID},
		Hash:      msgHash,
		Confirmed: fftypes.Now(),
	}, nil)
	mdi.On("GetDataByID", ctx, dataID, true).Return(&fftypes.Data{
		ID:      dataID,
		Hash:    dataHash,
		Created: fftypes.Now(),
	}, nil)

	batch, err := dm.HydrateBatch(ctx, bp)
	assert.NoError(t, err)
	assert.Equal(t, bp.BatchHeader, batch.BatchHeader)
	assert.Equal(t, bp.TX, batch.Payload.TX)
	assert.Equal(t, msgID, batch.Payload.Messages[0].Header.ID)
	assert.Equal(t, msgHash, batch.Payload.Messages[0].Hash)
	assert.Nil(t, batch.Payload.Messages[0].Confirmed)
	assert.Equal(t, dataID, batch.Payload.Data[0].ID)
	assert.Equal(t, dataHash, batch.Payload.Data[0].Hash)
	assert.Equal(t, dataHash, batch.Payload.Data[0].Hash)
	assert.NotNil(t, batch.Payload.Data[0].Created)

	mdi.AssertExpectations(t)
}

func TestHydrateBatchDataFail(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	batchID := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	msgHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()
	bp := &fftypes.BatchPersisted{
		BatchHeader: fftypes.BatchHeader{
			Type:      fftypes.BatchTypeBroadcast,
			ID:        batchID,
			Namespace: "ns1",
		},
		Manifest: fftypes.JSONAnyPtr(fmt.Sprintf(`{"id":"%s","messages":[{"id":"%s","hash":"%s"}],"data":[{"id":"%s","hash":"%s"}]}`,
			batchID, msgID, msgHash, dataID, dataHash,
		)),
		TX: fftypes.TransactionRef{
			ID: fftypes.NewUUID(),
		},
	}

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", ctx, msgID).Return(&fftypes.Message{
		Header:    fftypes.MessageHeader{ID: msgID},
		Hash:      msgHash,
		Confirmed: fftypes.Now(),
	}, nil)
	mdi.On("GetDataByID", ctx, dataID, true).Return(nil, fmt.Errorf("pop"))

	_, err := dm.HydrateBatch(ctx, bp)
	assert.Regexp(t, "FF10372.*pop", err)

	mdi.AssertExpectations(t)
}

func TestHydrateBatchMsgNotFound(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	batchID := fftypes.NewUUID()
	msgID := fftypes.NewUUID()
	msgHash := fftypes.NewRandB32()
	dataID := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()
	bp := &fftypes.BatchPersisted{
		BatchHeader: fftypes.BatchHeader{
			Type:      fftypes.BatchTypeBroadcast,
			ID:        batchID,
			Namespace: "ns1",
		},
		Manifest: fftypes.JSONAnyPtr(fmt.Sprintf(`{"id":"%s","messages":[{"id":"%s","hash":"%s"}],"data":[{"id":"%s","hash":"%s"}]}`,
			batchID, msgID, msgHash, dataID, dataHash,
		)),
		TX: fftypes.TransactionRef{
			ID: fftypes.NewUUID(),
		},
	}

	mdi := dm.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", ctx, msgID).Return(nil, nil)

	_, err := dm.HydrateBatch(ctx, bp)
	assert.Regexp(t, "FF10372", err)

	mdi.AssertExpectations(t)
}

func TestHydrateBatchMsgBadManifest(t *testing.T) {
	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	bp := &fftypes.BatchPersisted{
		Manifest: fftypes.JSONAnyPtr(`!json`),
	}

	_, err := dm.HydrateBatch(ctx, bp)
	assert.Regexp(t, "FF10151", err)
}

func TestGetMessageWithDataOk(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	dataID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: dataID, Hash: hash}},
	}

	mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil).Once()
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(&fftypes.Data{
		ID:   dataID,
		Hash: hash,
	}, nil).Once()
	msgRet, data, foundAll, err := dm.GetMessageWithDataCached(ctx, msg.Header.ID)
	assert.Equal(t, msg, msgRet)
	assert.NotEmpty(t, data)
	assert.Equal(t, *dataID, *data[0].ID)
	assert.True(t, foundAll)
	assert.NoError(t, err)

	// Check cache kicks in for second call
	msgRet, data, foundAll, err = dm.GetMessageWithDataCached(ctx, msg.Header.ID)
	assert.Equal(t, msg, msgRet)
	assert.NotEmpty(t, data)
	assert.Equal(t, *dataID, *data[0].ID)
	assert.True(t, foundAll)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetMessageWithDataCRORequirePublicBlobRefs(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	dataID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: dataID, Hash: hash}},
	}

	mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil).Twice()
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(&fftypes.Data{
		ID:   dataID,
		Hash: hash,
		Blob: &fftypes.BlobRef{
			Hash: fftypes.NewRandB32(),
		},
	}, nil).Twice()
	msgRet, data, foundAll, err := dm.GetMessageWithDataCached(ctx, msg.Header.ID)
	assert.Equal(t, msg, msgRet)
	assert.NotEmpty(t, data)
	assert.Equal(t, *dataID, *data[0].ID)
	assert.True(t, foundAll)
	assert.NoError(t, err)

	// Check cache does not kick in as we have missing blob ref
	msgRet, data, foundAll, err = dm.GetMessageWithDataCached(ctx, msg.Header.ID, CRORequirePublicBlobRefs)
	assert.Equal(t, msg, msgRet)
	assert.NotEmpty(t, data)
	assert.Equal(t, *dataID, *data[0].ID)
	assert.True(t, foundAll)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestGetMessageWithDataReadDataFail(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	dataID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: dataID, Hash: hash}},
	}

	mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, nil)
	mdi.On("GetDataByID", mock.Anything, mock.Anything, true).Return(nil, fmt.Errorf("pop"))
	_, _, _, err := dm.GetMessageWithDataCached(ctx, msg.Header.ID)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestGetMessageWithDataReadMessageFail(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()
	mdi := dm.database.(*databasemocks.Plugin)
	dataID := fftypes.NewUUID()
	hash := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{ID: fftypes.NewUUID()},
		Data:   fftypes.DataRefs{{ID: dataID, Hash: hash}},
	}

	mdi.On("GetMessageByID", mock.Anything, mock.Anything).Return(msg, fmt.Errorf("pop"))
	_, _, _, err := dm.GetMessageWithDataCached(ctx, msg.Header.ID)
	assert.Regexp(t, "pop", err)

	mdi.AssertExpectations(t)
}

func TestUpdateMessageCacheCRORequirePins(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	data := fftypes.DataArray{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
	}
	msgNoPins := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
		Data: data.Refs(),
	}
	msgWithPins := &fftypes.Message{
		Header: msgNoPins.Header,
		Data:   data.Refs(),
		Pins:   fftypes.FFStringArray{"pin1"},
	}

	dm.UpdateMessageCache(msgNoPins, data)

	mce := dm.queryMessageCache(ctx, msgNoPins.Header.ID, CRORequirePins)
	assert.Nil(t, mce)

	dm.UpdateMessageIfCached(ctx, msgWithPins)
	for mce == nil {
		mce = dm.queryMessageCache(ctx, msgNoPins.Header.ID, CRORequirePins)
	}

}

func TestUpdateMessageCacheCRORequireBatchID(t *testing.T) {

	dm, ctx, cancel := newTestDataManager(t)
	defer cancel()

	data := fftypes.DataArray{
		{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
	}
	msgNoPins := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:     fftypes.NewUUID(),
			Topics: fftypes.FFStringArray{"topic1"},
		},
		Data: data.Refs(),
	}
	msgWithBatch := &fftypes.Message{
		Header:  msgNoPins.Header,
		Data:    data.Refs(),
		BatchID: fftypes.NewUUID(),
	}

	dm.UpdateMessageCache(msgNoPins, data)

	mce := dm.queryMessageCache(ctx, msgNoPins.Header.ID, CRORequireBatchID)
	assert.Nil(t, mce)

	dm.UpdateMessageIfCached(ctx, msgWithBatch)
	for mce == nil {
		mce = dm.queryMessageCache(ctx, msgNoPins.Header.ID, CRORequireBatchID)
	}

}
