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

package txcommon

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestAddTokenTransferInputs(t *testing.T) {
	op := &fftypes.Operation{}
	transfer := &fftypes.TokenTransfer{
		LocalID: fftypes.NewUUID(),
	}

	AddTokenTransferInputs(op, transfer)
	assert.Equal(t, transfer.LocalID.String(), op.Input.GetString("id"))
}

func TestRetrieveTokenTransferInputs(t *testing.T) {
	id := fftypes.NewUUID()
	op := &fftypes.Operation{
		Input: fftypes.JSONObject{
			"id": id.String(),
		},
	}
	transfer := &fftypes.TokenTransfer{}

	err := RetrieveTokenTransferInputs(context.Background(), op, transfer)
	assert.NoError(t, err)
	assert.Equal(t, *id, *transfer.LocalID)
}

func TestRetrieveTokenTransferInputsBadID(t *testing.T) {
	op := &fftypes.Operation{
		Input: fftypes.JSONObject{
			"id": "bad",
		},
	}
	transfer := &fftypes.TokenTransfer{}

	err := RetrieveTokenTransferInputs(context.Background(), op, transfer)
	assert.Regexp(t, "FF10142", err)
}
