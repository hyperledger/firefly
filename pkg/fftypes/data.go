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
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/firefly/internal/i18n"
)

type DataRef struct {
	ID   *UUID    `json:"id,omitempty"`
	Hash *Bytes32 `json:"hash,omitempty"`

	ValueSize int64 `json:"-"` // used internally for message size calculation, without full payload retrieval
}

type BlobRef struct {
	Hash   *Bytes32 `json:"hash"`
	Size   int64    `json:"size"`
	Name   string   `json:"name"`
	Public string   `json:"public,omitempty"`
}

type Data struct {
	ID        *UUID         `json:"id,omitempty"`
	Validator ValidatorType `json:"validator"`
	Namespace string        `json:"namespace,omitempty"`
	Hash      *Bytes32      `json:"hash,omitempty"`
	Created   *FFTime       `json:"created,omitempty"`
	Datatype  *DatatypeRef  `json:"datatype,omitempty"`
	Value     *JSONAny      `json:"value"`
	Blob      *BlobRef      `json:"blob,omitempty"`

	ValueSize int64 `json:"-"` // Used internally for message size calcuation, without full payload retrieval
}

func (br *BlobRef) BatchBlobRef(broadcast bool) *BlobRef {
	if br == nil {
		return nil
	}
	// For broadcast data the blob reference contains the "public" (shared storage) reference, which
	// must have been allocated to this data item before sealing the batch.
	if broadcast {
		return br
	}
	// For private we omit the "public" ref in all cases, to avoid an potential for the batch pay to change due
	// to the same data being allocated by the same data being sent in a broadcast batch (thus assigining a public ref).
	return &BlobRef{
		Hash: br.Hash,
		Size: br.Size,
		Name: br.Name,
	}
}

// BatchData is the fields in a data record that are assured to be consistent on all parties.
// This is what is transferred and hashed in a batch payload between nodes.
func (d *Data) BatchData(broadcast bool) *Data {
	return &Data{
		ID:        d.ID,
		Validator: d.Validator,
		Namespace: d.Namespace,
		Hash:      d.Hash,
		Datatype:  d.Datatype,
		Value:     d.Value,
		Blob:      d.Blob.BatchBlobRef(broadcast),
	}
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
		return NullString
	}
	return fmt.Sprintf("%s/%s", dr.Name, dr.Version)
}

type DataRefs []*DataRef

func (d DataRefs) Hash() *Bytes32 {
	b, _ := json.Marshal(&d)
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32
}

func CheckValidatorType(ctx context.Context, validator ValidatorType) error {
	switch validator {
	case ValidatorTypeJSON, ValidatorTypeNone, ValidatorTypeSystemDefinition:
		return nil
	default:
		return i18n.NewError(ctx, i18n.MsgUnknownValidatorType, validator)
	}
}

const dataSizeEstimateBase = int64(256)

func (d *Data) EstimateSize() int64 {
	// For now we have a static estimate for the size of the serialized outer structure.
	// As long as this has been persisted, the value size will represent the length
	return dataSizeEstimateBase + d.ValueSize
}

func (d *Data) CalcHash(ctx context.Context) (*Bytes32, error) {
	if d.Value == nil {
		d.Value = JSONAnyPtr(NullString)
	}
	valueIsNull := d.Value.String() == NullString
	if valueIsNull && (d.Blob == nil || d.Blob.Hash == nil) {
		return nil, i18n.NewError(ctx, i18n.MsgDataValueIsNull)
	}
	// The hash is either the blob hash, the value hash, or if both are supplied
	// (e.g. a blob with associated metadata) it a hash of the two HEX hashes
	// concattenated together (no spaces or separation).
	switch {
	case !valueIsNull && (d.Blob == nil || d.Blob.Hash == nil):
		return d.Value.Hash(), nil
	case valueIsNull && d.Blob != nil && d.Blob.Hash != nil:
		return d.Blob.Hash, nil
	default:
		hash := sha256.New()
		hash.Write([]byte(d.Value.Hash().String()))
		hash.Write([]byte(d.Blob.Hash.String()))
		return HashResult(hash), nil
	}
}

func (d *Data) Seal(ctx context.Context, blob *Blob) (err error) {
	if d.Validator == "" {
		d.Validator = ValidatorTypeJSON
	}
	if d.ID == nil {
		d.ID = NewUUID()
	}
	if d.Created == nil {
		d.Created = Now()
	}
	if blob != nil {
		if d.Blob == nil || !d.Blob.Hash.Equals(blob.Hash) {
			return i18n.NewError(ctx, i18n.MsgBlobMismatchSealingData)
		}
		d.Blob.Size = blob.Size
		if d.Value != nil {
			valJSON := d.Value.JSONObjectNowarn()
			jName := valJSON.GetString("name")
			if jName != "" {
				d.Blob.Name = jName
			} else {
				jPath := valJSON.GetString("path")
				jFilename := valJSON.GetString("filename")
				if jPath != "" && jFilename != "" {
					d.Blob.Name = fmt.Sprintf("%s/%s", jPath, jFilename)
				} else if jFilename != "" {
					d.Blob.Name = jFilename
				}
			}
		}
	} else if d.Blob != nil && d.Blob.Hash != nil {
		return i18n.NewError(ctx, i18n.MsgBlobMismatchSealingData)
	}
	d.Hash, err = d.CalcHash(ctx)
	if err == nil {
		err = CheckValidatorType(ctx, d.Validator)
	}
	return err
}
