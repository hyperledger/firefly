// Copyright © 2022 Kaleido, Inc.
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

//go:build reference
// +build reference

package reference

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateMarkdownPages(t *testing.T) {
	markdownMap, err := GenerateObjectsReferenceMarkdown()
	assert.NoError(t, err)
	assert.NotNil(t, markdownMap)

	for pageName, markdown := range markdownMap {
		f, err := os.Create(filepath.Join("..", "..", "docs", "reference", "types", fmt.Sprintf("%s.md", pageName)))
		assert.NoError(t, err)
		_, err = f.Write(markdown)
		assert.NoError(t, err)
		err = f.Close()
		assert.NoError(t, err)
	}
}
