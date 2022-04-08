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

//revive:disable
var (
	ConfigLang          = FFM("config.lang", "Default language for translation (API calls may support language override using headers)")
	ConfigLogCompress   = FFM("config.log.compress", "Determines if the rotated log files should be compressed using gzip")
	ConfigLogFilename   = FFM("config.log.filename", "Filename is the file to write logs to.  Backup log files will be retained in the same directory")
	ConfigLogFilesize   = FFM("config.log.filesize", "MaxSize is the maximum size the log file before it gets rotated")
	ConfigLogForceColor = FFM("config.log.forceColor", "Force color to be enabled, even when a non-TTY output is detected")
	ConfigLogLevel      = FFM("config.log.level", "The log level - error, warn, info, debug, trace")
	ConfigLogMaxAge     = FFM("config.log.maxAge", "The maximum time to retain old log files based on the timestamp encoded in their filename.")
	ConfigLogMaxBackups = FFM("config.log.maxBackups", "Maximum number of old log files to retain")
	ConfigLogNoColor    = FFM("config.log.noColor", "Force color to be disabled, event when TTY output is detected")
	ConfigLogTimeFormat = FFM("config.log.timeFormat", "Custom time format, using the rules described here: https://pkg.go.dev/time#pkg-constants")
	ConfigLogUtc        = FFM("config.log.utc", "Use UTC timestamps for logs")
)
