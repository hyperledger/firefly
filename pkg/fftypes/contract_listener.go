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

package fftypes

import (
	"context"
	"database/sql/driver"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/i18n"
)

type ContractListener struct {
	ID         *UUID                    `json:"id,omitempty"`
	Interface  *FFIReference            `json:"interface,omitempty"`
	Namespace  string                   `json:"namespace,omitempty"`
	Name       string                   `json:"name,omitempty"`
	ProtocolID string                   `json:"protocolId,omitempty"`
	Location   *JSONAny                 `json:"location,omitempty"`
	Created    *FFTime                  `json:"created,omitempty"`
	Event      *FFISerializedEvent      `json:"event,omitempty"`
	Options    *ContractListenerOptions `json:"options,omitempty"`
}

type ContractListenerOptions struct {
	FirstEvent string `json:"firstEvent,omitempty"`
}

type ContractListenerInput struct {
	ContractListener
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
	case string:
		return json.Unmarshal([]byte(src), &fse)
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

// Scan implements sql.Scanner
func (o *ContractListenerOptions) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		o = nil
		return nil
	case string:
		return json.Unmarshal([]byte(src), &o)
	case []byte:
		return json.Unmarshal(src, &o)
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, o)
	}
}

func (o ContractListenerOptions) Value() (driver.Value, error) {
	bytes, _ := json.Marshal(o)
	return bytes, nil
}
