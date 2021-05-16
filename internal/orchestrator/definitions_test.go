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

package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/mocks/broadcastmocks"
	"github.com/kaleido-io/firefly/mocks/databasemocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBroadcastDataDefinitionBadType(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.BroadcastDataDefinition(context.Background(), "ns1", &fftypes.DataDefinition{
		Validator: fftypes.ValidatorType("wrong"),
	})
	assert.Regexp(t, "FF10132.*validator", err.Error())
}

func TestBroadcastDataDefinitionBadNamespace(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.BroadcastDataDefinition(context.Background(), "_ns1", &fftypes.DataDefinition{})
	assert.Regexp(t, "FF10131.*namespace", err.Error())
}

func TestBroadcastDataDefinitionBadEntity(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.BroadcastDataDefinition(context.Background(), "ns1", &fftypes.DataDefinition{
		Namespace: "ns1",
	})
	assert.Regexp(t, "FF10131.*name", err.Error())
}

func TestBroadcastDataDefinitionBadVersion(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.BroadcastDataDefinition(context.Background(), "ns1", &fftypes.DataDefinition{
		Namespace: "ns1",
		Name:      "ent1",
	})
	assert.Regexp(t, "FF10131.*version", err.Error())
}

func TestBroadcastDataDefinitionMissingValue(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.BroadcastDataDefinition(context.Background(), "ns1", &fftypes.DataDefinition{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
	})
	assert.Regexp(t, "FF10140.*value", err.Error())
}

func TestBroadcastDataDefinitionBadValue(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	_, err := o.BroadcastDataDefinition(context.Background(), "ns1", &fftypes.DataDefinition{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value: fftypes.JSONObject{
			"json": map[bool]string{false: "unparsable"},
		},
	})
	assert.Regexp(t, "FF10151.*value", err.Error())
}

func TestBroadcastUpsertFail(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	mp := &databasemocks.Plugin{}
	o.database = mp

	mp.On("UpsertData", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := o.BroadcastDataDefinition(context.Background(), "ns1", &fftypes.DataDefinition{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value: fftypes.JSONObject{
			"some": "data",
		},
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastBroadcastFail(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	o.nodeIdentity = "0x12345"
	mp := &databasemocks.Plugin{}
	mb := &broadcastmocks.BroadcastManager{}
	o.database = mp
	o.broadcast = mb

	mp.On("UpsertData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mb.On("BroadcastMessage", mock.Anything, mock.Anything).Return(fmt.Errorf("pop"))

	_, err := o.BroadcastDataDefinition(context.Background(), "ns1", &fftypes.DataDefinition{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value: fftypes.JSONObject{
			"some": "data",
		},
	})
	assert.EqualError(t, err, "pop")
}

func TestBroadcastOk(t *testing.T) {
	o := NewOrchestrator().(*orchestrator)
	o.nodeIdentity = "0x12345"
	mp := &databasemocks.Plugin{}
	mb := &broadcastmocks.BroadcastManager{}
	o.database = mp
	o.broadcast = mb

	mp.On("UpsertData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mb.On("BroadcastMessage", mock.Anything, mock.Anything).Return(nil)

	_, err := o.BroadcastDataDefinition(context.Background(), "ns1", &fftypes.DataDefinition{
		Namespace: "ns1",
		Name:      "ent1",
		Version:   "0.0.1",
		Value: fftypes.JSONObject{
			"some": "data",
		},
	})
	assert.NoError(t, err)
}
