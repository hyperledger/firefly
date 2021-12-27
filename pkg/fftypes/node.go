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

import (
	"context"

	"github.com/hyperledger/firefly/internal/i18n"
)

// Node is a FireFly node within the network
type Node struct {
	ID          *UUID   `json:"id"`
	Message     *UUID   `json:"message,omitempty"`
	Owner       string  `json:"owner,omitempty"`
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	DX          DXInfo  `json:"dx"`
	Created     *FFTime `json:"created,omitempty"`
}

// DXInfo is the data exchange information
type DXInfo struct {
	Peer     string     `json:"peer,omitempty"`
	Endpoint JSONObject `json:"endpoint,omitempty"`
}

func (n *Node) Validate(ctx context.Context, existing bool) (err error) {
	if err = ValidateFFNameFieldNoUUID(ctx, n.Name, "name"); err != nil {
		return err
	}
	if err = ValidateLength(ctx, n.Description, "description", 4096); err != nil {
		return err
	}
	if n.Owner == "" {
		return i18n.NewError(ctx, i18n.MsgOwnerMissing)
	}
	if existing {
		if n.ID == nil {
			return i18n.NewError(ctx, i18n.MsgNilID)
		}
	}
	return nil
}

func (n *Node) Topic() string {
	return orgTopic(n.Owner)
}

func (n *Node) SetBroadcastMessage(msgID *UUID) {
	n.Message = msgID
}
