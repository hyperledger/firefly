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

type ValidatorType string

const (
	// ValidatorTypeJSON is the validator type for JSON Schema validation
	ValidatorTypeJSON ValidatorType = "json"
	// ValidatorTypeBLOB is the validator type for binary blob data, that is passed through without any parsing or validation
	ValidatorTypeBLOB ValidatorType = "blob"
	// ValidatorTypeDataDefinition is the validator type for data definitions, which are a built in type that defines other types
	ValidatorTypeDataDefinition ValidatorType = "datadef"
)

// DataDefinition is the structure defining a data definition, such as a JSON schema
type DataDefinition struct {
	ID        *UUID         `json:"id,omitempty"`
	Validator ValidatorType `json:"validator"`
	Namespace string        `json:"namespace,omitempty"`
	Name      string        `json:"name,omitempty"`
	Version   string        `json:"version,omitempty"`
	Hash      *Bytes32      `json:"hash,omitempty"`
	Created   *FFTime       `json:"created,omitempty"`
	Value     JSONObject    `json:"value,omitempty"`
}
