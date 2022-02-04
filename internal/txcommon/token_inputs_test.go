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

package txcommon

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestAddTokenPoolCreateInputs(t *testing.T) {
	op := &fftypes.Operation{}
	config := fftypes.JSONObject{
		"foo": "bar",
	}
	pool := &fftypes.TokenPool{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
		Name:      "testpool",
		Symbol:    "FFT",
		Config:    config,
	}

	AddTokenPoolCreateInputs(op, pool)
	assert.Equal(t, pool.ID.String(), op.Input.GetString("id"))
	assert.Equal(t, "ns1", op.Input.GetString("namespace"))
	assert.Equal(t, "testpool", op.Input.GetString("name"))
	assert.Equal(t, "FFT", op.Input.GetString("symbol"))
	assert.Equal(t, pool.Config, op.Input.GetObject("config"))
}

func TestRetrieveTokenPoolCreateInputs(t *testing.T) {
	id := fftypes.NewUUID()
	config := fftypes.JSONObject{
		"foo": "bar",
	}
	op := &fftypes.Operation{
		Input: fftypes.JSONObject{
			"id":        id.String(),
			"namespace": "ns1",
			"name":      "testpool",
			"symbol":    "FFT",
			"config":    config,
		},
	}
	pool := &fftypes.TokenPool{}

	err := RetrieveTokenPoolCreateInputs(context.Background(), op, pool)
	assert.NoError(t, err)
	assert.Equal(t, *id, *pool.ID)
	assert.Equal(t, "ns1", pool.Namespace)
	assert.Equal(t, "testpool", pool.Name)
	assert.Equal(t, "FFT", pool.Symbol)
	assert.Equal(t, config, pool.Config)
}

func TestRetrieveTokenPoolCreateInputsBadID(t *testing.T) {
	op := &fftypes.Operation{
		Input: fftypes.JSONObject{
			"id": "bad",
		},
	}
	pool := &fftypes.TokenPool{}

	err := RetrieveTokenPoolCreateInputs(context.Background(), op, pool)
	assert.Regexp(t, "FF10142", err)
}

func TestRetrieveTokenPoolCreateInputsNoName(t *testing.T) {
	op := &fftypes.Operation{
		Input: fftypes.JSONObject{
			"id":        fftypes.NewUUID().String(),
			"namespace": "ns1",
		},
	}
	pool := &fftypes.TokenPool{}

	err := RetrieveTokenPoolCreateInputs(context.Background(), op, pool)
	assert.Error(t, err)
}

func TestAddTokenTransferInputs(t *testing.T) {
	op := &fftypes.Operation{}
	transfer := &fftypes.TokenTransfer{
		LocalID: fftypes.NewUUID(),
		Type:    fftypes.TokenTransferTypeTransfer,
		Amount:  *fftypes.NewFFBigInt(1),
		TX: fftypes.TransactionRef{
			Type: fftypes.TransactionTypeTokenTransfer,
			ID:   fftypes.NewUUID(),
		},
	}

	AddTokenTransferInputs(op, transfer)
	assert.Equal(t, fftypes.JSONObject{
		"amount":  "1",
		"localId": transfer.LocalID.String(),
		"tx": map[string]interface{}{
			"id":   transfer.TX.ID.String(),
			"type": "token_transfer",
		},
		"type": "transfer",
	}, op.Input)
}

func TestRetrieveTokenTransferInputs(t *testing.T) {
	id := fftypes.NewUUID()
	op := &fftypes.Operation{
		Input: fftypes.JSONObject{
			"amount":  "1",
			"localId": id.String(),
		},
	}
	transfer := &fftypes.TokenTransfer{Amount: *fftypes.NewFFBigInt(2)}

	err := RetrieveTokenTransferInputs(context.Background(), op, transfer)
	assert.NoError(t, err)
	assert.Equal(t, *id, *transfer.LocalID)
	assert.Equal(t, int64(2), transfer.Amount.Int().Int64())
}

func TestRetrieveTokenTransferInputsBadID(t *testing.T) {
	op := &fftypes.Operation{
		Input: fftypes.JSONObject{
			"localId": "bad",
		},
	}
	transfer := &fftypes.TokenTransfer{}

	err := RetrieveTokenTransferInputs(context.Background(), op, transfer)
	assert.Regexp(t, "FF10151", err)
}

func TestRetrieveTokenTransferInputsMissingID(t *testing.T) {
	op := &fftypes.Operation{
		Input: fftypes.JSONObject{},
	}
	transfer := &fftypes.TokenTransfer{}

	err := RetrieveTokenTransferInputs(context.Background(), op, transfer)
	assert.Regexp(t, "FF10142", err)
}
