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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatatypeReference(t *testing.T) {

	var dr *DatatypeRef
	assert.Equal(t, "null", dr.String())
	dr = &DatatypeRef{
		Name:    "customer",
		Version: "0.0.1",
	}
	assert.Equal(t, "customer/0.0.1", dr.String())

}

func TestSealNoData(t *testing.T) {
	d := &Data{}
	err := d.Seal(context.Background())
	assert.Regexp(t, "FF10199", err.Error())
}

func TestSealOK(t *testing.T) {
	d := &Data{
		Value: []byte("{}"),
	}
	err := d.Seal(context.Background())
	assert.NoError(t, err)
}
