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

type InvokeContractRequest struct {
	ContractID *UUID                  `json:"contractId,omitempty"`
	Ledger     Byteable               `json:"ledger,omitempty"`
	Location   Byteable               `json:"location,omitempty"`
	Method     *FFIMethod             `json:"method,omitempty"`
	Params     map[string]interface{} `json:"params"`
}

type ContractAPI struct {
	ID        *UUID         `json:"id,omitempty"`
	Namespace string        `json:"namespace,omitempty"`
	Interface *FFIReference `json:"interface"`
	Ledger    Byteable      `json:"ledger,omitempty"`
	Location  Byteable      `json:"location,omitempty"`
	Name      string        `json:"name"`
	Message   *UUID         `json:"message,omitempty"`
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
