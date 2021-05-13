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

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/i18n"
)

type MessageType string

const (
	MessageTypeDefinition MessageType = "definition"
	MessageTypeBroadcast  MessageType = "broadcast"
	MessageTypePrivate    MessageType = "private"
)

// MessageHeader contains all fields that contribute to the hash
type MessageHeader struct {
	ID        *uuid.UUID     `json:"id,omitempty"`
	CID       *uuid.UUID     `json:"cid,omitempty"`
	Type      MessageType    `json:"type"`
	TX        TransactionRef `json:"tx,omitempty"`
	Author    string         `json:"author,omitempty"`
	Created   int64          `json:"created,omitempty"`
	Namespace string         `json:"namespace,omitempty"`
	Topic     string         `json:"topic,omitempty"`
	Context   string         `json:"context,omitempty"`
	Group     *uuid.UUID     `json:"group,omitempty"`
	DataHash  *Bytes32       `json:"datahash,omitempty"`
}

type Message struct {
	Header    MessageHeader `json:"header"`
	Hash      *Bytes32      `json:"hash,omitempty"`
	BatchID   *uuid.UUID    `json:"batchId,omitempty"`
	Sequence  int64         `json:"sequence,omitempty"`
	Confirmed int64         `json:"confirmed,omitempty"`
	Data      DataRefs      `json:"data"`
}

func (h *MessageHeader) Hash() *Bytes32 {
	b, _ := json.Marshal(&h)
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32
}

func (m *Message) Seal(ctx context.Context) (err error) {
	if m.Header.ID == nil {
		m.Header.ID = NewUUID()
	}
	if m.Header.Created == 0 {
		m.Header.Created = NowMillis()
	}
	m.Confirmed = 0
	if m.Data == nil {
		m.Data = DataRefs{}
	}
	err = m.DupDataCheck(ctx)
	if err == nil {
		m.Header.DataHash = m.Data.Hash(ctx)
		m.Hash = m.Header.Hash()
	}
	return err
}

func (m *Message) DupDataCheck(ctx context.Context) (err error) {
	dupCheck := make(map[string]bool)
	for i, d := range m.Data {
		if d.ID == nil || d.Hash == nil {
			return i18n.NewError(ctx, i18n.MsgNilDataReferenceSealFail, i)
		}
		if dupCheck[d.ID.String()] || dupCheck[d.Hash.String()] {
			return i18n.NewError(ctx, i18n.MsgDupDataReferenceSealFail, i)
		}
		dupCheck[d.ID.String()] = true
		dupCheck[d.Hash.String()] = true
	}
	return nil
}

func (m *Message) Verify(ctx context.Context) error {
	err := m.DupDataCheck(ctx)
	if err != nil {
		return err
	}
	if m.Hash == nil || m.Header.DataHash == nil {
		return i18n.NewError(ctx, i18n.MsgVerifyFailedNilHashes)
	}
	headerHash := m.Header.Hash()
	dataHash := m.Data.Hash(ctx)
	if *m.Hash != *headerHash || *m.Header.DataHash != *dataHash {
		return i18n.NewError(ctx, i18n.MsgVerifyFailedInvalidHashes, m.Hash.String(), headerHash.String(), m.Header.DataHash.String(), dataHash.String())
	}
	return nil
}
