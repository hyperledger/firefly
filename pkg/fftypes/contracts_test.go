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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateContractAPI(t *testing.T) {
	api := &ContractAPI{
		Namespace: "ns1",
		Name:      "banana",
	}
	err := api.Validate(context.Background(), false)
	assert.NoError(t, err)
}

func TestValidateInvalidContractAPI(t *testing.T) {
	api := &ContractAPI{
		Namespace: "&%&^#()#",
		Name:      "banana",
	}
	err := api.Validate(context.Background(), false)
	assert.Regexp(t, "FF10131", err)

	api = &ContractAPI{
		Namespace: "ns1",
		Name:      "(%&@!^%^)",
	}
	err = api.Validate(context.Background(), false)
	assert.Regexp(t, "FF10131", err)
}

func TestContractAPITopic(t *testing.T) {
	api := &ContractAPI{
		Namespace: "ns1",
	}
	assert.Equal(t, "4cccc66c1f0eebcf578f1e63b73a2047d4eb4c84c0a00c69b0e00c7490403d20", api.Topic())
}

func TestContractAPISetBroadCastMessage(t *testing.T) {
	msgID := NewUUID()
	api := &ContractAPI{}
	api.SetBroadcastMessage(msgID)
	assert.Equal(t, api.Message, msgID)
}

func TestLocationAndLedgerEquals(t *testing.T) {
	var c1 *ContractAPI = nil
	var c2 *ContractAPI = nil
	assert.False(t, c1.LocationAndLedgerEquals(c2))

	c1 = &ContractAPI{
		ID:       NewUUID(),
		Location: JSONAnyPtr("abc"),
		Ledger:   JSONAnyPtr("def"),
	}
	c2 = &ContractAPI{
		ID:       NewUUID(),
		Location: JSONAnyPtr("abc"),
		Ledger:   JSONAnyPtr("def"),
	}
	assert.True(t, c1.LocationAndLedgerEquals(c2))

	c1 = &ContractAPI{
		ID:       NewUUID(),
		Location: JSONAnyPtr("abc"),
		Ledger:   JSONAnyPtr("fff"),
	}
	c2 = &ContractAPI{
		ID:       NewUUID(),
		Location: JSONAnyPtr("abc"),
		Ledger:   JSONAnyPtr("def"),
	}
	assert.False(t, c1.LocationAndLedgerEquals(c2))

	c1 = &ContractAPI{
		ID:       NewUUID(),
		Location: JSONAnyPtr("fff"),
		Ledger:   JSONAnyPtr("def"),
	}
	c2 = &ContractAPI{
		ID:       NewUUID(),
		Location: JSONAnyPtr("abc"),
		Ledger:   JSONAnyPtr("def"),
	}
	assert.False(t, c1.LocationAndLedgerEquals(c2))

	c1 = &ContractAPI{
		ID:       NewUUID(),
		Location: nil,
		Ledger:   nil,
	}
	c2 = &ContractAPI{
		ID:       NewUUID(),
		Location: nil,
		Ledger:   nil,
	}
	assert.True(t, c1.LocationAndLedgerEquals(c2))
}
