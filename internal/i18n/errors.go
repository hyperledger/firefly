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
	"context"

	"github.com/pkg/errors"
)

// NewError creates a new error
func NewError(ctx context.Context, msg MessageKey, inserts ...interface{}) error {
	return errors.Errorf(SanitizeLimit(ExpandWithCode(ctx, msg, inserts...), 2048))
}

// WrapError wraps an error
func WrapError(ctx context.Context, err error, msg MessageKey, inserts ...interface{}) error {
	if err == nil {
		return NewError(ctx, msg, inserts...)
	}
	return errors.Wrap(err, SanitizeLimit(ExpandWithCode(ctx, msg, inserts...), 2048))
}
