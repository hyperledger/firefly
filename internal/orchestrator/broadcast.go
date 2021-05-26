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

package orchestrator

import (
	"context"
	"encoding/json"

	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

func (or *orchestrator) broadcastDefinition(ctx context.Context, ns string, defObject interface{}, topic string) (msg *fftypes.Message, err error) {

	// Serialize it into a data object, as a piece of data we can write to a message
	data := &fftypes.Data{
		Validator: fftypes.ValidatorTypeDatatype,
		ID:        fftypes.NewUUID(),
		Namespace: ns,
		Created:   fftypes.Now(),
	}
	data.Value, err = json.Marshal(&defObject)
	if err == nil {
		err = data.Seal(ctx)
	}
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// Write as data to the local store
	if err = or.database.UpsertData(ctx, data, true, false /* we just generated the ID, so it is new */); err != nil {
		return nil, err
	}

	// Create a broadcast message referring to the data
	msg = &fftypes.Message{
		Header: fftypes.MessageHeader{
			Namespace: ns,
			Type:      fftypes.MessageTypeDefinition,
			Author:    or.nodeIDentity,
			Topic:     topic,
			Context:   fftypes.SystemContext,
			TX: fftypes.TransactionRef{
				Type: fftypes.TransactionTypePin,
			},
		},
		Data: fftypes.DataRefs{
			{ID: data.ID, Hash: data.Hash},
		},
	}

	// Broadcast the message
	if err = or.broadcast.BroadcastMessage(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (or *orchestrator) BroadcastDatatype(ctx context.Context, ns string, datatype *fftypes.Datatype) (msg *fftypes.Message, err error) {

	// Validate the input data definition data
	datatype.ID = fftypes.NewUUID()
	datatype.Created = fftypes.Now()
	datatype.Namespace = ns
	if datatype.Validator == "" {
		datatype.Validator = fftypes.ValidatorTypeJSON
	}
	if datatype.Validator != fftypes.ValidatorTypeJSON {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownFieldValue, "validator")
	}
	if err = or.verifyNamespaceExists(ctx, datatype.Namespace); err != nil {
		return nil, err
	}
	if err = fftypes.ValidateFFNameField(ctx, datatype.Name, "name"); err != nil {
		return nil, err
	}
	if err = fftypes.ValidateFFNameField(ctx, datatype.Version, "version"); err != nil {
		return nil, err
	}
	if len(datatype.Value) == 0 {
		return nil, i18n.NewError(ctx, i18n.MsgMissingRequiredField, "value")
	}
	datatype.Hash = datatype.Value.Hash()

	// Verify the data type is now all valid, before we broadcast it
	err = or.data.CheckDatatype(ctx, datatype)
	if err != nil {
		return nil, err
	}

	return or.broadcastDefinition(ctx, ns, datatype, fftypes.DatatypeTopicName)
}

func (or *orchestrator) BroadcastNamespace(ctx context.Context, ns *fftypes.Namespace) (msg *fftypes.Message, err error) {

	// Validate the input data definition data
	ns.ID = fftypes.NewUUID()
	ns.Created = fftypes.Now()
	ns.Type = fftypes.NamespaceTypeBroadcast
	if err = fftypes.ValidateFFNameField(ctx, ns.Name, "name"); err != nil {
		return nil, err
	}
	if err = fftypes.ValidateLength(ctx, ns.Description, "description", 4096); err != nil {
		return nil, err
	}

	return or.broadcastDefinition(ctx, ns.Name, ns, fftypes.NamespaceDefinitionTopicName)
}
