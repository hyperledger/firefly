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

// Parked represents a ledger-pinning event that has been
// detected from the blockchain, but cannot be processed immediately.
//
// A batch contains many messages, and each of those messages can be on a different
// topic (or topics)
// All messages on the same topic must be processed in the order that
// the batch pinning events arrive from the blockchain.
//
// For broadcasts the hash of all pinning events in the sequence
// is the same.
// For private group messages, the hash is obviscated in a way that it
// is deterministic what the next hash will be for each sender in that group.
//
// As we need to correlate the on-chain events, with off-chain data that might
// arrive at a different time (or never), we "park" all pinned sequences first,
// then only complete them (and generate the associated events) once all the data
// has been assembled for all messages on that sequence, within that batch.
type Parked struct {
	Sequence int64    `json:"sequence,omitempty"`
	Hash     *Bytes32 `json:"hash,omitempty"`
	Ledger   *UUID    `json:"ledger,omitempty"`
	Batch    *UUID    `json:"batch,omitempty"`
	Created  *FFTime  `json:"created,omitempty"`
}
