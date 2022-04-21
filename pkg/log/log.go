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

package log

import (
	"context"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	rootLogger = logrus.NewEntry(logrus.StandardLogger())

	// L accesses the current logger from the context
	L = loggerFromContext
)

type (
	ctxLogKey struct{}
)

// WithLogger adds the specified logger to the context
func WithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, ctxLogKey{}, logger)
}

// WithLogField adds the specified field to the logger in the context
func WithLogField(ctx context.Context, key, value string) context.Context {
	if len(value) > 61 {
		value = value[0:61] + "..."
	}
	return WithLogger(ctx, loggerFromContext(ctx).WithField(key, value))
}

// LoggerFromContext returns the logger for the current context, or no logger if there is no context
func loggerFromContext(ctx context.Context) *logrus.Entry {
	logger := ctx.Value(ctxLogKey{})
	if logger == nil {
		return rootLogger
	}
	return logger.(*logrus.Entry)
}

func SetLevel(level string) {
	switch strings.ToLower(level) {
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}

type Formatting struct {
	DisableColor       bool
	ForceColor         bool
	TimestampFormat    string
	UTC                bool
	IncludeCodeInfo    bool
	JSONEnabled        bool
	JSONTimestampField string
	JSONLevelField     string
	JSONMessageField   string
	JSONFuncField      string
	JSONFileField      string
}

type utcFormat struct {
	f logrus.Formatter
}

func (utc *utcFormat) Format(e *logrus.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return utc.f.Format(e)
}

func SetFormatting(format Formatting) {
	var formatter logrus.Formatter

	switch {
	case format.JSONEnabled:
		formatter = &logrus.JSONFormatter{
			TimestampFormat: format.TimestampFormat,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  format.JSONTimestampField,
				logrus.FieldKeyLevel: format.JSONLevelField,
				logrus.FieldKeyMsg:   format.JSONMessageField,
				logrus.FieldKeyFunc:  format.JSONFuncField,
				logrus.FieldKeyFile:  format.JSONFileField,
			},
		}
		break
	case format.IncludeCodeInfo:
		formatter = &logrus.TextFormatter{
			DisableColors:   format.DisableColor,
			ForceColors:     format.ForceColor,
			TimestampFormat: format.TimestampFormat,
			DisableSorting:  false,
			FullTimestamp:   true,
		}
		break
	default:
		formatter = &prefixed.TextFormatter{
			DisableColors:   format.DisableColor,
			ForceColors:     format.ForceColor,
			TimestampFormat: format.TimestampFormat,
			DisableSorting:  false,
			ForceFormatting: true,
			FullTimestamp:   true,
		}
	}

	if format.IncludeCodeInfo {
		logrus.SetReportCaller(true)
	}

	if format.UTC {
		formatter = &utcFormat{f: formatter}
	}

	logrus.SetFormatter(formatter)
}
