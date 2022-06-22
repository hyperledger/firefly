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

package core

import (
	"context"
	"testing"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

func TestNamespaceValidation(t *testing.T) {

	ns := &Namespace{
		Name: "!wrong",
	}
	assert.Regexp(t, "FF00140.*name", ns.Validate(context.Background(), false))

	ns = &Namespace{
		Name:        "ok",
		Description: string(make([]byte, 4097)),
	}
	assert.Regexp(t, "FF00135.*description", ns.Validate(context.Background(), false))

	ns = &Namespace{
		Name:        "ok",
		Description: "ok",
	}
	assert.NoError(t, ns.Validate(context.Background(), false))

	assert.Regexp(t, "FF00114", ns.Validate(context.Background(), true))

	var nsDef Definition = ns
	assert.Equal(t, "358de1708c312f6b9eb4c44e0d9811c6f69bf389871d38dd7501992b2c00b557", nsDef.Topic())
	nsDef.SetBroadcastMessage(fftypes.NewUUID())
	assert.NotNil(t, ns.Message)

}

func TestFireFlyContractsDatabaseSerialization(t *testing.T) {
	contracts1 := &FireFlyContracts{
		Active: FireFlyContractInfo{
			Index:        1,
			FirstEvent:   "oldest",
			Subscription: "1234",
			Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
				"address": "0x123",
			}.String()),
		},
		Terminated: []FireFlyContractInfo{
			{
				Index:        0,
				FinalEvent:   "50",
				Subscription: "12345",
				FirstEvent:   "oldest",
				Location: fftypes.JSONAnyPtr(fftypes.JSONObject{
					"address": "0x1234",
				}.String()),
			},
		},
	}

	// Verify it serializes as bytes to the database
	val1, err := contracts1.Value()
	assert.NoError(t, err)
	assert.Equal(t, `{"active":{"index":1,"location":{"address":"0x123"},"firstEvent":"oldest","subscription":"1234"},"terminated":[{"index":0,"finalEvent":"50","location":{"address":"0x1234"},"firstEvent":"oldest","subscription":"12345"}]}`, string(val1.([]byte)))

	// Verify it restores ok
	contracts2 := &FireFlyContracts{}
	err = contracts2.Scan(val1)
	assert.NoError(t, err)
	assert.Equal(t, 1, contracts2.Active.Index)
	assert.Equal(t, *fftypes.JSONAnyPtr(fftypes.JSONObject{
		"address": "0x123",
	}.String()), *contracts2.Active.Location)
	assert.Len(t, contracts2.Terminated, 1)

	// Verify it ignores a blank string
	err = contracts2.Scan("")
	assert.NoError(t, err)
	assert.Equal(t, 1, contracts2.Active.Index)

	// Out of luck with anything else
	err = contracts2.Scan(false)
	assert.Regexp(t, "FF00105", err)
}
