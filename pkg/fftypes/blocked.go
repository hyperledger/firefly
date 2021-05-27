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

// Blocked refers to a context that is currently blocked and no further
// messages will be confirmed until the blockage is removed (or this
// blocked record is manually deleted)
type Blocked struct {
	ID        *UUID   `json:"id,omitempty"`
	Namespace string  `json:"namespace,omitempty"`
	Context   string  `json:"context,omitempty"`
	Group     *UUID   `json:"group,omitempty"`
	Message   *UUID   `json:"message,omitempty"`
	Created   *FFTime `json:"created,omitempty"`
}
