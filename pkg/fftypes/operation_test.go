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

type fakePlugin struct{}

func (f *fakePlugin) Name() string { return "fake" }

func TestNewPendingMessageOp(t *testing.T) {

	txID := NewUUID()
	op := NewOperation(&fakePlugin{}, "ns1", txID, OpTypePublicStorageBatchBroadcast)
	assert.Equal(t, Operation{
		ID:          op.ID,
		Namespace:   "ns1",
		Transaction: txID,
		Plugin:      "fake",
		Type:        OpTypePublicStorageBatchBroadcast,
		Status:      OpStatusPending,
		Created:     op.Created,
	}, *op)
}
