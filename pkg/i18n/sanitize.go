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

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var sanitize = bluemonday.StrictPolicy()

func SanitizeLimit(s string, limit int) string {
	if len(s) >= limit {
		if limit > 16 {
			s = s[0:limit-3] + "..."
		} else {
			s = s[0:limit]
		}
	}
	s = strings.ReplaceAll(sanitize.Sanitize(s), "&#39;", "'")
	return s
}
