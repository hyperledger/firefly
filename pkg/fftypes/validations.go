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
	"regexp"

	"github.com/hyperledger/firefly/internal/i18n"
)

var (
	ffNameValidator = regexp.MustCompile(`^[0-9a-zA-Z]([0-9a-zA-Z._-]{0,62}[0-9a-zA-Z])?$`)
)

func ValidateFFNameField(ctx context.Context, str string, fieldName string) error {
	if !ffNameValidator.MatchString(str) {
		return i18n.NewError(ctx, i18n.MsgInvalidName, fieldName)
	}
	if _, err := ParseUUID(ctx, str); err == nil {
		// Name must not be a UUID
		return i18n.NewError(ctx, i18n.MsgNoUUID, fieldName)
	}
	return nil
}

func ValidateLength(ctx context.Context, str string, fieldName string, max int) error {
	if len([]byte(str)) > max {
		return i18n.NewError(ctx, i18n.MsgFieldTooLong, fieldName, max)
	}
	return nil
}
