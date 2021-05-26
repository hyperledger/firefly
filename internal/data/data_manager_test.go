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
	assert.Regexp(t, "FF10198", err.Error())

	data.Value = fftypes.Byteable(`{"field1":"value1"}`)
	data.Seal(context.Background())
	err = v.Validate(context.Background(), data)
	assert.NoError(t, err)

}

func TestInitBadDeps(t *testing.T) {
	_, err := NewDataManager(context.Background(), nil)
	assert.Regexp(t, "FF10128", err.Error())
}

func TestValidatorForWrongType(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)

	_, err := dm.GetValidator(context.Background(), &fftypes.Data{
		Validator: fftypes.ValidatorType("wrong"),
	})
	assert.Regexp(t, "FF10200.*wrong", err.Error())

}

func TestValidatorForMissingName(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	dm := newTestDataManager(t, mdi)

	_, err := dm.GetValidator(context.Background(), &fftypes.Data{
		Validator: fftypes.ValidatorTypeJSON,
	})
	assert.Regexp(t, "FF10195.*null", err.Error())

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
	assert.Regexp(t, "FF10195", err.Error())

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
	assert.Regexp(t, "pop", err.Error())

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
	assert.Regexp(t, "FF10201", err.Error())

}
