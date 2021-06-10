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
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/hyperledger-labs/firefly/internal/i18n"
)

// FFNameArray is an array of strings, each conforming to the requirements
// of a FireFly name, with a combined length (when joined with commas) of 1024
type FFNameArray []string

func (na FFNameArray) Value() (driver.Value, error) {
	if na == nil {
		return "", nil
	}
	return strings.Join([]string(na), ","), nil
}

func (na *FFNameArray) Scan(src interface{}) error {
	switch st := src.(type) {
	case string:
		if st == "" {
			*na = []string{}
			return nil
		}
		*na = strings.Split(st, ",")
		return nil
	case []byte:
		if len(st) == 0 {
			*na = []string{}
			return nil
		}
		*na = strings.Split(string(st), ",")
		return nil
	case FFNameArray:
		*na = st
		return nil
	case nil:
		return nil
	default:
		return i18n.NewError(context.Background(), i18n.MsgScanFailed, src, na)
	}
}

func (na FFNameArray) String() string {
	if na == nil {
		return ""
	}
	return strings.Join([]string(na), ",")
}

func (na FFNameArray) Validate(ctx context.Context, fieldName string) error {
	dupCheck := make(map[string]bool)
	for i, n := range na {
		if dupCheck[n] {
			return i18n.NewError(ctx, i18n.MsgDuplicateArrayEntry, fieldName, i, n)
		}
		dupCheck[n] = true
		if err := ValidateFFNameField(ctx, n, fmt.Sprintf("%s[%d]", fieldName, i)); err != nil {
			return err
		}
	}
	if len(na) > 15 {
		return i18n.NewError(ctx, i18n.MsgTooManyItems, fieldName, 15, len(na))
	}
	return nil
}
