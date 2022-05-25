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

import (
	"context"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
)

type TokenType = FFEnum

var (
	TokenTypeFungible    = ffEnum("tokentype", "fungible")
	TokenTypeNonFungible = ffEnum("tokentype", "nonfungible")
)

// TokenPoolState is the current confirmation state of a token pool
type TokenPoolState = FFEnum

var (
	// TokenPoolStateUnknown is a token pool that may not yet be activated
	// (should not be used in the code - only set via database migration for previously-created pools)
	TokenPoolStateUnknown = ffEnum("tokenpoolstate", "unknown")
	// TokenPoolStatePending is a token pool that has been announced but not yet confirmed
	TokenPoolStatePending = ffEnum("tokenpoolstate", "pending")
	// TokenPoolStateConfirmed is a token pool that has been confirmed on chain
	TokenPoolStateConfirmed = ffEnum("tokenpoolstate", "confirmed")
)

type TokenPool struct {
	ID        *fftypes.UUID      `ffstruct:"TokenPool" json:"id,omitempty" ffexcludeinput:"true"`
	Type      TokenType          `ffstruct:"TokenPool" json:"type" ffenum:"tokentype"`
	Namespace string             `ffstruct:"TokenPool" json:"namespace,omitempty" ffexcludeinput:"true"`
	Name      string             `ffstruct:"TokenPool" json:"name,omitempty"`
	Standard  string             `ffstruct:"TokenPool" json:"standard,omitempty" ffexcludeinput:"true"`
	Locator   string             `ffstruct:"TokenPool" json:"locator,omitempty" ffexcludeinput:"true"`
	Key       string             `ffstruct:"TokenPool" json:"key,omitempty"`
	Symbol    string             `ffstruct:"TokenPool" json:"symbol,omitempty"`
	Decimals  int                `ffstruct:"TokenPool" json:"decimals,omitempty" ffexcludeinput:"true"`
	Connector string             `ffstruct:"TokenPool" json:"connector,omitempty"`
	Message   *fftypes.UUID      `ffstruct:"TokenPool" json:"message,omitempty" ffexcludeinput:"true"`
	State     TokenPoolState     `ffstruct:"TokenPool" json:"state,omitempty" ffenum:"tokenpoolstate" ffexcludeinput:"true"`
	Created   *fftypes.FFTime    `ffstruct:"TokenPool" json:"created,omitempty" ffexcludeinput:"true"`
	Config    fftypes.JSONObject `ffstruct:"TokenPool" json:"config,omitempty" ffexcludeoutput:"true"` // for REST calls only (not stored)
	Info      fftypes.JSONObject `ffstruct:"TokenPool" json:"info,omitempty" ffexcludeinput:"true"`
	TX        TransactionRef     `ffstruct:"TokenPool" json:"tx,omitempty" ffexcludeinput:"true"`
}

type TokenPoolAnnouncement struct {
	Pool *TokenPool `json:"pool"`
}

func (t *TokenPool) Validate(ctx context.Context) (err error) {
	if err = ValidateFFNameField(ctx, t.Namespace, "namespace"); err != nil {
		return err
	}
	if err = ValidateFFNameFieldNoUUID(ctx, t.Name, "name"); err != nil {
		return err
	}
	return nil
}

func (t *TokenPoolAnnouncement) Topic() string {
	return typeNamespaceNameTopicHash("tokenpool", t.Pool.Namespace, t.Pool.Name)
}

func (t *TokenPoolAnnouncement) SetBroadcastMessage(msgID *fftypes.UUID) {
	t.Pool.Message = msgID
}
