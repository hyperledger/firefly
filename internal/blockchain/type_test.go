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
	"testing"

	"github.com/google/uuid"
	"github.com/likexian/gokit/assert"
)

func TestHexUUIDFromUUID(t *testing.T) {
	u := uuid.Must(uuid.NewRandom())
	b := HexUUIDFromUUID(u)
	var dec [16]byte
	hex.Decode(dec[0:16], b[0:32])
	assert.Equal(t, dec[0:16], u[0:16])
	assert.Equal(t, u.String(), uuid.UUID(dec).String())
}
