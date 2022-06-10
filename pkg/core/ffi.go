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
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// FFIInputType is the type of a JSON field in a request to FireFly's API
type FFIInputType = FFEnum

var (
	// FFIInputTypeInteger is a json integer or string to be treated as an integer
	FFIInputTypeInteger = ffEnum("ffiinputtype", "integer")
	// FFIInputTypeString is a JSON string
	FFIInputTypeString = ffEnum("ffiinputtype", "string")
	// FFIInputTypeArray is a JSON boolean
	FFIInputTypeBoolean = ffEnum("ffiinputtype", "boolean")
	// FFIInputTypeArray is a JSON array
	FFIInputTypeArray = ffEnum("ffiinputtype", "array")
	// FFIInputTypeObject is a JSON object
	FFIInputTypeObject = ffEnum("ffiinputtype", "object")
)

type FFIParamValidator interface {
	Compile(ctx jsonschema.CompilerContext, m map[string]interface{}) (jsonschema.ExtSchema, error)
	GetMetaSchema() *jsonschema.Schema
	GetExtensionName() string
}

type FFIReference struct {
	ID      *fftypes.UUID `ffstruct:"FFIReference" json:"id,omitempty"`
	Name    string        `ffstruct:"FFIReference" json:"name,omitempty"`
	Version string        `ffstruct:"FFIReference" json:"version,omitempty"`
}

type FFI struct {
	ID          *fftypes.UUID `ffstruct:"FFI" json:"id,omitempty" ffexcludeinput:"true"`
	Message     *fftypes.UUID `ffstruct:"FFI" json:"message,omitempty" ffexcludeinput:"true"`
	Namespace   string        `ffstruct:"FFI" json:"namespace,omitempty" ffexcludeinput:"true"`
	Name        string        `ffstruct:"FFI" json:"name"`
	Description string        `ffstruct:"FFI" json:"description"`
	Version     string        `ffstruct:"FFI" json:"version"`
	Methods     []*FFIMethod  `ffstruct:"FFI" json:"methods,omitempty"`
	Events      []*FFIEvent   `ffstruct:"FFI" json:"events,omitempty"`
}

type FFIMethod struct {
	ID          *fftypes.UUID      `ffstruct:"FFIMethod" json:"id,omitempty" ffexcludeinput:"true"`
	Interface   *fftypes.UUID      `ffstruct:"FFIMethod" json:"interface,omitempty" ffexcludeinput:"true"`
	Name        string             `ffstruct:"FFIMethod" json:"name"`
	Namespace   string             `ffstruct:"FFIMethod" json:"namespace,omitempty" ffexcludeinput:"true"`
	Pathname    string             `ffstruct:"FFIMethod" json:"pathname" ffexcludeinput:"true"`
	Description string             `ffstruct:"FFIMethod" json:"description"`
	Params      FFIParams          `ffstruct:"FFIMethod" json:"params"`
	Returns     FFIParams          `ffstruct:"FFIMethod" json:"returns"`
	Details     fftypes.JSONObject `ffstruct:"FFIMethod" json:"details,omitempty"`
}

type FFIEventDefinition struct {
	Name        string             `ffstruct:"FFIEvent" json:"name"`
	Description string             `ffstruct:"FFIEvent" json:"description"`
	Params      FFIParams          `ffstruct:"FFIEvent" json:"params"`
	Details     fftypes.JSONObject `ffstruct:"FFIEvent" json:"details,omitempty"`
}

type FFIEvent struct {
	ID        *fftypes.UUID `ffstruct:"FFIEvent" json:"id,omitempty" ffexcludeinput:"true"`
	Interface *fftypes.UUID `ffstruct:"FFIEvent" json:"interface,omitempty" ffexcludeinput:"true"`
	Namespace string        `ffstruct:"FFIEvent" json:"namespace,omitempty" ffexcludeinput:"true"`
	Pathname  string        `ffstruct:"FFIEvent" json:"pathname,omitempty" ffexcludeinput:"true"`
	Signature string        `ffstruct:"FFIEvent" json:"signature" ffexcludeinput:"true"`
	FFIEventDefinition
}

type FFIParam struct {
	Name   string           `ffstruct:"FFIParam" json:"name"`
	Schema *fftypes.JSONAny `ffstruct:"FFIParam" json:"schema,omitempty"`
}

type FFIParams []*FFIParam

type FFIGenerationRequest struct {
	Namespace   string           `ffstruct:"FFIGenerationRequest" json:"namespace,omitempty"`
	Name        string           `ffstruct:"FFIGenerationRequest" json:"name"`
	Description string           `ffstruct:"FFIGenerationRequest" json:"description"`
	Version     string           `ffstruct:"FFIGenerationRequest" json:"version"`
	Input       *fftypes.JSONAny `ffstruct:"FFIGenerationRequest" json:"input"`
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

func (f *FFI) SetBroadcastMessage(msgID *fftypes.UUID) {
	f.Message = msgID
}

// Scan implements sql.Scanner
func (p *FFIParams) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		p = nil
		return nil
	case string:
		return json.Unmarshal([]byte(src), &p)
	case []byte:
		return json.Unmarshal(src, &p)
	default:
		return i18n.NewError(context.Background(), i18n.MsgTypeRestoreFailed, src, p)
	}
}

func (p FFIParams) Value() (driver.Value, error) {
	bytes, _ := json.Marshal(p)
	return bytes, nil
}
