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

// type InterfaceDefinition struct {
// 	ID        *UUID  `json:"id,omitempty"`
// 	Namespace string `json:"namespace,omitempty"`
// 	Message   *UUID  `json:"message,omitempty"`
// 	FFI       *FFI   `json:"interface,omitempty"`
// }

// type InterfaceDefinitionBroadcast struct {
// 	InterfaceDefinition
// }

// func (id *InterfaceDefinition) Validate(ctx context.Context, existing bool) (err error) {
// 	if err = ValidateFFNameField(ctx, id.Namespace, "namespace"); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (id *InterfaceDefinition) Topic() string {
// 	return namespaceTopic(id.Namespace)
// }

// func (id *InterfaceDefinitionBroadcast) SetBroadcastMessage(msgID *UUID) {
// 	id.Message = msgID
// }

// type ContractInstance struct {
// 	ID                 *UUID                `json:"id,omitempty"`
// 	Namespace          string               `json:"namespace,omitempty"`
// 	Name               string               `json:"name,omitempty"`
// 	Message            *UUID                `json:"message,omitempty"`
// 	OnChainLocation    OnChainLocation      `json:"onChainLocation,omitempty"`
// 	ContractDefinition *InterfaceDefinition `json:"contractDefinition,omitempty"`
// }

// type ContractInstanceBroadcast struct {
// 	ContractInstance
// }

// func (ci *ContractInstance) Validate(ctx context.Context, existing bool) (err error) {
// 	if err = ValidateFFNameField(ctx, ci.Namespace, "namespace"); err != nil {
// 		return err
// 	}
// 	if ci.Name != "" {
// 		if err = ValidateFFNameField(ctx, ci.Name, "name"); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (ci *ContractInstance) Topic() string {
// 	return namespaceTopic(ci.Namespace)
// }

// func (ci *ContractInstance) SetBroadcastMessage(msgID *UUID) {
// 	ci.Message = msgID
// }

type ContractInvocationRequest struct {
	OnChainLocation OnChainLocation        `json:"onChainLocation,omitempty"`
	Method          string                 `json:"method,omitempty"`
	Params          map[string]interface{} `json:"params"`
}

type OnChainLocation interface{}
