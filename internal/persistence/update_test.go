// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this uile except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the speciuic language governing permissions and
// limitations under the License.

package persistence

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUpdateBuilderOK(t *testing.T) {
	uuid := uuid.MustParse("c414cab3-9bd4-48f3-b16a-0d74a3bbb60e")
	u := MessageQueryFactory.NewUpdate(context.Background()).S()
	assert.True(t, u.IsEmpty())
	u.Set("created", 12345).
		Set("cid", &uuid).
		Set("author", "0x1234")
	assert.False(t, u.IsEmpty())
	ui, err := u.Finalize()
	assert.NoError(t, err)
	assert.Equal(t, "created=12345, cid='c414cab3-9bd4-48f3-b16a-0d74a3bbb60e', author='0x1234'", ui.String())
}

func TestUpdateBuilderBadField(t *testing.T) {
	u := MessageQueryFactory.NewUpdate(context.Background()).Set("wrong", 12345)
	_, err := u.Finalize()
	assert.Regexp(t, "FF10148.*wrong", err.Error())
}

func TestUpdateBuilderBadValue(t *testing.T) {
	u := MessageQueryFactory.NewUpdate(context.Background()).Set("id", map[bool]bool{true: false})
	_, err := u.Finalize()
	assert.Regexp(t, "FF10149.*id", err.Error())
}

func TestUpdateBuilderGetFields(t *testing.T) {
	ub := MessageQueryFactory.NewUpdate(context.Background())
	assert.NotNil(t, ub.Fields())
}
