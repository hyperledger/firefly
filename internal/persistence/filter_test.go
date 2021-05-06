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

package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildMessageFilter(t *testing.T) {
	fb := MessageFilterBuilder.New(context.Background())
	f, err := fb.And(
		fb.Eq("namespace", "ns1"),
		fb.Or(
			fb.Eq("id", "35c11cba-adff-4a4d-970a-02e3a0858dc8"),
			fb.Eq("id", "caefb9d1-9fc9-4d6a-a155-514d3139adf7"),
		),
		fb.Gt("created", "12345"),
	).Finalize()

	assert.NoError(t, err)
	assert.Equal(t, "( namespace == 'ns1' ) && ( ( id == '35c11cba-adff-4a4d-970a-02e3a0858dc8' ) || ( id == 'caefb9d1-9fc9-4d6a-a155-514d3139adf7' ) ) && ( created > 12345 )", f.String())
}
