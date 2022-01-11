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

func TestNodeValidation(t *testing.T) {

	n := &Node{
		Name: "!name",
	}
	assert.Regexp(t, "FF10131.*name", n.Validate(context.Background(), false))

	n = &Node{
		Name:        "ok",
		Description: string(make([]byte, 4097)),
	}
	assert.Regexp(t, "FF10188.*description", n.Validate(context.Background(), false))

	n = &Node{
		Name:        "ok",
		Description: "ok",
	}
	assert.Regexp(t, "FF10211", n.Validate(context.Background(), false))

	n.Owner = "0x12345"
	assert.NoError(t, n.Validate(context.Background(), false))

	assert.Regexp(t, "FF10203", n.Validate(context.Background(), true))

	var def Definition = n
	assert.Equal(t, "ff_organizations", def.Topic())
	def.SetBroadcastMessage(NewUUID())
	assert.NotNil(t, n.Message)
}
