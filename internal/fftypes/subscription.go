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
	"github.com/google/uuid"
)

type SubscriptionFilter struct {
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
	FirstEvent   SubOptsFirstEvent `json:"firstEvent"`
	BatchEnabled bool              `json:"batchEnabled"`
	BatchTimeout *FFDuration       `json:"batchTimeout"`
	BatchSize    uint              `json:"batchSize"`
}

type Subscription struct {
	ID        *uuid.UUID           `json:"id"`
	Namespace string               `json:"namespace"`
	Name      string               `json:"name"`
	Events    EventTypes           `json:"events"`
	Filter    *SubscriptionFilter  `json:"filter,omitempty"`
	Options   *SubscriptionOptions `json:"options"`
	Created   *FFTime              `json:"created"`
}
