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

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

func (pm *privateMessaging) SendMessage(ctx context.Context, ns string, in *fftypes.MessageInput) (out *fftypes.Message, err error) {
	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin
	in.Header.Namespace = ns
	in.Header.Type = fftypes.MessageTypePrivate
	if in.Header.Author == "" {
		in.Header.Author = config.GetString(config.NodeIdentity)
	}
	in.Header.TX.Type = fftypes.TransactionTypeBatchPin
	err = pm.database.RunAsGroup(ctx, func(ctx context.Context) error {

		// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
		in.Message.Data, err = pm.data.ResolveInputData(ctx, ns, in.InputData)
		if err != nil {
			return err
		}

		// Seal the message
		if err = in.Message.Seal(ctx); err != nil {
			return err
		}

		// Store the message - this asynchronously triggers the next step in process
		return pm.database.UpsertMessage(ctx, &in.Message, false /* newly generated UUID in Seal */, false)
	})
	if err != nil {
		return nil, err
	}
	// The broadcastMessage function modifies the input message to create all the refs
	return &in.Message, err
}
