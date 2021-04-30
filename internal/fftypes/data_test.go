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

package fftypes

import (
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	uuid1 := uuid.MustParse("94968e08-0a3f-4926-b917-586425b454db")
	uuid2 := uuid.MustParse("f8469593-5b8c-46a2-9e83-70d5d8b9bb50")
	uuid3 := uuid.MustParse("6d2f68e4-998e-478d-89ee-89470e5e6924")
	d := DataRefSortable{
		{ID: nil},
		{ID: &uuid1},
		{ID: nil},
		{ID: &uuid2},
		{ID: &uuid3},
	}
	sort.Sort(d)
	assert.Nil(t, d[0].ID)
	assert.Nil(t, d[1].ID)
	assert.Equal(t, uuid3.String(), d[2].ID.String())
	assert.Equal(t, uuid1.String(), d[3].ID.String())
	assert.Equal(t, uuid2.String(), d[4].ID.String())
}
