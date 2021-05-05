// Copyright © 2021 Kaleido, Inc.
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

package engine

import (
	"context"
	"encoding/json"

	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
)

func (e *engine) BroadcastSchemaDefinition(ctx context.Context, s *fftypes.Schema) (msg *fftypes.Message, err error) {

	// Validate the input schema data
	s.ID = fftypes.NewUUID()
	s.Created = fftypes.NowMillis()
	if s.Type == "" {
		s.Type = fftypes.SchemaTypeJSONSchema
	}
	if s.Type != fftypes.SchemaTypeJSONSchema {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownFieldValue, "type")
	}
	if err = fftypes.ValidateFFNameField(ctx, s.Namespace, "namespace"); err != nil {
		return nil, err
	}
	if err = fftypes.ValidateFFNameField(ctx, s.Entity, "entity"); err != nil {
		return nil, err
	}
	if err = fftypes.ValidateFFNameField(ctx, s.Version, "version"); err != nil {
		return nil, err
	}
	if len(s.Value) == 0 {
		return nil, i18n.NewError(ctx, i18n.MsgMissingRequiredField, "value")
	}
	if s.Hash, err = s.Value.Hash(ctx, "value"); err != nil {
		return nil, err
	}

	// Serialize it into a data object, as a piece of data we can write to a message
	data := &fftypes.Data{
		Type:      fftypes.DataTypeDefinition,
		ID:        fftypes.NewUUID(),
		Namespace: s.Namespace,
		Created:   fftypes.NowMillis(),
	}
	b, _ := json.Marshal(&s)
	_ = json.Unmarshal(b, &data.Value)
	data.Hash, _ = data.Value.Hash(ctx, "value")

	// Write as data to the local store
	if err = e.persistence.UpsertData(ctx, data); err != nil {
		return nil, err
	}

	// Create a broadcast message referring to the data
	msg = &fftypes.Message{
		Header: fftypes.MessageHeader{
			Type:    fftypes.MessageTypeDefinition,
			Author:  e.nodeIdentity,
			Topic:   fftypes.SchemaTopicDefinitionName,
			Context: fftypes.SystemContext,
		},
		Data: fftypes.DataRefs{
			{ID: data.ID, Hash: data.Hash},
		},
	}

	// Broadcast the message
	if err = e.broadcast.BroadcastMessage(ctx, e.nodeIdentity, msg, data); err != nil {
		return nil, err
	}

	return msg, nil
}
