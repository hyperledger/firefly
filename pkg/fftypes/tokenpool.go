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
	ID         *UUID          `json:"id,omitempty"`
	Type       TokenType      `json:"type" ffenum:"tokentype"`
	Namespace  string         `json:"namespace,omitempty"`
	Name       string         `json:"name,omitempty"`
	Standard   string         `json:"standard,omitempty"`
	ProtocolID string         `json:"protocolId,omitempty"`
	Key        string         `json:"key,omitempty"`
	Symbol     string         `json:"symbol,omitempty"`
	Connector  string         `json:"connector,omitempty"`
	Message    *UUID          `json:"message,omitempty"`
	State      TokenPoolState `json:"state,omitempty" ffenum:"tokenpoolstate"`
	Created    *FFTime        `json:"created,omitempty"`
	Config     JSONObject     `json:"config,omitempty"` // for REST calls only (not stored)
	Info       JSONObject     `json:"info,omitempty"`
	TX         TransactionRef `json:"tx,omitempty"`
}

type TokenPoolAnnouncement struct {
	Pool  *TokenPool       `json:"pool"`
	Event *BlockchainEvent `json:"event"`
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
	return namespaceTopic(t.Pool.Namespace)
}

func (t *TokenPoolAnnouncement) SetBroadcastMessage(msgID *UUID) {
	t.Pool.Message = msgID
}
