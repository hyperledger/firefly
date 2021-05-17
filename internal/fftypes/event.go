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
	"context"
	"database/sql/driver"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/i18n"
)

type EventType string

type EventTypes []EventType

const (
	EventTypeDataArrived      EventType = "data_arrived"
	EventTypeMessageBroadcast EventType = "message_broadcast"
	EventTypeMessageConfirmed EventType = "message_confirmed"
)

type Event struct {
	ID        *uuid.UUID  `json:"id"`
	Type      EventType   `json:"type"`
	Namespace string      `json:"namespace"`
	Reference *uuid.UUID  `json:"reference"`
	Sequence  int64       `json:"sequence"`
	Data      interface{} `json:"data"` // Not persisted
}

func (et *EventTypes) UnmarshalJSON(b []byte) error {
	stringArray := []string{}
	err := json.Unmarshal(b, &stringArray)
	if err != nil {
		var strValue string
		err = json.Unmarshal(b, &strValue)
		if err != nil {
			return i18n.WrapError(context.Background(), err, i18n.MsgEventTypesParseFail)
		}
		if len(strValue) > 0 {
			stringArray = strings.Split(strValue, ",")
		}
	}
	*et, err = stringArrayToEventTypes(stringArray)
	return err
}

func stringArrayToEventTypes(stringArray []string) (EventTypes, error) {
	eventTypes := make(EventTypes, len(stringArray))
	for i, eventString := range stringArray {
		switch strings.ToLower(eventString) {
		case string(EventTypeMessageConfirmed):
			eventTypes[i] = EventTypeMessageConfirmed
		default:
			return nil, i18n.NewError(context.Background(), i18n.MsgUnknownEventType, eventString)
		}
	}
	return eventTypes, nil
}

// Scan implements sql.Scanner
func (et *EventTypes) Scan(src interface{}) (err error) {
	switch src := src.(type) {
	case []byte:
		var stringArray []string
		if len(src) > 0 {
			stringArray = strings.Split(string(src), ",")
		}
		*et, err = stringArrayToEventTypes(stringArray)
		return err
	case string:
		var stringArray []string
		if len(src) > 0 {
			stringArray = strings.Split(src, ",")
		}
		*et, err = stringArrayToEventTypes(stringArray)
		return err
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, et)
	}

}

// Value implements sql.Valuer
func (et EventTypes) Value() (driver.Value, error) {
	stringArray := make([]string, len(et))
	for i, eventType := range et {
		stringArray[i] = string(eventType)
	}
	return strings.Join(stringArray, ","), nil
}

func (et *EventTypes) String() string {
	s, _ := et.Value()
	return s.(string)
}
