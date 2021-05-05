// Copyright © 2021 Kaleido, Inc.
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

package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/broadcastmocks"
	"github.com/kaleido-io/firefly/mocks/persistencemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastSchemaDefinitionBadType(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.BroadcastSchemaDefinition(context.Background(), &fftypes.Schema{
		Type: fftypes.SchemaType("wrong"),
	})
	assert.Regexp(t, "FF10132.*type", err.Error())
}

func TestBroadcastSchemaBadNamespace(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.BroadcastSchemaDefinition(context.Background(), &fftypes.Schema{})
	assert.Regexp(t, "FF10131.*namespace", err.Error())
}

func TestBroadcastSchemaBadEntity(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.BroadcastSchemaDefinition(context.Background(), &fftypes.Schema{
		Namespace: "ns1",
	})
	assert.Regexp(t, "FF10131.*entity", err.Error())
}

func TestBroadcastSchemaBadVersion(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.BroadcastSchemaDefinition(context.Background(), &fftypes.Schema{
		Namespace: "ns1",
		Entity:    "ent1",
	})
	assert.Regexp(t, "FF10131.*version", err.Error())
}

func TestBroadcastSchemaMissingValue(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.BroadcastSchemaDefinition(context.Background(), &fftypes.Schema{
		Namespace: "ns1",
		Entity:    "ent1",
		Version:   "0.0.1",
	})
	assert.Regexp(t, "FF10140.*value", err.Error())
}

func TestBroadcastSchemaBadValue(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.BroadcastSchemaDefinition(context.Background(), &fftypes.Schema{
		Namespace: "ns1",
		Entity:    "ent1",
		Version:   "0.0.1",
		Value: fftypes.JSONData{
			"json": map[bool]string{false: "unparsable"},
		},
	})
	assert.Regexp(t, "FF10125.*value", err.Error())
}

func TestBroadcastUpsertFail(t *testing.T) {
	e := NewEngine().(*engine)
	mp := &persistencemocks.Plugin{}
	e.persistence = mp

	mp.On("UpsertData", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := e.BroadcastSchemaDefinition(context.Background(), &fftypes.Schema{
		Namespace: "ns1",
		Entity:    "ent1",
		Version:   "0.0.1",
		Value: fftypes.JSONData{
			"some": "data",
		},
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastBroadcastFail(t *testing.T) {
	e := NewEngine().(*engine)
	e.nodeIdentity = "0x12345"
	mp := &persistencemocks.Plugin{}
	mb := &broadcastmocks.Broadcast{}
	e.persistence = mp
	e.broadcast = mb

	mp.On("UpsertData", mock.Anything, mock.Anything).Return(nil)
	mb.On("BroadcastMessage", mock.Anything, "0x12345", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := e.BroadcastSchemaDefinition(context.Background(), &fftypes.Schema{
		Namespace: "ns1",
		Entity:    "ent1",
		Version:   "0.0.1",
		Value: fftypes.JSONData{
			"some": "data",
		},
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastOk(t *testing.T) {
	e := NewEngine().(*engine)
	e.nodeIdentity = "0x12345"
	mp := &persistencemocks.Plugin{}
	mb := &broadcastmocks.Broadcast{}
	e.persistence = mp
	e.broadcast = mb

	mp.On("UpsertData", mock.Anything, mock.Anything).Return(nil)
	mb.On("BroadcastMessage", mock.Anything, "0x12345", mock.Anything, mock.Anything).Return(nil)

	_, err := e.BroadcastSchemaDefinition(context.Background(), &fftypes.Schema{
		Namespace: "ns1",
		Entity:    "ent1",
		Version:   "0.0.1",
		Value: fftypes.JSONData{
			"some": "data",
		},
	})
	assert.NoError(t, err)
}
