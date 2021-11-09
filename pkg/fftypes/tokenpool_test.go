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

func TestTokenPoolValidation(t *testing.T) {
	pool := &TokenPool{
		Namespace: "!wrong",
		Name:      "ok",
	}
	err := pool.Validate(context.Background(), false)
	assert.Regexp(t, "FF10131.*'namespace'", err)

	pool = &TokenPool{
		Namespace: "ok",
		Name:      "!wrong",
	}
	err = pool.Validate(context.Background(), false)
	assert.Regexp(t, "FF10131.*'name'", err)

	pool = &TokenPool{
		Namespace: "ok",
		Name:      "ok",
	}
	err = pool.Validate(context.Background(), false)
	assert.NoError(t, err)
}

func TestTokenPoolDefinition(t *testing.T) {
	pool := &TokenPool{
		Namespace: "ok",
		Name:      "ok",
	}
	var def Definition = &TokenPoolAnnouncement{Pool: pool}
	assert.Equal(t, "ff_ns_ok", def.Topic())

	id := NewUUID()
	def.SetBroadcastMessage(id)
	assert.Equal(t, id, pool.Message)
}
