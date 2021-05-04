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

	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	var rand1, rand2, rand3 Bytes32
	rand1.UnmarshalText([]byte("3fcc7e07069e441f07c9f6b26f16fcb2dc896222d72888675082fd308440d9ae"))
	rand2.UnmarshalText([]byte("1d1462e02d7acee49a8448267c65067e0bec893c9a0c050b9835efa376fec046"))
	rand3.UnmarshalText([]byte("284b535da66aa0734af56c708426d756331baec3bce3079e508003bcf4738ee6"))
	d := DataRefSortable{
		{Hash: nil},
		{Hash: &rand1},
		{Hash: nil},
		{Hash: &rand2},
		{Hash: &rand3},
	}
	sort.Sort(d)
	assert.Nil(t, d[0].Hash)
	assert.Nil(t, d[1].Hash)
	assert.Equal(t, rand2.String(), d[2].Hash.String())
	assert.Equal(t, rand3.String(), d[3].Hash.String())
	assert.Equal(t, rand1.String(), d[4].Hash.String())
}
