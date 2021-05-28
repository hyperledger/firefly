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

package i18n

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
)

func TestExpand(t *testing.T) {
	lang := language.Make("en")
	ctx := WithLang(context.Background(), lang)
	str := Expand(ctx, MsgWebsocketClientError, "myinsert")
	assert.Equal(t, "Error received from WebSocket client: myinsert", str)
}

func TestExpandWithCode(t *testing.T) {
	lang := language.Make("en")
	ctx := WithLang(context.Background(), lang)
	str := ExpandWithCode(ctx, MsgWebsocketClientError, "myinsert")
	assert.Equal(t, "FF10108: Error received from WebSocket client: myinsert", str)
}

func TestGetStatusHint(t *testing.T) {
	code, ok := GetStatusHint(string(MsgResponseMarshalError))
	assert.True(t, ok)
	assert.Equal(t, 400, code)
}

func TestDuplicateKey(t *testing.T) {
	ffm("ABCD1234", "test1")
	assert.Panics(t, func() {
		ffm("ABCD1234", "test2")
	})
}
