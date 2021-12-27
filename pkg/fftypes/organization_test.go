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

func TestOrganizationValidation(t *testing.T) {

	org := &Organization{
		Name: "!name",
	}
	assert.Regexp(t, "FF10131.*name", org.Validate(context.Background(), false))

	org = &Organization{
		Name:        "ok",
		Description: string(make([]byte, 4097)),
	}
	assert.Regexp(t, "FF10188.*description", org.Validate(context.Background(), false))

	org = &Organization{
		Name:        "ok",
		Description: "ok",
		Identity:    "ok",
	}
	assert.NoError(t, org.Validate(context.Background(), false))

	assert.Regexp(t, "FF10203", org.Validate(context.Background(), true))

	var def Definition = org
	assert.Equal(t, "ff_organizations", def.Topic())
	def.SetBroadcastMessage(NewUUID())
	assert.NotNil(t, org.Message)
}

func TestGetDID(t *testing.T) {

	var org *Organization
	assert.Equal(t, "", org.GetDID())

	org = &Organization{
		ID: NewUUID(),
	}
	assert.Equal(t, fmt.Sprintf("did:firefly:org/%s", org.ID), org.GetDID())

}
