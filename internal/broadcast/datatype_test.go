// Copyright © 2022 Kaleido, Inc.
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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastDatatypeBadType(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Validator: fftypes.ValidatorType("wrong"),
	}, false)
	assert.Regexp(t, "FF10132.*validator", err)
}

func TestBroadcastDatatypeNSGetFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(fmt.Errorf("pop"))
	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Name:      "name1",
		Namespace: "ns1",
		Version:   "0.0.1",
		Value:     fftypes.JSONAnyPtr(`{}`),
	}, false)
	assert.EqualError(t, err, "pop")
}

func TestBroadcastDatatypeBadValue(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	mim := bm.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningIdentity", mock.Anything, "ns1", mock.Anything).Return(nil)
	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.JSONAnyPtr(`!unparsable`),
	}, false)
	assert.Regexp(t, "FF10137.*value", err)
}

func TestBroadcastUpsertFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	mim.On("ResolveInputSigningIdentity", mock.Anything, "ns1", mock.Anything).Return(nil)
	mdm.On("WriteNewMessage", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)

	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.JSONAnyPtr(`{"some": "data"}`),
	}, false)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestBroadcastDatatypeInvalid(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	mim.On("ResolveInputIdentity", mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpsertData", mock.Anything, mock.Anything, database.UpsertOptimizationNew).Return(nil)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(fmt.Errorf("pop"))

	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.JSONAnyPtr(`{"some": "data"}`),
	}, false)
	assert.EqualError(t, err, "pop")
}

func TestBroadcastOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mim := bm.identity.(*identitymanagermocks.Manager)

	mim.On("ResolveInputSigningIdentity", mock.Anything, "ns1", mock.Anything).Return(nil)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	mdm.On("WriteNewMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.JSONAnyPtr(`{"some": "data"}`),
	}, false)
	assert.NoError(t, err)

	mdm.AssertExpectations(t)
	mim.AssertExpectations(t)
}
