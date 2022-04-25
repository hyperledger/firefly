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

package i18n

var TimeDurationType = "[`time.Duration`](https://pkg.go.dev/time#Duration)"
var TimeFormatType = "[Time format](https://pkg.go.dev/time#pkg-constants) `string`"
var ByteSizeType = "[`BytesSize`](https://pkg.go.dev/github.com/docker/go-units#BytesSize)"
var GoTemplateType = "[Go Template](https://pkg.go.dev/text/template) `string`"
var StringType = "`string`"
var IntType = "`int`"
var BooleanType = "`boolean`"
var FloatType = "`boolean`"
var MapStringStringType = "`map[string]string`"
var IgnoredType = "IGNORE"

//revive:disable
var (
	ConfigLang                  = FFC("config.lang", "Default language for translation (API calls may support language override using headers)", StringType)
	ConfigLogCompress           = FFC("config.log.compress", "Determines if the rotated log files should be compressed using gzip", BooleanType)
	ConfigLogFilename           = FFC("config.log.filename", "Filename is the file to write logs to.  Backup log files will be retained in the same directory", StringType)
	ConfigLogFilesize           = FFC("config.log.filesize", "MaxSize is the maximum size the log file before it gets rotated", ByteSizeType)
	ConfigLogForceColor         = FFC("config.log.forceColor", "Force color to be enabled, even when a non-TTY output is detected", BooleanType)
	ConfigLogLevel              = FFC("config.log.level", "The log level - error, warn, info, debug, trace", StringType)
	ConfigLogMaxAge             = FFC("config.log.maxAge", "The maximum time to retain old log files based on the timestamp encoded in their filename.", TimeDurationType)
	ConfigLogMaxBackups         = FFC("config.log.maxBackups", "Maximum number of old log files to retain", IntType)
	ConfigLogNoColor            = FFC("config.log.noColor", "Force color to be disabled, event when TTY output is detected", BooleanType)
	ConfigLogTimeFormat         = FFC("config.log.timeFormat", "Custom time format for logs", TimeFormatType)
	ConfigLogUtc                = FFC("config.log.utc", "Use UTC timestamps for logs", BooleanType)
	ConfigLogIncludeCodeInfo    = FFC("config.log.includeCodeInfo", "Enables the report caller for including the calling file and line number, and the calling function. If using text logs, it uses the logrus text format rather than the default prefix format.", BooleanType)
	ConfigLogJSONEnabled        = FFC("config.log.json.enabled", "Enables JSON formatted logs rather than text. All log color settings are ignored when enabled.", BooleanType)
	ConfigLogJSONTimestampField = FFC("config.log.json.fields.timestamp", "Configures the JSON key containing the timestamp of the log", StringType)
	ConfigLogJSONLevelField     = FFC("config.log.json.fields.level", "Configures the JSON key containing the log level", StringType)
	ConfigLogJSONMessageField   = FFC("config.log.json.fields.message", "Configures the JSON key containing the log message", StringType)
	ConfigLogJSONFuncField      = FFC("config.log.json.fields.func", "Configures the JSON key containing the calling function", StringType)
	ConfigLogJSONFileField      = FFC("config.log.json.fields.file", "configures the JSON key containing the calling file", StringType)
)
