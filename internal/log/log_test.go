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

package log

import (
	"context"
	"testing"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/likexian/gokit/assert"
	"github.com/sirupsen/logrus"
)

func TestLogContext(t *testing.T) {
	ctx := WithLogField(context.Background(), "myfield", "myvalue")
	assert.Equal(t, "myvalue", L(ctx).Data["myfield"])
}

func TestSettingErrorLevel(t *testing.T) {
	config.Set(config.LogLevel, "eRrOr")
	SetupLogging(context.Background())
	assert.Equal(t, logrus.ErrorLevel, logrus.GetLevel())
}

func TestSettingDebugLevel(t *testing.T) {
	config.Set(config.LogLevel, "DEBUG")
	SetupLogging(context.Background())
	assert.Equal(t, logrus.DebugLevel, logrus.GetLevel())
}

func TestSettingTraceLevel(t *testing.T) {
	config.Set(config.LogLevel, "trace")
	SetupLogging(context.Background())
	assert.Equal(t, logrus.TraceLevel, logrus.GetLevel())
}

func TestSettingInfoLevel(t *testing.T) {
	config.Set(config.LogLevel, "info")
	SetupLogging(context.Background())
	assert.Equal(t, logrus.InfoLevel, logrus.GetLevel())
}

func TestSettingDefaultLevel(t *testing.T) {
	config.Set(config.LogLevel, "something else")
	SetupLogging(context.Background())
	assert.Equal(t, logrus.InfoLevel, logrus.GetLevel())
}
