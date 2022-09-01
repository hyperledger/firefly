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

package core

import (
	"context"
	"database/sql/driver"
	"encoding/json"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
)

type ContractListener struct {
	ID        *fftypes.UUID            `ffstruct:"ContractListener" json:"id,omitempty" ffexcludeinput:"true"`
	Interface *fftypes.FFIReference    `ffstruct:"ContractListener" json:"interface,omitempty" ffexcludeinput:"postContractAPIListeners"`
	Namespace string                   `ffstruct:"ContractListener" json:"namespace,omitempty" ffexcludeinput:"true"`
	Name      string                   `ffstruct:"ContractListener" json:"name,omitempty"`
	BackendID string                   `ffstruct:"ContractListener" json:"backendId,omitempty" ffexcludeinput:"true"`
	Location  *fftypes.JSONAny         `ffstruct:"ContractListener" json:"location,omitempty"`
	Created   *fftypes.FFTime          `ffstruct:"ContractListener" json:"created,omitempty" ffexcludeinput:"true"`
	Event     *FFISerializedEvent      `ffstruct:"ContractListener" json:"event,omitempty" ffexcludeinput:"postContractAPIListeners"`
	Signature string                   `ffstruct:"ContractListener" json:"signature" ffexcludeinput:"true"`
	Topic     string                   `ffstruct:"ContractListener" json:"topic,omitempty"`
	Options   *ContractListenerOptions `ffstruct:"ContractListener" json:"options,omitempty"`
}

type ContractListenerWithStatus struct {
	ContractListener
	Status interface{} `ffstruct:"ContractListenerWithStatus" json:"status,omitempty" ffexcludeinput:"true"`
}
type ContractListenerOptions struct {
	FirstEvent string `ffstruct:"ContractListenerOptions" json:"firstEvent,omitempty"`
}

type ListenerStatusError struct {
	StatusError string `ffstruct:"ListenerStatusError" json:"error,omitempty"`
}

type ContractListenerInput struct {
	ContractListener
	EventPath string `ffstruct:"ContractListener" json:"eventPath,omitempty"`
}

type FFISerializedEvent struct {
	fftypes.FFIEventDefinition
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
		return i18n.NewError(context.Background(), i18n.MsgTypeRestoreFailed, src, fse)
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
		return i18n.NewError(context.Background(), i18n.MsgTypeRestoreFailed, src, o)
	}
}

func (o ContractListenerOptions) Value() (driver.Value, error) {
	bytes, _ := json.Marshal(o)
	return bytes, nil
}
