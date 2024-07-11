// Copyright © 2024 Kaleido, Inc.
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

package fabric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSubscriptionBadBlock(t *testing.T) {
	e, cancel := newTestFabric()
	defer cancel()

	_, err := e.streams.createSubscription(context.Background(), nil, "", "", "", "wrongness", "")
	assert.Regexp(t, "FF10473", err)
}

func TestResolveFromBlockCombinations(t *testing.T) {

	ctx := context.Background()

	fromBlock, err := resolveFromBlock(ctx, "", "")
	assert.Equal(t, "newest", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "latest", "")
	assert.Equal(t, "newest", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "newest", "")
	assert.Equal(t, "newest", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "0", "")
	assert.Equal(t, "0", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "0", "000000000010/4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d")
	assert.Equal(t, "9", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "20", "000000000010/4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d")
	assert.Equal(t, "20", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "", "000000000010/4763a0c50e3bba7cef1a7ba35dd3f9f3426bb04d0156f326e84ec99387c4746d")
	assert.Equal(t, "9", fromBlock)
	assert.NoError(t, err)

	_, err = resolveFromBlock(ctx, "", "wrong")
	assert.Regexp(t, "FF10472", err)

}
