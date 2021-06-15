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
	"database/sql/driver"
	"strings"
)

// SortableBool is a boolean, which is stored as a SMALLINT in databases where
// sorting/indexing by a BOOLEAN type is not consistently supported. TRUE>FALSE
type SortableBool bool

func (sb SortableBool) Value() (driver.Value, error) {
	if sb {
		return int64(1), nil
	}
	return int64(0), nil
}

func (sb *SortableBool) Scan(src interface{}) error {
	switch src := src.(type) {
	case int64:
		*sb = src > 0
		return nil
	case bool:
		*sb = SortableBool(src)
		return nil
	case string:
		*sb = SortableBool(strings.EqualFold("true", src))
		return nil
	default:
		*sb = false
		return nil
	}

}
