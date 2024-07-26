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

package ethereum

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSubscriptionBadBlock(t *testing.T) {
	e, cancel := newTestEthereum()
	defer cancel()

	_, err := e.streams.createSubscription(context.Background(), "", "", "wrongness", nil, nil, []*filter{}, "")
	assert.Regexp(t, "FF10473", err)
}

func TestResolveFromBlockCombinations(t *testing.T) {

	ctx := context.Background()

	fromBlock, err := resolveFromBlock(ctx, "", "")
	assert.Equal(t, "latest", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "latest", "")
	assert.Equal(t, "latest", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "newest", "")
	assert.Equal(t, "latest", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "0", "")
	assert.Equal(t, "0", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "0", "000000000010/000000/000050")
	assert.Equal(t, "9", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "20", "000000000010/000000/000050")
	assert.Equal(t, "20", fromBlock)
	assert.NoError(t, err)

	fromBlock, err = resolveFromBlock(ctx, "", "000000000010/000000/000050")
	assert.Equal(t, "9", fromBlock)
	assert.NoError(t, err)

	_, err = resolveFromBlock(ctx, "", "wrong")
	assert.Regexp(t, "FF10472", err)

}
