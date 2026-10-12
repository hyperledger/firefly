// Copyright © 2026 Kaleido, Inc.
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

package contracts

import (
	"context"
	"testing"

	"github.com/hyperledger-firefly/common/pkg/fftypes"
	"github.com/hyperledger-firefly/firefly/mocks/blockchainmocks"
	"github.com/hyperledger-firefly/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func errCacheTestMethod() *fftypes.FFIMethod {
	return &fftypes.FFIMethod{
		Name: "initiateTransfer",
		Params: []*fftypes.FFIParam{
			{Name: "to", Schema: fftypes.JSONAnyPtr(`{"type": "string", "details": {"type": "address"}}`)},
			{Name: "amount", Schema: fftypes.JSONAnyPtr(`{"type": "integer", "details": {"type": "uint256"}}`)},
		},
		Returns: []*fftypes.FFIParam{
			{Name: "transferId", Schema: fftypes.JSONAnyPtr(`{"type": "string", "details": {"type": "bytes32"}}`)},
		},
	}
}

func zeroArgError(name string) *fftypes.FFIError {
	return &fftypes.FFIError{
		FFIErrorDefinition: fftypes.FFIErrorDefinition{
			Name:   name,
			Params: fftypes.FFIParams{},
		},
	}
}

func paramError() *fftypes.FFIError {
	return &fftypes.FFIError{
		FFIErrorDefinition: fftypes.FFIErrorDefinition{
			Name: "AccessControlUnauthorizedAccount",
			Params: fftypes.FFIParams{
				{Name: "account", Schema: fftypes.JSONAnyPtr(`{"type": "string", "details": {"type": "address"}}`)},
			},
		},
	}
}

// Two interfaces sharing a method signature but declaring different sets of
// zero-argument errors must each be parsed with their own error set, rather
// than sharing a method cache entry.
func TestMethodCacheKeyDistinguishesZeroArgErrors(t *testing.T) {
	cm := newTestContractManager()
	ctx := context.Background()

	// One parameter-bearing error, one zero-argument error
	errorsV100 := []*fftypes.FFIError{
		paramError(),
		zeroArgError("InvalidAmount"),
	}
	// The same two, plus a zero-argument error the first set lacks
	errorsV101 := []*fftypes.FFIError{
		paramError(),
		zeroArgError("InvalidAmount"),
		zeroArgError("AmountExceedsAvailableBalance"),
	}

	mbi := cm.blockchain.(*blockchainmocks.Plugin)
	var parsedWith [][]*fftypes.FFIError
	mbi.On("ParseInterface", ctx, mock.Anything, mock.Anything).
		Return("parsed", nil).
		Run(func(args mock.Arguments) {
			parsedWith = append(parsedWith, args[2].([]*fftypes.FFIError))
		})

	input := map[string]interface{}{"to": "0xabc", "amount": float64(1)}

	// The smaller error set is parsed first, populating the cache
	_, err := cm.validateInvokeContractRequest(ctx, &core.ContractCallRequest{
		Type:   core.CallTypeInvoke,
		Method: errCacheTestMethod(),
		Errors: errorsV100,
		Input:  input,
	}, false)
	assert.NoError(t, err)

	// The larger set must not be served the cached parse
	_, err = cm.validateInvokeContractRequest(ctx, &core.ContractCallRequest{
		Type:   core.CallTypeInvoke,
		Method: errCacheTestMethod(),
		Errors: errorsV101,
		Input:  input,
	}, false)
	assert.NoError(t, err)

	require.Len(t, parsedWith, 2, "different error sets must produce different cache keys")
	require.Len(t, parsedWith[1], 3, "the larger request must parse all three of its errors")

	names := make([]string, 0, len(parsedWith[1]))
	for _, e := range parsedWith[1] {
		names = append(names, e.Name)
	}
	assert.Contains(t, names, "AmountExceedsAvailableBalance", "zero-argument error was dropped")
}
