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

type ContractCallType = FFEnum

var (
	// CallTypeInvoke is an invocation that submits a transaction for inclusion in the chain
	CallTypeInvoke = ffEnum("contractcalltype", "invoke")
	// CallTypeQuery is a query that returns data from the chain
	CallTypeQuery = ffEnum("contractcalltype", "query")
)

type ContractCallRequest struct {
	Type       ContractCallType       `ffstruct:"ContractCallRequest" json:"type,omitempty" ffenum:"contractcalltype" ffexcludeinput:"true"`
	Interface  *UUID                  `ffstruct:"ContractCallRequest" json:"interface,omitempty" ffexcludeinput:"postContractAPIInvoke,postContractAPIQuery"`
	Location   *JSONAny               `ffstruct:"ContractCallRequest" json:"location,omitempty"`
	Key        string                 `ffstruct:"ContractCallRequest" json:"key,omitempty"`
	Method     *FFIMethod             `ffstruct:"ContractCallRequest" json:"method,omitempty" ffexcludeinput:"postContractAPIInvoke,postContractAPIQuery"`
	MethodPath string                 `ffstruct:"ContractCallRequest" json:"methodPath,omitempty" ffexcludeinput:"postContractAPIInvoke,postContractAPIQuery"`
	Input      map[string]interface{} `ffstruct:"ContractCallRequest" json:"input"`
}

type ContractURLs struct {
	OpenAPI string `ffstruct:"ContractURLs" json:"openapi"`
	UI      string `ffstruct:"ContractURLs" json:"ui"`
}

type ContractAPI struct {
	ID        *UUID         `ffstruct:"ContractAPI" json:"id,omitempty" ffexcludeinput:"true"`
	Namespace string        `ffstruct:"ContractAPI" json:"namespace,omitempty" ffexcludeinput:"true"`
	Interface *FFIReference `ffstruct:"ContractAPI" json:"interface"`
	Location  *JSONAny      `ffstruct:"ContractAPI" json:"location,omitempty"`
	Name      string        `ffstruct:"ContractAPI" json:"name"`
	Message   *UUID         `ffstruct:"ContractAPI" json:"message,omitempty" ffexcludeinput:"true"`
	URLs      ContractURLs  `ffstruct:"ContractAPI" json:"urls" ffexcludeinput:"true"`
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
	return typeNamespaceNameTopicHash("contractapi", c.Namespace, c.Name)
}

func (c *ContractAPI) SetBroadcastMessage(msgID *UUID) {
	c.Message = msgID
}

func (c *ContractAPI) LocationAndLedgerEquals(a *ContractAPI) bool {
	if c == nil || a == nil {
		return false
	}
	return c.Location.Hash().Equals(a.Location.Hash())
}
