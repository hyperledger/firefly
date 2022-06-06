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

type fakePlugin struct{}

func (f *fakePlugin) Name() string { return "fake" }

func TestNewPendingMessageOp(t *testing.T) {

	txID := fftypes.NewUUID()
	op := NewOperation(&fakePlugin{}, "ns1", txID, OpTypeSharedStorageUploadBatch)
	assert.Equal(t, Operation{
		ID:          op.ID,
		Namespace:   "ns1",
		Transaction: txID,
		Plugin:      "fake",
		Type:        OpTypeSharedStorageUploadBatch,
		Status:      OpStatusPending,
		Created:     op.Created,
		Updated:     op.Created,
	}, *op)
}

func TestParseNamespacedOpID(t *testing.T) {

	ctx := context.Background()
	u := fftypes.NewUUID()

	_, _, err := ParseNamespacedOpID(ctx, "")
	assert.Regexp(t, "FF10411", err)

	_, _, err = ParseNamespacedOpID(ctx, "a::"+u.String())
	assert.Regexp(t, "FF10411", err)

	_, _, err = ParseNamespacedOpID(ctx, "bad%namespace:"+u.String())
	assert.Regexp(t, "FF00140", err)

	_, _, err = ParseNamespacedOpID(ctx, "ns1:Bad UUID")
	assert.Regexp(t, "FF00138", err)

	po := &PreparedOperation{
		ID:        u,
		Namespace: "ns1",
	}
	ns, u1, err := ParseNamespacedOpID(ctx, po.NamespacedIDString())
	assert.NoError(t, err)
	assert.Equal(t, u, u1)
	assert.Equal(t, "ns1", ns)

}
