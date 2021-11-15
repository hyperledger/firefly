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

import "context"

type FFI struct {
	ID        *UUID        `json:"id,omitempty"`
	Message   *UUID        `json:"message,omitempty"`
	Namespace string       `json:"namespace,omitempty"`
	Name      string       `json:"name,omitempty"`
	Version   string       `json:"version,omitempty"`
	Methods   []*FFIMethod `json:"methods"`
	Events    []*FFIEvent  `json:"events"`
}

type FFIMethod struct {
	Name    string
	Params  []*FFIParam `json:"params"`
	Returns []*FFIParam `json:"returns"`
}

type FFIEvent struct {
	Name   string      `json:"name"`
	Params []*FFIParam `json:"params"`
}

type FFIParam struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (f *FFI) Validate(ctx context.Context, existing bool) (err error) {
	if err = ValidateFFNameField(ctx, f.Namespace, "namespace"); err != nil {
		return err
	}
	if err = ValidateFFNameField(ctx, f.Name, "name"); err != nil {
		return err
	}
	if err = ValidateFFNameField(ctx, f.Version, "version"); err != nil {
		return err
	}
	return nil
}

func (f *FFI) Topic() string {
	return namespaceTopic(f.Namespace)
}

func (f *FFI) SetBroadcastMessage(msgID *UUID) {
	f.Message = msgID
}
