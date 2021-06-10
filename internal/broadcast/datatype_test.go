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
	"fmt"
	"testing"

	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/mocks/datamocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastDatatypeBadType(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Validator: fftypes.ValidatorType("wrong"),
	})
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
		Value:     fftypes.Byteable(`{}`),
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastDatatypeBadValue(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdm := bm.data.(*datamocks.Manager)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`!unparsable`),
	})
	assert.Regexp(t, "FF10137.*value", err)
}

func TestBroadcastUpsertFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)

	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(fmt.Errorf("pop"))
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)

	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`{"some": "data"}`),
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastDatatypeInvalid(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)

	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(nil)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(fmt.Errorf("pop"))

	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`{"some": "data"}`),
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastBroadcastFail(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)

	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(nil)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	mdi.On("InsertMessageLocal", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`{"some": "data"}`),
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastOk(t *testing.T) {
	bm, cancel := newTestBroadcast(t)
	defer cancel()
	mdi := bm.database.(*databasemocks.Plugin)
	mdm := bm.data.(*datamocks.Manager)

	mdi.On("UpsertData", mock.Anything, mock.Anything, true, false).Return(nil)
	mdm.On("VerifyNamespaceExists", mock.Anything, "ns1").Return(nil)
	mdm.On("CheckDatatype", mock.Anything, "ns1", mock.Anything).Return(nil)
	mdi.On("InsertMessageLocal", mock.Anything, mock.Anything).Return(nil)

	_, err := bm.BroadcastDatatype(context.Background(), "ns1", &fftypes.Datatype{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value:     fftypes.Byteable(`{"some": "data"}`),
	})
	assert.NoError(t, err)
}
