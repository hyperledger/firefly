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

package apiserver

import (
	"net/http/httptest"
	"testing"

	"github.com/hyperledger/firefly/pkg/database"
	"github.com/stretchr/testify/assert"
)

func TestBuildFilterDescending(t *testing.T) {
	as := &apiServer{
		maxFilterLimit: 250,
	}

	req := httptest.NewRequest("GET", "/things?created=0&confirmed=!0&Tag=>abc&TAG=<abc&tag=<=abc&tag=>=abc&tag=@abc&tag=^abc&tag=!@abc&tag=!^abc&skip=10&limit=50&sort=tag,sequence&descending", nil)
	filter, err := as.buildFilter(req, database.MessageQueryFactory)
	assert.NoError(t, err)
	fi, err := filter.Finalize()
	assert.NoError(t, err)

	assert.Equal(t, "( confirmed != 0 ) && ( created == 0 ) && ( ( tag %! 'abc' ) || ( tag ^! 'abc' ) || ( tag <= 'abc' ) || ( tag < 'abc' ) || ( tag >= 'abc' ) || ( tag > 'abc' ) || ( tag %= 'abc' ) || ( tag ^= 'abc' ) ) sort=-tag,-sequence skip=10 limit=50", fi.String())
}

func TestBuildFilterAscending(t *testing.T) {
	as := &apiServer{
		maxFilterLimit: 250,
	}

	req := httptest.NewRequest("GET", "/things?created=0&sort=tag,sequence&ascending", nil)
	filter, err := as.buildFilter(req, database.MessageQueryFactory)
	assert.NoError(t, err)
	fi, err := filter.Finalize()
	assert.NoError(t, err)

	assert.Equal(t, "( created == 0 ) sort=tag,sequence", fi.String())
}

func TestBuildFilterLimitSkip(t *testing.T) {
	as := &apiServer{
		maxFilterSkip: 250,
	}

	req := httptest.NewRequest("GET", "/things?skip=251", nil)
	_, err := as.buildFilter(req, database.MessageQueryFactory)
	assert.Regexp(t, "FF10183.*250", err)
}

func TestBuildFilterLimitLimit(t *testing.T) {
	as := &apiServer{
		maxFilterLimit: 500,
	}

	req := httptest.NewRequest("GET", "/things?limit=501", nil)
	_, err := as.buildFilter(req, database.MessageQueryFactory)
	assert.Regexp(t, "FF10184.*500", err)
}
