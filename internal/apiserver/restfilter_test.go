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

package apiserver

import (
	"net/http/httptest"
	"testing"

	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
)

func TestBuildFilter(t *testing.T) {
	req := httptest.NewRequest("GET", "/things?created=0&confirmed=!0&ID=>abc&Id=<abc&id=<=abc&id=>=abc&id=@abc&id=^abc&id=!@abc&id=!^abc&skip=10&limit=50&sort=id,sequence&descending", nil)
	filter := buildFilter(req, database.MessageQueryFactory)
	fi, err := filter.Finalize()
	assert.NoError(t, err)

	assert.Equal(t, "( confirmed != 0 ) && ( created == 0 ) && ( ( id %! 'abc' ) || ( id ^! 'abc' ) || ( id <= 'abc' ) || ( id < 'abc' ) || ( id >= 'abc' ) || ( id > 'abc' ) || ( id %= 'abc' ) || ( id ^= 'abc' ) ) sort=id,sequence descending skip=10 limit=50", fi.String())
}
