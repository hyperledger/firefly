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

const (

	// SystemNamespace is the system reserved namespace name
	SystemNamespace = "ff_system"
)

type SystemTag string

const (

	// SystemTagDefineDatatype is the topic for messages that broadcast data definitions
	SystemTagDefineDatatype SystemTag = "ff_define_datatype"

	// SystemTagDefineNamespace is the topic for messages that broadcast namespace definitions
	SystemTagDefineNamespace SystemTag = "ff_define_namespace"

	// SystemTagDefineOrganization is the topic for messages that broadcast organization definitions
	SystemTagDefineOrganization SystemTag = "ff_define_organization"

	// SystemTagDefineNode is the topic for messages that broadcast node definitions
	SystemTagDefineNode SystemTag = "ff_define_node"

	// SystemTagDefineGroup is the topic for messages that send the definition of a group, to all parties in that group
	SystemTagDefineGroup SystemTag = "ff_define_group"

	// SystemTagDefinePool is the topic for messages that broadcast data definitions
	SystemTagDefinePool SystemTag = "ff_define_pool"

	// SystemTagDefinePool is the topic for messages that broadcast contract definitions
	SystemTagDefineContract SystemTag = "ff_define_contract"
)
