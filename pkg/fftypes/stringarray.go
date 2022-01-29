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

package fftypes

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sort"
	"strings"

	"github.com/hyperledger/firefly/internal/i18n"
)

// FFStringArray is an array of strings, each conforming to the requirements of a FireFly name
type FFStringArray []string

// Because each FFName has a max length of 64, 15 names (plus comma delimeters) is a safe max
// to pack into a string column of length 1024
const FFStringNameItemsMax = 15

// FFStringArrayStandardMax is the standard length we set as a VARCHAR max in tables for a string array
const FFStringArrayStandardMax = 1024

func NewFFStringArray(initialContent ...string) FFStringArray {
	sa := make(FFStringArray, 0, len(initialContent))
	for _, s := range initialContent {
		if s != "" {
			sa = append(sa, s)
		}
	}
	return sa
}

func (sa FFStringArray) Value() (driver.Value, error) {
	if sa == nil {
		return "", nil
	}
	return strings.Join([]string(sa), ","), nil
}

func (sa *FFStringArray) Scan(src interface{}) error {
	switch st := src.(type) {
	case string:
		if st == "" {
			*sa = []string{}
			return nil
		}
		*sa = strings.Split(st, ",")
		return nil
	case []byte:
		if len(st) == 0 {
			*sa = []string{}
			return nil
		}
		*sa = strings.Split(string(st), ",")
		return nil
	case FFStringArray:
		*sa = st
		return nil
	case nil:
		*sa = []string{}
		return nil
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, sa)
	}
}

func (sa FFStringArray) String() string {
	if sa == nil {
		return ""
	}
	return strings.Join([]string(sa), ",")
}

func (sa FFStringArray) Validate(ctx context.Context, fieldName string, isName bool) error {
	var totalLength int
	dupCheck := make(map[string]bool)
	for i, n := range sa {
		if dupCheck[n] {
			return i18n.NewError(ctx, i18n.MsgDuplicateArrayEntry, fieldName, i, n)
		}
		dupCheck[n] = true
		totalLength += len(n)
		if isName {
			if err := ValidateFFNameField(ctx, n, fmt.Sprintf("%s[%d]", fieldName, i)); err != nil {
				return err
			}
		} else {
			if err := ValidateSafeCharsOnly(ctx, n, fmt.Sprintf("%s[%d]", fieldName, i)); err != nil {
				return err
			}
		}
	}
	if isName && len(sa) > FFStringNameItemsMax {
		return i18n.NewError(ctx, i18n.MsgTooManyItems, fieldName, FFStringNameItemsMax, len(sa))
	}
	if totalLength > FFStringArrayStandardMax {
		return i18n.NewError(ctx, i18n.MsgFieldTooLong, fieldName, FFStringArrayStandardMax)
	}
	return nil
}

func (sa FFStringArray) appendLowerIfUnique(s string) FFStringArray {
	if s == "" {
		return sa
	}
	for _, existing := range sa {
		if strings.EqualFold(s, existing) {
			return sa
		}
	}
	return append(sa, strings.ToLower(s))
}

// AddToSortedSet determines if the new string is already in the set of strings (case insensitive),
// and if not it adds it to the list (lower case) and returns a new slice of alphabetically sorted
// strings reference and true.
// If no change is made, the original reference is returned and false.
func (sa FFStringArray) AddToSortedSet(newValues ...string) (res FFStringArray, changed bool) {
	res = sa
	for _, s := range newValues {
		res = res.appendLowerIfUnique(s)
	}
	if len(res) != len(sa) {
		sort.Strings(res)
		return res, true
	}
	return sa, false
}
