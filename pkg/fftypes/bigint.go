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
	"database/sql/driver"
	"encoding/json"
	"math/big"

	"github.com/hyperledger/firefly/internal/i18n"
)

const MaxBigIntHexLength = 65

// BigInt is a wrapper on a Go big.Int that standardizes JSON and DB serialization
type BigInt big.Int

func (i BigInt) MarshalText() ([]byte, error) {
	// Represent as base 10 string in Marshalled JSON
	// This could become configurable to other options, such as:
	// - Hex formatted string
	// - Number up to max float64, then string if larger
	return []byte((*big.Int)(&i).Text(10)), nil
}

func (i *BigInt) UnmarshalJSON(b []byte) error {
	var val interface{}
	if err := json.Unmarshal(b, &val); err != nil {
		return i18n.WrapError(context.Background(), err, i18n.MsgBigIntParseFailed, b)
	}
	switch val := val.(type) {
	case string:
		if _, ok := i.Int().SetString(val, 0); !ok {
			return i18n.NewError(context.Background(), i18n.MsgBigIntParseFailed, b)
		}
		return nil
	case float64:
		i.Int().SetInt64(int64(val))
		return nil
	default:
		return i18n.NewError(context.Background(), i18n.MsgBigIntParseFailed, b)
	}
}

func (i *BigInt) Value() (driver.Value, error) {
	// Represent as base 16 string in database, to allow a 64 character limit
	res := (*big.Int)(i).Text(16)
	if len(res) > MaxBigIntHexLength {
		return nil, i18n.NewError(context.Background(), i18n.MsgBigIntTooLarge, len(res), MaxBigIntHexLength)
	}
	return res, nil
}

func (i *BigInt) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil
	case string:
		if src == "" {
			return nil
		}
		// Scan is different to JSON deserialization - always read as HEX (without any 0x prefix)
		if _, ok := i.Int().SetString(src, 16); !ok {
			return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, i)
		}
		return nil
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, i)
	}
}

func (i *BigInt) Int() *big.Int {
	return (*big.Int)(i)
}

func (i *BigInt) Equals(i2 *BigInt) bool {
	switch {
	case i == nil && i2 == nil:
		return true
	case i == nil || i2 == nil:
		return false
	default:
		return (*big.Int)(i).Cmp((*big.Int)(i2)) == 0
	}
}
