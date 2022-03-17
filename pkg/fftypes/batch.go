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
	BatchTypeBroadcast = ffEnum("batchtype", "broadcast")
	// BatchTypePrivate is a batch that is sent privately to a group
	BatchTypePrivate = ffEnum("batchtype", "private")
)

const (
	ManifestVersionUnset uint = 0
	ManifestVersion1     uint = 1
)

// BatchHeader is the common fields between the serialized batch, and the batch manifest
type BatchHeader struct {
	ID        *UUID     `json:"id"`
	Type      BatchType `json:"type" ffenum:"batchtype"`
	Namespace string    `json:"namespace"`
	Node      *UUID     `json:"node,omitempty"`
	SignerRef
	Group   *Bytes32 `jdon:"group,omitempty"`
	Created *FFTime  `json:"created"`
}

type MessageManifestEntry struct {
	MessageRef
	Topics int `json:"topics"` // We only need the count, to be able to match up the pins
}

// BatchManifest is all we need to persist to be able to reconstitute
// an identical batch, and also all of the fields that are protected by
// the hash of the batch.
// It can be generated from a received batch to
// confirm you have received an identical batch to that sent.
type BatchManifest struct {
	Version uint           `json:"version"`
	ID      *UUID          `json:"id"`
	TX      TransactionRef `json:"tx"`
	SignerRef
	Messages []*MessageManifestEntry `json:"messages"`
	Data     DataRefs                `json:"data"`
}

// Batch is the full payload object used in-flight.
type Batch struct {
	BatchHeader
	Hash    *Bytes32     `json:"hash"`
	Payload BatchPayload `json:"payload"`
}

// BatchPersisted is the structure written to the database
type BatchPersisted struct {
	BatchHeader
	Hash       *Bytes32       `json:"hash"`
	Manifest   *JSONAny       `json:"manifest"`
	TX         TransactionRef `json:"tx"`
	PayloadRef string         `json:"payloadRef,omitempty"`
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
	Data     DataArray      `json:"data"`
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
		Version:  ManifestVersion1,
		ID:       id,
		TX:       ma.TX,
		Messages: make([]*MessageManifestEntry, 0, len(ma.Messages)),
		Data:     make(DataRefs, 0, len(ma.Data)),
	}
	for _, m := range ma.Messages {
		if m != nil && m.Header.ID != nil {
			tm.Messages = append(tm.Messages, &MessageManifestEntry{
				MessageRef: MessageRef{
					ID:   m.Header.ID,
					Hash: m.Hash,
				},
				Topics: len(m.Header.Topics),
			})
		}
	}
	for _, d := range ma.Data {
		if d != nil && d.ID != nil {
			tm.Data = append(tm.Data, &DataRef{
				ID:   d.ID,
				Hash: d.Hash,
			})
		}
	}
	return tm
}

func (b *BatchPersisted) GenManifest(messages []*Message, data DataArray) *BatchManifest {
	return (&BatchPayload{
		TX:       b.TX,
		Messages: messages,
		Data:     data,
	}).Manifest(b.ID)
}

func (b *BatchPersisted) Inflight(messages []*Message, data DataArray) *Batch {
	return &Batch{
		BatchHeader: b.BatchHeader,
		Hash:        b.Hash,
		Payload: BatchPayload{
			TX:       b.TX,
			Messages: messages,
			Data:     data,
		},
	}
}

// Confirmed generates a newly confirmed persisted batch, including (re-)generating the manifest
func (b *Batch) Confirmed() (*BatchPersisted, *BatchManifest) {
	manifest := b.Payload.Manifest(b.ID)
	manifestString := manifest.String()
	return &BatchPersisted{
		BatchHeader: b.BatchHeader,
		Hash:        b.Hash,
		TX:          b.Payload.TX,
		Manifest:    JSONAnyPtr(manifestString),
		Confirmed:   Now(),
	}, manifest
}
