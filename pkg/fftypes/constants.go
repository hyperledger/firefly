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

const (

	// SystemNamespace is the system reserved namespace name
	SystemNamespace = "ff_system"
)

type SystemTag string

const (

	// SystemTagDefineDatatype is the tag for messages that broadcast data definitions
	SystemTagDefineDatatype SystemTag = "ff_define_datatype"

	// SystemTagDefineNamespace is the tag for messages that broadcast namespace definitions
	SystemTagDefineNamespace SystemTag = "ff_define_namespace"

	// SystemTagDefineOrganization is the tag for messages that broadcast organization definitions
	SystemTagDefineOrganization SystemTag = "ff_define_organization"

	// SystemTagDefineNode is the tag for messages that broadcast node definitions
	SystemTagDefineNode SystemTag = "ff_define_node"

	// SystemTagDefineGroup is the tag for messages that send the definition of a group, to all parties in that group
	SystemTagDefineGroup SystemTag = "ff_define_group"

	// SystemTagDefinePool is the tag for messages that broadcast data definitions
	SystemTagDefinePool SystemTag = "ff_define_pool"

	// SystemTagDefineFFI is the tag for messages that broadcast contract FFIs
	SystemTagDefineFFI SystemTag = "ff_define_ffi"

	// SystemTagDefineContractAPI is the tag for messages that broadcast contract APIs
	SystemTagDefineContractAPI SystemTag = "ff_define_contract_api"

	// SystemTagIdentityClaim is the tag for messages that broadcast an identity claim
	SystemTagIdentityClaim SystemTag = "ff_identity_claim"

	// SystemTagIdentityVerification is the tag for messages that broadcast an identity verification
	SystemTagIdentityVerification SystemTag = "ff_identity_verification"

	// SystemTagIdentityUpdate is the tag for messages that broadcast an identity update
	SystemTagIdentityUpdate SystemTag = "ff_identity_update"
)
