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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenAccountHash(t *testing.T) {
	poolID := MustParseUUID("39296b6e-91b9-4a61-b279-833c85b04d94")
	account := &TokenAccount{
		Identifier: TokenAccountIdentifier{
			Namespace:  "namespace",
			PoolID:     poolID,
			TokenIndex: "1",
			Identity:   "0x12345",
		},
	}
	assert.Equal(t, "1584669442bd1f5cea25b17d59df84c63dbeb6f15397faa3344fdf33326ddb62", account.Identifier.Hash().String())
}
