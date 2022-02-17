// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.identity/licenses/LICENSE-2.0
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

func TestIdentityValidation(t *testing.T) {

	identity := &Identity{
		Name: "!name",
	}
	assert.Regexp(t, "FF10131.*name", identity.Validate(context.Background(), false))

	identity = &Identity{
		Name:        "ok",
		Description: string(make([]byte, 4097)),
	}
	assert.Regexp(t, "FF10188.*description", identity.Validate(context.Background(), false))

	identity = &Identity{
		Name:        "ok",
		Description: "ok",
	}
	assert.NoError(t, identity.Validate(context.Background(), false))

	assert.Regexp(t, "FF10203", identity.Validate(context.Background(), true))

	var def Definition = identity
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", def.Topic())
	def.SetBroadcastMessage(NewUUID())
	assert.NotNil(t, identity.Message)
}
