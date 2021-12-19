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

package fftypes

import (
	"context"
	"database/sql/driver"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/i18n"
)

type ContractSubscription struct {
	ID         *UUID               `json:"id,omitempty"`
	Interface  *UUID               `json:"interface,omitempty"`
	Namespace  string              `json:"namespace,omitempty"`
	Name       string              `json:"name,omitempty"`
	ProtocolID string              `json:"protocolId,omitempty"`
	Location   Byteable            `json:"location,omitempty"`
	Created    *FFTime             `json:"created,omitempty"`
	Event      *FFISerializedEvent `json:"event,omitempty"`
}

type ContractSubscriptionInput struct {
	ContractSubscription
	EventID *UUID `json:"eventId,omitempty"`
}

type FFISerializedEvent struct {
	FFIEventDefinition
}

// Scan implements sql.Scanner
func (fse *FFISerializedEvent) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		fse = nil
		return nil
	case []byte:
		return json.Unmarshal(src, &fse)
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, fse)
	}
}

func (fse FFISerializedEvent) Value() (driver.Value, error) {
	bytes, _ := json.Marshal(fse)
	return bytes, nil
}
