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

package blockchain

import (
	"encoding/hex"

	"github.com/google/uuid"
)

// Bytes32 is a holder of a hash, that can be used to correlate onchain data with off-chain data.
type Bytes32 = [32]byte

// HexUUID is 32 character ASCII string containing the hex representation of UUID, with the dashes of the canonical representation removed
type HexUUID = [32]byte

// HexUUIDFromUUID returns the bytes of a UUID as a compressed hex string
func HexUUIDFromUUID(u uuid.UUID) HexUUID {
	var d HexUUID
	hex.Encode(d[0:32], u[0:16])
	return d
}
