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
	"database/sql/driver"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/i18n"
)

type Batch struct {
	ID         *uuid.UUID     `json:"id"`
	Namespace  string         `json:"namespace"`
	Type       MessageType    `json:"type"`
	Author     string         `json:"author"`
	Hash       *Bytes32       `json:"hash"`
	Created    int64          `json:"created"`
	Confirmed  int64          `json:"confirmed"`
	Payload    BatchPayload   `json:"payload"`
	PayloadRef *Bytes32       `json:"payloadRef,omitempty"`
	TX         TransactionRef `json:"tx"`
}

type BatchPayload struct {
	Messages []*Message `json:"messages"`
	Data     []*Data    `json:"data"`
}

// Value implements sql.Valuer
func (ma BatchPayload) Value() (driver.Value, error) {
	return json.Marshal(&ma)
}

func (ma BatchPayload) Hash() *Bytes32 {
	b, _ := json.Marshal(&ma)
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32
}

// Scan implements sql.Scanner
func (ma *BatchPayload) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil

	case string, []byte:
		if src == "" {
			return nil
		}
		return json.Unmarshal(src.([]byte), &ma)

	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, ma)
	}

}
