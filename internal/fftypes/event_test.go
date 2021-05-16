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

package fftypes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventJSONSerialization(t *testing.T) {

	var utStruct struct {
		EventTypes     EventTypes `json:"eventTypes"`
		EventTypesOmit EventTypes `json:"eventTypesOmit,omitempty"`
	}

	err := json.Unmarshal([]byte(`
	{}
	`), &utStruct)
	assert.NoError(t, err)
	assert.Nil(t, utStruct.EventTypes)
	assert.Nil(t, utStruct.EventTypesOmit)

	err = json.Unmarshal([]byte(`
	{
		"eventTypes": [],
		"eventTypesOmit": ""
	}
	`), &utStruct)
	assert.NoError(t, err)
	assert.Empty(t, utStruct.EventTypes)
	assert.Empty(t, utStruct.EventTypesOmit)

	err = json.Unmarshal([]byte(`
	{
		"eventTypes": ["message_confirmed"],
		"eventTypesOmit": "message_confirmed"
	}
	`), &utStruct)
	assert.NoError(t, err)
	assert.Equal(t, EventTypes{EventTypeMessageConfirmed}, utStruct.EventTypes)
	assert.Equal(t, EventTypes{EventTypeMessageConfirmed}, utStruct.EventTypesOmit)

	err = json.Unmarshal([]byte(`
	{
		"eventTypes": false
	}
	`), &utStruct)
	assert.Regexp(t, "FF10168", err.Error())

	err = json.Unmarshal([]byte(`
	{
		"eventTypes": ["wrong"]
	}
	`), &utStruct)
	assert.Regexp(t, "FF10169.*wrong", err.Error())
}

func TestEventDatabaseDeserialization(t *testing.T) {

	var eventTypes EventTypes

	err := eventTypes.Scan(string(EventTypeMessageConfirmed))
	assert.NoError(t, err)
	assert.Equal(t, EventTypes{EventTypeMessageConfirmed}, eventTypes)

	err = eventTypes.Scan([]byte(EventTypeMessageConfirmed))
	assert.NoError(t, err)
	assert.Equal(t, EventTypes{EventTypeMessageConfirmed}, eventTypes)

	err = eventTypes.Scan(false)
	assert.Regexp(t, "FF10125", err)

}

func TestEventDatabaseSerialization(t *testing.T) {

	eventTypes := EventTypes{EventTypeMessageConfirmed}

	v, err := eventTypes.Value()
	assert.NoError(t, err)
	assert.Equal(t, string(EventTypeMessageConfirmed), v)

	v = eventTypes.String()
	assert.Equal(t, string(EventTypeMessageConfirmed), v)

	eventTypes = nil
	v, err = eventTypes.Value()
	assert.NoError(t, err)
	assert.Equal(t, "", v)

}
