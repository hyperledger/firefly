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

package privatemessaging

import (
	"context"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (pm *privateMessaging) SendMessage(ctx context.Context, ns string, in *fftypes.MessageInput) (out *fftypes.Message, err error) {
	in.Header.Namespace = ns
	in.Header.Type = fftypes.MessageTypePrivate
	if in.Header.Author == "" {
		in.Header.Author = config.GetString(config.OrgIdentity)
	}
	if in.Header.TxType == "" {
		in.Header.TxType = fftypes.TransactionTypeBatchPin
	}

	sender, err := pm.identity.Resolve(ctx, in.Header.Author)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
	}

	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin
	err = pm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		return pm.resolveAndSend(ctx, sender, in)
	})
	if err != nil {
		return nil, err
	}
	// The broadcastMessage function modifies the input message to create all the refs
	return &in.Message, err
}

func (pm *privateMessaging) resolveAndSend(ctx context.Context, sender *fftypes.Identity, in *fftypes.MessageInput) (err error) {
	// Resolve the member list into a group
	if err = pm.resolveReceipientList(ctx, sender, in); err != nil {
		return err
	}

	// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
	in.Message.Data, err = pm.data.ResolveInputData(ctx, in.Header.Namespace, in.InputData)
	if err != nil {
		return err
	}

	// Seal the message
	if err := in.Message.Seal(ctx); err != nil {
		return err
	}

	// Store the message - this asynchronously triggers the next step in process
	return pm.database.InsertMessageLocal(ctx, &in.Message)
}
