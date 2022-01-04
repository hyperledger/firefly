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

import "context"

type InvokeContractRequest struct {
	Interface *UUID                  `json:"interface,omitempty"`
	Ledger    Byteable               `json:"ledger,omitempty"`
	Location  Byteable               `json:"location,omitempty"`
	Key       string                 `json:"key,omitempty"`
	Method    *FFIMethod             `json:"method,omitempty"`
	Input     map[string]interface{} `json:"input"`
}

type ContractSubscribeRequest struct {
	Interface *UUID     `json:"interface,omitempty"`
	Location  Byteable  `json:"location,omitempty"`
	Event     *FFIEvent `json:"event,omitempty"`
}

type ContractURLs struct {
	OpenAPI string `json:"openapi"`
	UI      string `json:"ui"`
}

type ContractAPI struct {
	ID        *UUID         `json:"id,omitempty"`
	Namespace string        `json:"namespace,omitempty"`
	Interface *FFIReference `json:"interface"`
	Ledger    Byteable      `json:"ledger,omitempty"`
	Location  Byteable      `json:"location,omitempty"`
	Name      string        `json:"name"`
	Message   *UUID         `json:"message,omitempty"`
	URLs      ContractURLs  `json:"urls"`
}

func (c *ContractAPI) Validate(ctx context.Context, existing bool) (err error) {
	if err = ValidateFFNameField(ctx, c.Namespace, "namespace"); err != nil {
		return err
	}
	if err = ValidateFFNameField(ctx, c.Name, "name"); err != nil {
		return err
	}
	return nil
}

func (c *ContractAPI) Topic() string {
	return namespaceTopic(c.Namespace)
}

func (c *ContractAPI) SetBroadcastMessage(msgID *UUID) {
	c.Message = msgID
}
