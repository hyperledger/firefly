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

	"github.com/kaleido-io/firefly/internal/i18n"
)

// Organization is a top-level identity in the network
type Organization struct {
	ID          *UUID      `json:"id"`
	Parent      *UUID      `json:"parent,omitempty"`
	Identity    string     `json:"identity,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Profile     JSONObject `json:"profile,omitempty"`
	Created     *FFTime    `json:"created,omitempty"`
	Confirmed   *FFTime    `json:"updated,omitempty"`
}

func (org *Organization) Validate(ctx context.Context, existing bool) (err error) {
	if err = ValidateFFNameField(ctx, org.Name, "name"); err != nil {
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
