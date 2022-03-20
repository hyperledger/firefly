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
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type FFIParamValidator interface {
	Compile(ctx jsonschema.CompilerContext, m map[string]interface{}) (jsonschema.ExtSchema, error)
	GetMetaSchema() *jsonschema.Schema
	GetExtensionName() string
}

type FFIReference struct {
	ID      *UUID  `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type FFI struct {
	ID          *UUID        `json:"id,omitempty"`
	Message     *UUID        `json:"message,omitempty"`
	Namespace   string       `json:"namespace,omitempty"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Version     string       `json:"version"`
	Methods     []*FFIMethod `json:"methods,omitempty"`
	Events      []*FFIEvent  `json:"events,omitempty"`
}

type FFIMethod struct {
	ID          *UUID     `json:"id,omitempty"`
	Contract    *UUID     `json:"contract,omitempty"`
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace,omitempty"`
	Pathname    string    `json:"pathname"`
	Description string    `json:"description"`
	Params      FFIParams `json:"params"`
	Returns     FFIParams `json:"returns"`
}

type FFIEventDefinition struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Params      FFIParams `json:"params"`
}

type FFIEvent struct {
	ID        *UUID  `json:"id,omitempty"`
	Contract  *UUID  `json:"contract,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Pathname  string `json:"pathname,omitempty"`
	FFIEventDefinition
}

type FFIParam struct {
	Name   string   `json:"name"`
	Schema *JSONAny `json:"schema,omitempty"`
}

type FFIParams []*FFIParam

type FFIGenerationRequest struct {
	Namespace   string   `json:"namespace,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Input       *JSONAny `json:"input"`
}

func (f *FFI) Validate(ctx context.Context, existing bool) (err error) {
	if err = ValidateFFNameField(ctx, f.Namespace, "namespace"); err != nil {
		return err
	}
	if err = ValidateFFNameField(ctx, f.Name, "name"); err != nil {
		return err
	}
	if err = ValidateFFNameField(ctx, f.Version, "version"); err != nil {
		return err
	}
	return nil
}

func (f *FFI) Topic() string {
	return typeNamespaceNameTopicHash("ffi", f.Namespace, f.Name)
}

func (f *FFI) SetBroadcastMessage(msgID *UUID) {
	f.Message = msgID
}

// Scan implements sql.Scanner
func (m *FFIParams) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		m = nil
		return nil
	case string:
		return json.Unmarshal([]byte(src), &m)
	case []byte:
		return json.Unmarshal(src, &m)
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, m)
	}
}

func (m FFIParams) Value() (driver.Value, error) {
	bytes, _ := json.Marshal(m)
	return bytes, nil
}
