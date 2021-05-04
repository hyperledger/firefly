// Copyright © 2021 Kaleido, Inc.
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

package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTransactionByIdBadId(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.GetTransactionById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetMessageByIdBadId(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.GetMessageById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetBatchByIdBadId(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.GetBatchById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}

func TestGetDataByIdBadId(t *testing.T) {
	e := NewEngine().(*engine)
	_, err := e.GetDataById(context.Background(), "", "")
	assert.Regexp(t, "FF10142", err.Error())
}
