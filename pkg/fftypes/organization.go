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

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly/internal/i18n"
)

const (
	DIDPrefix            = "did:"
	FireFlyDIDPrefix     = "did:firefly:"
	FireFlyOrgDIDPrefix  = "did:firefly:org/"
	FireFlyNodeDIDPrefix = "did:firefly:node/"
	OrgTopic             = "ff_organizations"
)

// Organization is a top-level identity in the network
type Organization struct {
	ID          *UUID      `json:"id"`
	Message     *UUID      `json:"message,omitempty"`
	Parent      string     `json:"parent,omitempty"`
	Identity    string     `json:"identity,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Profile     JSONObject `json:"profile,omitempty"`
	Created     *FFTime    `json:"created,omitempty"`
}

func (org *Organization) Validate(ctx context.Context, existing bool) (err error) {
	if err = ValidateFFNameFieldNoUUID(ctx, org.Name, "name"); err != nil {
		return err
	}
	if err = ValidateLength(ctx, org.Description, "description", 4096); err != nil {
		return err
	}
	if existing {
		if org.ID == nil {
			return i18n.NewError(ctx, i18n.MsgNilID)
		}
	}
	return nil
}

func (org *Organization) Topic() string {
	return OrgTopic
}

func (org *Organization) SetBroadcastMessage(msgID *UUID) {
	org.Message = msgID
}

func (org *Organization) GetDID() string {
	if org == nil {
		return ""
	}
	return fmt.Sprintf("%s%s", FireFlyOrgDIDPrefix, org.Name)
}
