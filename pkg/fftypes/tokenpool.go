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
)

type TokenType = FFEnum

var (
	TokenTypeFungible    TokenType = ffEnum("tokentype", "fungible")
	TokenTypeNonFungible TokenType = ffEnum("tokentype", "nonfungible")
)

type TokenPool struct {
	ID         *UUID          `json:"id,omitempty"`
	Type       TokenType      `json:"type" ffenum:"tokentype"`
	Namespace  string         `json:"namespace,omitempty"`
	Name       string         `json:"name,omitempty"`
	ProtocolID string         `json:"protocolId,omitempty"`
	Key        string         `json:"key,omitempty"`
	Symbol     string         `json:"symbol,omitempty"`
	Connector  string         `json:"connector,omitempty"`
	Message    *UUID          `json:"message,omitempty"`
	Created    *FFTime        `json:"created,omitempty"`
	Config     JSONObject     `json:"config,omitempty"`
	TX         TransactionRef `json:"tx,omitempty"`
}

type TokenPoolAnnouncement struct {
	TokenPool
	ProtocolTxID string `json:"protocolTxID"`
}

func (t *TokenPool) Validate(ctx context.Context, existing bool) (err error) {
	if err = ValidateFFNameField(ctx, t.Namespace, "namespace"); err != nil {
		return err
	}
	if err = ValidateFFNameField(ctx, t.Name, "name"); err != nil {
		return err
	}
	return nil
}

func (t *TokenPool) Topic() string {
	return namespaceTopic(t.Namespace)
}

func (t *TokenPool) SetBroadcastMessage(msgID *UUID) {
	t.Message = msgID
}
