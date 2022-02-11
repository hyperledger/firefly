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

package ethereum

var batchPinMethodABI = ABIElementMarshaling{
	Name: "pinBatch",
	Type: "function",
	Inputs: []ABIArgumentMarshaling{
		{
			InternalType: "string",
			Name:         "namespace",
			Type:         "string",
		},
		{
			InternalType: "bytes32",
			Name:         "uuids",
			Type:         "bytes32",
		},
		{
			InternalType: "bytes32",
			Name:         "batchHash",
			Type:         "bytes32",
		},
		{
			InternalType: "string",
			Name:         "payloadRef",
			Type:         "string",
		},
		{
			InternalType: "bytes32[]",
			Name:         "contexts",
			Type:         "bytes32[]",
		},
	},
}

var batchPinEventABI = ABIElementMarshaling{
	Name: "BatchPin",
	Type: "event",
	Inputs: []ABIArgumentMarshaling{
		{
			Indexed:      false,
			InternalType: "address",
			Name:         "author",
			Type:         "address",
		},
		{
			Indexed:      false,
			InternalType: "uint256",
			Name:         "timestamp",
			Type:         "uint256",
		},
		{
			Indexed:      false,
			InternalType: "string",
			Name:         "namespace",
			Type:         "string",
		},
		{
			Indexed:      false,
			InternalType: "bytes32",
			Name:         "uuids",
			Type:         "bytes32",
		},
		{
			Indexed:      false,
			InternalType: "bytes32",
			Name:         "batchHash",
			Type:         "bytes32",
		},
		{
			Indexed:      false,
			InternalType: "string",
			Name:         "payloadRef",
			Type:         "string",
		},
		{
			Indexed:      false,
			InternalType: "bytes32[]",
			Name:         "contexts",
			Type:         "bytes32[]",
		},
	},
}
