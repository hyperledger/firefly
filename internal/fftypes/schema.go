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

import "github.com/google/uuid"

type SchemaType string

const (
	SchemaTypeJSONSchema SchemaType = "jsonschema"
)

type Schema struct {
	ID        *uuid.UUID `json:"id,omitempty"`
	Type      SchemaType `json:"type"`
	Namespace string     `json:"namespace,omitempty"`
	Entity    string     `json:"entity,omitempty"`
	Version   string     `json:"version,omitempty"`
	Hash      *Bytes32   `json:"hash,omitempty"`
	Created   int64      `json:"created,omitempty"`
	Value     JSONData   `json:"value,omitempty"`
}
