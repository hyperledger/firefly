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
	"crypto/rand"
	"database/sql/driver"
	"encoding/hex"
	"hash"
	"strings"

	"github.com/hyperledger/firefly/internal/i18n"
)

// Bytes32 is a holder of a hash, that can be used to correlate onchain data with off-chain data.
type Bytes32 [32]byte

func NewRandB32() *Bytes32 {
	var b Bytes32
	_, _ = rand.Read(b[0:32])
	return &b
}

func HashResult(hash hash.Hash) *Bytes32 {
	sum := hash.Sum(make([]byte, 0, 32))
	var b32 Bytes32
	copy(b32[:], sum)
	return &b32
}

func (b32 Bytes32) MarshalText() ([]byte, error) {
	hexstr := make([]byte, 64)
	hex.Encode(hexstr, b32[0:32])
	return hexstr, nil
}

func (b32 *Bytes32) UnmarshalText(b []byte) error {
	// We don't encourage the 0x prefix or use it internally, but we will strip it if supplied
	s := strings.TrimPrefix(string(b), "0x")
	_, err := hex.Decode(b32[0:32], []byte(s))
	return err
}

func ParseBytes32(ctx context.Context, hexStr string) (*Bytes32, error) {
	trimmed := []byte(strings.TrimPrefix(hexStr, "0x"))
	if len(trimmed) != 64 {
		return nil, i18n.NewError(context.Background(), i18n.MsgInvalidWrongLenB32)
	}
	var b32 Bytes32
	err := b32.UnmarshalText(trimmed)
	if err != nil {
		return nil, i18n.WrapError(context.Background(), err, i18n.MsgInvalidHex)
	}
	return &b32, nil
}

// Scan implements sql.Scanner
func (b32 *Bytes32) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil

	case string:
		if src == "" {
			return nil
		}
		return b32.UnmarshalText([]byte(src))

	case []byte:
		if len(src) == 0 {
			return nil
		}
		if len(src) != 32 {
			return b32.UnmarshalText(src)
		}
		copy((*b32)[:], src)
		return nil

	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, b32)
	}

}

// Value implements sql.Valuer
func (b32 *Bytes32) Value() (driver.Value, error) {
	if b32 == nil {
		return nil, nil
	}
	return b32.String(), nil
}

func (b32 *Bytes32) String() string {
	if b32 == nil {
		return ""
	}
	return hex.EncodeToString(b32[0:32])
}

// HexUUID is 32 character ASCII string containing the hex representation of UUID, with the dashes of the canonical representation removed
type HexUUID = Bytes32

// UUIDBytes returns the bytes of a UUID as a compressed hex string
func UUIDBytes(u *UUID) *Bytes32 {
	var d Bytes32
	copy(d[:], u[:])
	return &d
}

func (b32 *Bytes32) Equals(b2 *Bytes32) bool {
	switch {
	case b32 == nil && b2 == nil:
		return true
	case b32 == nil || b2 == nil:
		return false
	default:
		return *b32 == *b2
	}
}
