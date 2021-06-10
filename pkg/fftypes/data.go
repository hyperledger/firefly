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
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/firefly/internal/i18n"
)

type DataRef struct {
	ID   *UUID    `json:"id,omitempty"`
	Hash *Bytes32 `json:"hash,omitempty"`
}

type Data struct {
	ID        *UUID         `json:"id,omitempty"`
	Validator ValidatorType `json:"validator"`
	Blobstore bool          `json:"blobstore,omitempty"`
	Namespace string        `json:"namespace,omitempty"`
	Hash      *Bytes32      `json:"hash,omitempty"`
	Created   *FFTime       `json:"created,omitempty"`
	Datatype  *DatatypeRef  `json:"datatype,omitempty"`
	Value     Byteable      `json:"value"`
}

type DatatypeRef struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

func (dr *DatatypeRef) String() string {
	if dr == nil {
		return "null"
	}
	return fmt.Sprintf("%s/%s", dr.Name, dr.Version)
}

type DataRefs []*DataRef

func (d DataRefs) Hash() *Bytes32 {
	b, _ := json.Marshal(&d)
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32
}

func (d *Data) Seal(ctx context.Context) error {
	if d.Value == nil || d.Value.String() == "null" {
		return i18n.NewError(ctx, i18n.MsgDataValueIsNull)
	}
	if d.ID == nil {
		d.ID = NewUUID()
	}
	if d.Created == nil {
		d.Created = Now()
	}
	d.Hash = d.Value.Hash()
	return nil
}
