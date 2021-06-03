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
	SystemNamespace = "ff-system"

	// SystemTopicDefineDatatype is the topic for messages that broadcast data definitions
	SystemTopicDefineDatatype = "ff-define-datatype"

	// SystemTopicDefineNamespace is the topic for messages that broadcast namespace definitions
	SystemTopicDefineNamespace = "ff-define-namespace"

	// SystemTopicDefineOrganization is the topic for messages that broadcast organization definitions
	SystemTopicDefineOrganization = "ff-define-organization"

	// SystemTopicDefineNode is the topic for messages that broadcast node definitions
	SystemTopicDefineNode = "ff-define-node"

	// SystemTopicDefineGroup is the topic for messages that send the definition of a group, to all parties in that group
	SystemTopicDefineGroup = "ff-define-group"
)
