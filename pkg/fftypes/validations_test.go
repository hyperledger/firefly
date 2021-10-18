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

func TestValidateFFNameField(t *testing.T) {

	err := ValidateFFNameField(context.Background(), "_badstart", "badField")
	assert.Regexp(t, "FF10131.*badField", err)

	err = ValidateFFNameField(context.Background(), "badend_", "badField")
	assert.Regexp(t, "FF10131.*badField", err)

	err = ValidateFFNameField(context.Background(), "0123456789_123456789-123456789.123456789-123456789_1234567890123", "badField")
	assert.NoError(t, err)

	err = ValidateFFNameField(context.Background(), "0123456789_123456789-123456789.123456789-123456789_12345678901234", "badField")
	assert.Regexp(t, "FF10131.*badField", err)

	err = ValidateFFNameField(context.Background(), "af34658e-a728-4b21-b9cf-8451f07be065", "badField")
	assert.Regexp(t, "FF10288.*badField", err)

}

func TestValidateLength(t *testing.T) {

	err := ValidateLength(context.Background(), "long string", "test", 5)
	assert.Regexp(t, "FF10188.*test", err)

	err = ValidateLength(context.Background(), "short string", "test", 50)
	assert.NoError(t, err)

}
