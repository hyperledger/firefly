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

type FFEnum string

var enumValues = map[string][]interface{}{}

func ffEnum(t string, val string) FFEnum {
	enumValues[t] = append(enumValues[t], val)
	return FFEnum(val)
}

func FFEnumValues(t string) []interface{} {
	return enumValues[t]
}

func (ts FFEnum) String() string {
	return strings.ToLower(string(ts))
}

func (ts FFEnum) Lower() FFEnum {
	return FFEnum(strings.ToLower(string(ts)))
}

func (ts FFEnum) Equals(ts2 FFEnum) bool {
	return strings.EqualFold(string(ts), string(ts2))
}

func (ts FFEnum) Value() (driver.Value, error) {
	return ts.String(), nil
}

func (ts *FFEnum) UnmarshalText(b []byte) error {
	*ts = FFEnum(strings.ToLower(string(b)))
	return nil
}
