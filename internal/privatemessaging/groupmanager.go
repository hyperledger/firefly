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
	"time"

	"github.com/kaleido-io/firefly/internal/data"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/karlseguin/ccache"
)

type groupManager struct {
	database      database.Plugin
	data          data.Manager
	groupCacheTTL time.Duration
	groupCache    *ccache.Cache
}

func (gm *groupManager) groupInit(ctx context.Context, signer *fftypes.Identity, group *fftypes.Group) (err error) {

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
			Tag:       string(fftypes.SystemTagDefineGroup),
			Topics:    fftypes.FFNameArray{group.Topic()},
			TxType:    fftypes.TransactionTypeBatchPin,
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

func (gm *groupManager) GetGroupByID(ctx context.Context, id string) (*fftypes.Group, error) {
	u, err := fftypes.ParseUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	return gm.database.GetGroupByID(ctx, u)
}

func (gm *groupManager) GetGroups(ctx context.Context, filter database.AndFilter) ([]*fftypes.Group, error) {
	return gm.database.GetGroups(ctx, filter)
}

func (gm *groupManager) getGroupNodes(ctx context.Context, groupID *fftypes.UUID) ([]*fftypes.Node, error) {

	if cached := gm.groupCache.Get(groupID.String()); cached != nil {
		cached.Extend(gm.groupCacheTTL)
		return cached.Value().([]*fftypes.Node), nil
	}

	group, err := gm.database.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, i18n.NewError(ctx, i18n.MsgGroupNotFound, groupID)
	}

	// We de-duplicate nodes in the case that the payload needs to be received by multiple org identities
	// that share a single node.
	nodes := make([]*fftypes.Node, 0, len(group.Members))
	knownIDs := make(map[fftypes.UUID]bool)
	for _, r := range group.Members {
		node, err := gm.database.GetNodeByID(ctx, r.Node)
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, i18n.NewError(ctx, i18n.MsgNodeNotFound, r.Node)
		}
		if !knownIDs[*node.ID] {
			knownIDs[*node.ID] = true
			nodes = append(nodes, node)
		}
	}

	gm.groupCache.Set(groupID.String(), nodes, gm.groupCacheTTL)
	return nodes, nil
}
