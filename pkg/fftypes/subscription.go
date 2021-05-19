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

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/i18n"
)

type SubscriptionFilter struct {
	Events  string `json:"events,omitempty"`
	Topic   string `json:"topic,omitempty"`
	Context string `json:"context,omitempty"`
	Group   string `json:"group,omitempty"`
}

type SubOptsFirstEvent string

const (
	SubOptsFirstEventOldest SubOptsFirstEvent = "oldest"
	SubOptsFirstEventNewest SubOptsFirstEvent = "newest"
)

type SubscriptionOptions struct {
	FirstEvent   *SubOptsFirstEvent `json:"firstEvent,omitempty"`
	BatchEnabled *bool              `json:"batchEnabled,omitempty"`
	BatchTimeout *FFDuration        `json:"batchTimeout,omitempty"`
	BatchSize    *uint64            `json:"batchSize,omitempty"`
	ExpandData   *bool              `json:"expandData,omitempty"`
}

type Subscription struct {
	ID        *uuid.UUID          `json:"id"`
	Namespace string              `json:"namespace"`
	Name      string              `json:"name"`
	Transport string              `json:"transport"`
	Filter    SubscriptionFilter  `json:"filter"`
	Options   SubscriptionOptions `json:"options"`
	Ephemeral bool                `json:"ephemeral,omitempty"`
	Created   *FFTime             `json:"created"`
}

type SubscriptionRef struct {
	ID        *uuid.UUID `json:"id"`
	Namespace string     `json:"namespace"`
	Name      string     `json:"name"`
}

// Scan implements sql.Scanner
func (so *SubscriptionOptions) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return json.Unmarshal(src, &so)
	case string:
		return json.Unmarshal([]byte(src), &so)

	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, so)
	}

}

// Value implements sql.Valuer
func (so SubscriptionOptions) Value() (driver.Value, error) {
	return json.Marshal(&so)
}
