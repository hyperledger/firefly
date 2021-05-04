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
	"crypto/sha256"
	"encoding/json"
	"sort"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/i18n"
)

type DataType string

const (
	DataTypeDefinition DataType = "definition"
	DataTypeJSON       DataType = "json"
	DataTypeBLOB       DataType = "blob"
)

type DataRef struct {
	ID   *uuid.UUID `json:"id,omitempty"`
	Hash *Bytes32   `json:"hash,omitempty"`
}

type Data struct {
	ID        *uuid.UUID `json:"id,omitempty"`
	Type      DataType   `json:"type"`
	Namespace string     `json:"namespace,omitempty"`
	Hash      *Bytes32   `json:"hash,omitempty"`
	Created   int64      `json:"created,omitempty"`
	Schema    *SchemaRef `json:"schema,omitempty"`
	Value     JSONData   `json:"value,omitempty"`
}

type SchemaRef struct {
	Entity  string `json:"entity,omitempty"`
	Version string `json:"version,omitempty"`
}

type DataRefSortable []DataRef

func (d DataRefSortable) Len() int      { return len(d) }
func (d DataRefSortable) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d DataRefSortable) Less(i, j int) bool {
	return d[j].Hash != nil && (d[i].Hash == nil || d[j].Hash != nil && d[i].Hash.String() < d[j].Hash.String())
}

func (d DataRefSortable) Hash(ctx context.Context) (*Bytes32, error) {
	sort.Sort(d)
	var strArray = make([]string, 0, len(d))
	for i, de := range d {
		if de.Hash == nil {
			return nil, i18n.NewError(ctx, i18n.MsgMissingDataHashIndex, i)
		}
		strArray = append(strArray, de.Hash.String())
	}
	b, _ := json.Marshal(&strArray)
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32, nil
}
