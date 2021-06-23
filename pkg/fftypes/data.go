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

type BlobRef struct {
	Hash   *Bytes32 `json:"hash"`
	Public string   `json:"public"`
}

type Data struct {
	ID        *UUID         `json:"id,omitempty"`
	Validator ValidatorType `json:"validator"`
	Namespace string        `json:"namespace,omitempty"`
	Hash      *Bytes32      `json:"hash,omitempty"`
	Created   *FFTime       `json:"created,omitempty"`
	Datatype  *DatatypeRef  `json:"datatype,omitempty"`
	Value     Byteable      `json:"value"`
	Blob      *BlobRef      `json:"blob,omitempty"`
}

type DataAndBlob struct {
	Data *Data
	Blob *Blob
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

func (d *Data) CalcHash(ctx context.Context) (*Bytes32, error) {
	if (d.Value == nil || d.Value.String() == "null") && (d.Blob == nil || d.Blob.Hash == nil) {
		return nil, i18n.NewError(ctx, i18n.MsgDataValueIsNull)
	}
	// The hash is either the blob hash, the value hash, or if both are supplied
	// (e.g. a blob with associated metadata) it a hash of the two HEX hashes
	// concattenated together (no spaces or separation).
	switch {
	case d.Value != nil && (d.Blob == nil || d.Blob.Hash == nil):
		return d.Value.Hash(), nil
	case d.Value == nil && d.Blob != nil && d.Blob.Hash != nil:
		return d.Blob.Hash, nil
	default:
		hash := sha256.New()
		hash.Write([]byte(d.Value.Hash().String()))
		hash.Write([]byte(d.Blob.Hash.String()))
		return HashResult(hash), nil
	}
}

func (d *Data) Seal(ctx context.Context) (err error) {
	if d.ID == nil {
		d.ID = NewUUID()
	}
	if d.Created == nil {
		d.Created = Now()
	}
	d.Hash, err = d.CalcHash(ctx)
	return err
}
