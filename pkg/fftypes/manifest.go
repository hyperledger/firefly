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

import "encoding/json"

// Manifest is a list of references to messages and data
type Manifest struct {
	Messages []MessageRef `json:"messages"`
	Data     []DataRef    `json:"data"`
}

func (mf *Manifest) String() string {
	b, _ := json.Marshal(&mf)
	return string(b)
}
