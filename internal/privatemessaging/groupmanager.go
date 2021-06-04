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
	"encoding/json"

	"github.com/kaleido-io/firefly/internal/data"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

type groupManager struct {
	database database.Plugin
	data     data.Manager
}

func (gm *groupManager) GroupInit(ctx context.Context, signer *fftypes.Identity, group *fftypes.Group) (err error) {

	// Serialize it into a data object, as a piece of data we can write to a message
	data := &fftypes.Data{
		Validator: fftypes.ValidatorTypeSystemDefinition,
		ID:        fftypes.NewUUID(),
		Namespace: fftypes.SystemNamespace,
		Created:   fftypes.Now(),
	}
	data.Value, err = json.Marshal(&group)
	if err == nil {
		err = data.Seal(ctx)
	}
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Write as data to the local store
	if err = gm.database.UpsertData(ctx, data, true, false /* we just generated the ID, so it is new */); err != nil {
		return err
	}

	// Create a private send message referring to the data
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: fftypes.SystemNamespace,
			Type:      fftypes.MessageTypeGroupInit,
			Author:    signer.Identifier,
			Topic:     fftypes.SystemTopicDefineGroup,
			Context:   group.Context(),
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypeBatchPin,
			},
		},
		Data: fftypes.DataRefs{
			{ID: data.ID, Hash: data.Hash},
		},
	}

	// Seal the message
	err = msg.Seal(ctx)
	if err == nil {
		// Store the message - this asynchronously triggers the next step in process
		err = gm.database.UpsertMessage(ctx, msg, false /* newly generated UUID in Seal */, false)
	}
	return err

}
