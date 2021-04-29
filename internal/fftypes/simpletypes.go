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
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/aidarkhanov/nanoid"
	"github.com/google/uuid"
)

const (
	// ShortIDlphabet is designed for easy double-click select
	ShortIDlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"
)

func ShortID() string {
	return nanoid.Must(nanoid.Generate(ShortIDlphabet, 8))
}

// Bytes32 is a holder of a hash, that can be used to correlate onchain data with off-chain data.
type Bytes32 [32]byte

func NewRandB32() Bytes32 {
	var b [32]byte
	_, _ = rand.Read(b[0:32])
	return b
}

func (b32 *Bytes32) MarshalText() ([]byte, error) {
	hexstr := make([]byte, 64)
	hex.Encode(hexstr, b32[0:32])
	return hexstr, nil
}

func (b32 *Bytes32) String() string {
	return hex.EncodeToString(b32[0:32])
}

func (b32 *Bytes32) UnmarshalText(b []byte) error {
	// We don't encourage the 0x prefix or use it internally, but we will strip it if supplied
	s := strings.TrimPrefix(string(b), "0x")
	_, err := hex.Decode(b32[0:32], []byte(s))
	return err
}

// HexUUID is 32 character ASCII string containing the hex representation of UUID, with the dashes of the canonical representation removed
type HexUUID = Bytes32

// HexUUIDFromUUID returns the bytes of a UUID as a compressed hex string
func HexUUIDFromUUID(u uuid.UUID) HexUUID {
	var d HexUUID
	hex.Encode(d[0:32], u[0:16])
	return d
}
