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

// VerifierType is the type of an identity berifier. Where possible we use established DID verifier type strings
type VerifierType = FFEnum

var (
	// VerifierTypeEthAddress is an Ethereum (secp256k1) address string
	VerifierTypeEthAddress VerifierType = ffEnum("verifiertype", "ethereum_address")
	// VerifierTypeMSPIdentity is the MSP id (X509 distinguished name) of an issued signing certificate / keypair
	VerifierTypeMSPIdentity VerifierType = ffEnum("verifiertype", "fabric_msp_id")
)

// VerifierRef is just the type + value (public key identifier etc.) from the verifier
type VerifierRef struct {
	Type  VerifierType `json:"type" ffenum:"verifiertype"`
	Value string       `json:"value"`
}

// Verifier is an identity verification system that has been established for this identity, such as a blockchain signing key identifier
type Verifier struct {
	ID        *UUID  `json:"id"`
	Identity  *UUID  `json:"identity,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	VerifierRef
	Created *FFTime `json:"created,omitempty"`
}
