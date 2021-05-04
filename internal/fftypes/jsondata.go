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

	"github.com/kaleido-io/firefly/internal/i18n"
)

// JSONData is a holder of a hash, that can be used to correlate onchain data with off-chain data.
type JSONData map[string]interface{}

// Scan implements sql.Scanner
func (jd *JSONData) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil

	case string, []byte:
		if src == "" {
			return nil
		}
		return json.Unmarshal(src.([]byte), &jd)

	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, jd)
	}

}

// Value implements sql.Valuer
func (jd JSONData) Value() (driver.Value, error) {
	return json.Marshal(&jd)
}

func (jd *JSONData) String() string {
	b, _ := json.Marshal(&jd)
	return string(b)
}

func (jd *JSONData) Hash(ctx context.Context, jsonDesc string) (*Bytes32, error) {
	b, err := json.Marshal(&jd)
	if err != nil {
		return nil, i18n.NewError(ctx, i18n.MsgJSONDataParseFailed, jsonDesc)
	}
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32, nil
}
