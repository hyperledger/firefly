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

package log

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLogContext(t *testing.T) {
	ctx := WithLogField(context.Background(), "myfield", "myvalue")
	assert.Equal(t, "myvalue", L(ctx).Data["myfield"])
}

func TestLogContextLimited(t *testing.T) {
	ctx := WithLogField(context.Background(), "myfield", "0123456789012345678901234567890123456789012345678901234567890123456789")
	assert.Equal(t, "0123456789012345678901234567890123456789012345678901234567890...", L(ctx).Data["myfield"])
}

func TestSettingErrorLevel(t *testing.T) {
	SetLevel("eRrOr")
	assert.Equal(t, logrus.ErrorLevel, logrus.GetLevel())
}

func TestSettingDebugLevel(t *testing.T) {
	SetLevel("DEBUG")
	assert.Equal(t, logrus.DebugLevel, logrus.GetLevel())
}

func TestSettingTraceLevel(t *testing.T) {
	SetLevel("trace")
	assert.Equal(t, logrus.TraceLevel, logrus.GetLevel())
}

func TestSettingInfoLevel(t *testing.T) {
	SetLevel("info")
	assert.Equal(t, logrus.InfoLevel, logrus.GetLevel())
}

func TestSettingDefaultLevel(t *testing.T) {
	SetLevel("something else")
	assert.Equal(t, logrus.InfoLevel, logrus.GetLevel())
}

func TestSetFormatting(t *testing.T) {
	SetFormatting(Formatting{
		DisableColor: true,
		UTC:          true,
	})
	L(context.Background()).Infof("time in UTC")
}
