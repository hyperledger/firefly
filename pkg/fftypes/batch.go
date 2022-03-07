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
	"crypto/sha256"
	"encoding/json"
)

// BatchType is the type of a batch
type BatchType = FFEnum

var (
	// BatchTypeBroadcast is a batch that is broadcast via the shared data interface
	BatchTypeBroadcast BatchType = ffEnum("batchtype", "broadcast")
	// BatchTypePrivate is a batch that is sent privately to a group
	BatchTypePrivate BatchType = ffEnum("batchtype", "private")
)

// BatchHeader is the common fields between the serialized batch, and the batch manifest
type BatchHeader struct {
	ID        *UUID     `json:"id"`
	Type      BatchType `json:"type" ffenum:"batchtype"`
	Namespace string    `json:"namespace"`
	Node      *UUID     `json:"node,omitempty"`
	SignerRef
	Group *Bytes32 `jdon:"group,omitempty"`
	Hash  *Bytes32 `json:"hash"`
}

// BatchManifest is all we need to persist to be able to reconstitute
// an identical batch. It can be generated from a received batch to
// confirm you have received an identical batch to that sent.
type BatchManifest struct {
	ID       *UUID        `json:"id"`
	Messages []MessageRef `json:"messages"`
	Data     []DataRef    `json:"data"`
}

// Batch is the full payload object used in-flight.
type Batch struct {
	BatchHeader
	Payload    BatchPayload `json:"payload"`
	PayloadRef string       `json:"payloadRef,omitempty"`
}

// BatchPersisted is the structure written to the database
type BatchPersisted struct {
	BatchHeader
	Manifest   string         `json:"manifest"` // not automatically parsed
	TX         TransactionRef `json:"tx"`
	PayloadRef string         `json:"payloadRef,omitempty"`
	Created    *FFTime        `json:"created"`
	Confirmed  *FFTime        `json:"confirmed"`
}

// BatchPayload contains the full JSON of the messages and data, but
// importantly only the immutable parts of the messages/data.
// In v0.13 and earlier, we used the whole of this payload object to
// form the hash of the in-flight batch. Subsequent to that we only
// calculate the hash of the manifest, as that contains the hashes
// of all the messages and data (thus minimizing the overhead of
// calculating the hash).
// - See Message.BatchMessage() and Data.BatchData()
type BatchPayload struct {
	TX       TransactionRef `json:"tx"`
	Messages []*Message     `json:"messages"`
	Data     []*Data        `json:"data"`
}

func (bm *BatchManifest) String() string {
	b, _ := json.Marshal(&bm)
	return string(b)
}

func (ma *BatchPayload) Hash() *Bytes32 {
	b, _ := json.Marshal(&ma)
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32
}

func (ma *BatchPayload) Manifest(id *UUID) *BatchManifest {
	tm := &BatchManifest{
		ID:       id,
		Messages: make([]MessageRef, len(ma.Messages)),
		Data:     make([]DataRef, len(ma.Data)),
	}
	for i, m := range ma.Messages {
		tm.Messages[i].ID = m.Header.ID
		tm.Messages[i].Hash = m.Hash
	}
	for i, d := range ma.Data {
		tm.Data[i].ID = d.ID
		tm.Data[i].Hash = d.Hash
	}
	return tm
}
func (b *Batch) Manifest() *BatchManifest {
	if b == nil {
		return nil
	}
	return b.Payload.Manifest(b.ID)
}
