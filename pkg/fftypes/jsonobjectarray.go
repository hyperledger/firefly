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
	"database/sql/driver"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/i18n"
)

// JSONObjectArray is an array of JSONObject
type JSONObjectArray []JSONObject

// Scan implements sql.Scanner
func (jd *JSONObjectArray) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		*jd = JSONObjectArray{}
		return nil

	case []byte:
		if src == nil {
			*jd = JSONObjectArray{}
			return nil
		}
		return json.Unmarshal(src, &jd)

	case string:
		if src == "" {
			*jd = JSONObjectArray{}
			return nil
		}
		return json.Unmarshal([]byte(src), &jd)

	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, jd)
	}

}

func (jd JSONObjectArray) Value() (driver.Value, error) {
	b, err := json.Marshal(&jd)
	if err != nil {
		return nil, err
	}
	return string(b), err
}

func (jd JSONObjectArray) String() string {
	b, _ := json.Marshal(&jd)
	return string(b)
}

func (jd JSONObjectArray) Hash(jsonDesc string) (*Bytes32, error) {
	b, err := json.Marshal(&jd)
	if err != nil {
		return nil, i18n.NewError(context.Background(), i18n.MsgJSONObjectParseFailed, jsonDesc)
	}
	var b32 Bytes32 = sha256.Sum256(b)
	return &b32, nil
}
