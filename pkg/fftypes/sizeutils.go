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

	"github.com/docker/go-units"
	"github.com/hyperledger/firefly/internal/log"
)

// ParseToByteSize is a standard handling of a number of bytes, in config or API options
func ParseToByteSize(byteString string) int64 {
	if byteString == "" {
		return 0
	}
	bytes, err := units.RAMInBytes(byteString)
	if err != nil {
		log.L(context.Background()).Warn(err)
		return 0
	}
	return bytes
}
