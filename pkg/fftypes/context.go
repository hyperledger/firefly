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

// Context is this local node's state record for a context. It records the
// node's latest allocated sequence number for the context.
// A context is a hash of a GroupID and a topic, concattenated together
type Context struct {
	Hash  *Bytes32 `json:"hash"`
	Nonce int64    `json:"nonce"`
	Group *UUID    `json:"group"`
	Topic string   `json:"topic"`
}
