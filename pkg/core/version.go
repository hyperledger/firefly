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

package core

type Version struct {
	Version    string `json:"Version,omitempty" yaml:"Version,omitempty"`
	Commit     string `json:"Commit,omitempty" yaml:"Commit,omitempty"`
	Date       string `json:"Date,omitempty" yaml:"Date,omitempty"`
	License    string `json:"License,omitempty" yaml:"License,omitempty"`
	APIVersion string `json:"APIVersion,omitempty" yaml:"APIVersion"`
}
