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

	"github.com/hyperledger/firefly/internal/i18n"
)

const (
	// DefaultTopic will be set as the topic of any messages set without a topic
	DefaultTopic = "default"
)

// MessageType is the fundamental type of a message
type MessageType = FFEnum

var (
	// MessageTypeDefinition is a message broadcasting a definition of a system type, pre-defined by firefly (namespaces, members, data definitions, etc.)
	MessageTypeDefinition MessageType = ffEnum("messagetype", "definition")
	// MessageTypeBroadcast is a broadcast message, meaning it is intended to be visible by all parties in the network
	MessageTypeBroadcast MessageType = ffEnum("messagetype", "broadcast")
	// MessageTypePrivate is a private message, meaning it is only sent explicitly to individual parties in the network
	MessageTypePrivate MessageType = ffEnum("messagetype", "private")
	// MessageTypeGroupInit is a special private message that contains the definition of the group
	MessageTypeGroupInit MessageType = ffEnum("messagetype", "groupinit")
)

// MessageHeader contains all fields that contribute to the hash
// The order of the serialization mut not change, once released
type MessageHeader struct {
	ID     *UUID           `json:"id,omitempty"`
	CID    *UUID           `json:"cid,omitempty"`
	Type   MessageType     `json:"type" ffenum:"messagetype"`
	TxType TransactionType `json:"txtype,omitempty"`
	Identity
	Created   *FFTime     `json:"created,omitempty"`
	Namespace string      `json:"namespace,omitempty"`
	Group     *Bytes32    `json:"group,omitempty"`
	Topics    FFNameArray `json:"topics,omitempty"`
	Tag       string      `json:"tag,omitempty"`
	DataHash  *Bytes32    `json:"datahash,omitempty"`
}

// Message is the envelope by which coordinated data exchange can happen between parties in the network
// Data is passed by reference in these messages, and a chain of hashes covering the data and the
// details of the message, provides a verification against tampering.
type Message struct {
	Header    MessageHeader `json:"header"`
	Hash      *Bytes32      `json:"hash,omitempty"`
	BatchID   *UUID         `json:"batch,omitempty"`
	Local     bool          `json:"local,omitempty"`
	Rejected  bool          `json:"rejected,omitempty"`
	Pending   SortableBool  `json:"pending"`
	Confirmed *FFTime       `json:"confirmed,omitempty"`
	Data      DataRefs      `json:"data"`
	Pins      FFNameArray   `json:"pins,omitempty"`
	Sequence  int64         `json:"-"` // Local database sequence used internally for batch assembly
}

// MessageInOut allows API users to submit values in-line in the payload submitted, which
// will be broken out and stored separately during the call.
type MessageInOut struct {
	Message
	InlineData InlineData  `json:"data"`
	Group      *InputGroup `json:"group,omitempty"`
}

// InputGroup declares a group in-line for auotmatic resolution, without having to define a group up-front
type InputGroup struct {
	Name    string        `json:"name,omitempty"`
	Ledger  *UUID         `json:"ledger,omitempty"`
	Members []MemberInput `json:"members"`
}

// InlineData is an array of data references or values
type InlineData []*DataRefOrValue

// DataRefOrValue allows a value to be specified in-line in the data array of an input
// message, avoiding the need for a multiple API calls.
type DataRefOrValue struct {
	DataRef

	Validator ValidatorType `json:"validator,omitempty"`
	Datatype  *DatatypeRef  `json:"datatype,omitempty"`
	Value     Byteable      `json:"value,omitempty"`
	Blob      *BlobRef      `json:"blob,omitempty"`
}

// MessageRef is a lightweight data structure that can be used to refer to a message
type MessageRef struct {
	ID       *UUID    `json:"id,omitempty"`
	Sequence int64    `json:"sequence,omitempty"`
	Hash     *Bytes32 `json:"hash,omitempty"`
}

func (h *MessageHeader) Hash() *Bytes32 {
	b, _ := json.Marshal(&h)
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32
}

func (m *MessageInOut) SetInlineData(data []*Data) {
	m.InlineData = make(InlineData, len(data))
	for i, d := range data {
		m.InlineData[i] = &DataRefOrValue{
			DataRef: DataRef{
				ID:   d.ID,
				Hash: d.Hash,
			},
			Validator: d.Validator,
			Datatype:  d.Datatype,
			Value:     d.Value,
		}
	}
}

func (m *Message) Seal(ctx context.Context) (err error) {
	if len(m.Header.Topics) == 0 {
		m.Header.Topics = []string{DefaultTopic}
	}
	if m.Header.ID == nil {
		m.Header.ID = NewUUID()
	}
	if m.Header.Created == nil {
		m.Header.Created = Now()
	}
	m.Confirmed = nil
	m.Pending = true
	if m.Data == nil {
		m.Data = DataRefs{}
	}
	err = m.DupDataCheck(ctx)
	if err == nil {
		m.Header.DataHash = m.Data.Hash()
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
	if err := m.Header.Topics.Validate(ctx, "header.topics"); err != nil {
		return err
	}
	if m.Header.Tag != "" {
		if err := ValidateFFNameField(ctx, m.Header.Tag, "header.tag"); err != nil {
			return err
		}
	}
	err := m.DupDataCheck(ctx)
	if err != nil {
		return err
	}
	if m.Hash == nil || m.Header.DataHash == nil {
		return i18n.NewError(ctx, i18n.MsgVerifyFailedNilHashes)
	}
	headerHash := m.Header.Hash()
	dataHash := m.Data.Hash()
	if *m.Hash != *headerHash || *m.Header.DataHash != *dataHash {
		return i18n.NewError(ctx, i18n.MsgVerifyFailedInvalidHashes, m.Hash.String(), headerHash.String(), m.Header.DataHash.String(), dataHash.String())
	}
	return nil
}

func (m *Message) LocalSequence() int64 {
	return m.Sequence
}
