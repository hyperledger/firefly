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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupValidation(t *testing.T) {

	group := &Group{
		Namespace: "!wrong",
	}
	assert.Regexp(t, "FF10131.*namespace", group.Validate(context.Background(), false))

	group = &Group{
		Namespace:   "ok",
		Description: string(make([]byte, 4097)),
	}
	assert.Regexp(t, "FF10188.*description", group.Validate(context.Background(), false))

	group = &Group{
		Namespace:   "ok",
		Description: "ok",
	}
	assert.Regexp(t, "FF10219.*recipient", group.Validate(context.Background(), false))

	group = &Group{
		Namespace:   "ok",
		Description: "ok",
		Recipients: Recipients{
			{Identity: "0x12345"},
		},
	}
	assert.NoError(t, group.Validate(context.Background(), false))

	assert.Regexp(t, "FF10203", group.Validate(context.Background(), true))

	group = &Group{
		Namespace:   "ok",
		Description: "ok",
		Recipients: Recipients{
			{ /* blank */ },
		},
	}
	assert.Regexp(t, "FF10220", group.Validate(context.Background(), false))

	group = &Group{
		Namespace:   "ok",
		Description: "ok",
		Recipients: Recipients{
			{Identity: "0x12345"},
			{Identity: "0x12345"},
		},
	}
	assert.Regexp(t, "FF10221", group.Validate(context.Background(), false))

	var def Definition = group
	assert.Equal(t, fmt.Sprintf("ff-grp-%s", group.ID), def.Context())
	def.SetBroadcastMessage(NewUUID())
	assert.NotNil(t, group.Message)
}
