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

	"github.com/hyperledger-labs/firefly/internal/data"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/karlseguin/ccache"
)

type GroupManager interface {
	GetGroupByID(ctx context.Context, id string) (*fftypes.Group, error)
	GetGroups(ctx context.Context, filter database.AndFilter) ([]*fftypes.Group, error)
	ResolveInitGroup(ctx context.Context, msg *fftypes.Message) (*fftypes.Group, error)
}

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
		Namespace: group.Namespace, // must go in the same ordering context as the message
		Created:   fftypes.Now(),
	}
	data.Value, err = json.Marshal(&group)
	if err == nil {
		err = group.Validate(ctx, true)
		if err == nil {
			err = data.Seal(ctx)
		}
	}
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// In the case of groups, we actually write the unconfirmed group directly to our database.
	// So it can be used straight away.
	// We're able to do this by making the identifier of the group a hash of the identity fields
	// (name, ledger and member list), as that is all the group contains. There's no data in there.
	if err = gm.database.UpsertGroup(ctx, group, true); err != nil {
		return err
	}

	// Write as data to the local store
	if err = gm.database.UpsertData(ctx, data, true, false /* we just generated the ID, so it is new */); err != nil {
		return err
	}

	// Create a private send message referring to the data
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Group:     group.Hash,
			Namespace: group.Namespace, // Must go into the same ordering context as the message itself
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

func (gm *groupManager) GetGroupByID(ctx context.Context, hash string) (*fftypes.Group, error) {
	h, err := fftypes.ParseBytes32(ctx, hash)
	if err != nil {
		return nil, err
	}
	return gm.database.GetGroupByHash(ctx, h)
}

func (gm *groupManager) GetGroups(ctx context.Context, filter database.AndFilter) ([]*fftypes.Group, error) {
	return gm.database.GetGroups(ctx, filter)
}

func (gm *groupManager) getGroupNodes(ctx context.Context, groupHash *fftypes.Bytes32) ([]*fftypes.Node, error) {

	if cached := gm.groupCache.Get(groupHash.String()); cached != nil {
		cached.Extend(gm.groupCacheTTL)
		return cached.Value().([]*fftypes.Node), nil
	}

	group, err := gm.database.GetGroupByHash(ctx, groupHash)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, i18n.NewError(ctx, i18n.MsgGroupNotFound, groupHash)
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

	gm.groupCache.Set(group.Hash.String(), nodes, gm.groupCacheTTL)
	return nodes, nil
}

// ResolveInitGroup is called when a message comes in as the first private message on a particular context.
// If the message is a group creation request, then it is validated and the group is created.
// Otherwise, the existing group must exist.
//
// Errors are only returned for database issues. For validation issues, a nil group is returned without an error.
func (gm *groupManager) ResolveInitGroup(ctx context.Context, msg *fftypes.Message) (*fftypes.Group, error) {
	if msg.Header.Tag == string(fftypes.SystemTagDefineGroup) {
		// Store the new group
		data, foundAll, err := gm.data.GetMessageData(ctx, msg, true)
		if err != nil || !foundAll || len(data) == 0 {
			log.L(ctx).Warnf("Group %s definition in message %s invalid: missing data", msg.Header.Group, msg.Header.ID)
			return nil, err
		}
		var newGroup fftypes.Group
		err = json.Unmarshal(data[0].Value, &newGroup)
		if err != nil {
			log.L(ctx).Warnf("Group %s definition in message %s invalid: %s", msg.Header.Group, msg.Header.ID, err)
			return nil, nil
		}
		err = newGroup.Validate(ctx, true)
		if err != nil {
			log.L(ctx).Warnf("Group %s definition in message %s invalid: %s", msg.Header.Group, msg.Header.ID, err)
			return nil, nil
		}
		if !newGroup.Hash.Equals(msg.Header.Group) {
			log.L(ctx).Warnf("Group %s definition in message %s invalid: mismatched hash with message '%s'", msg.Header.Group, msg.Header.ID, newGroup.Hash)
			return nil, nil
		}
		newGroup.Message = msg.Header.ID
		err = gm.database.UpsertGroup(ctx, &newGroup, true)
		if err != nil {
			return nil, err
		}
		event := fftypes.NewEvent(fftypes.EventTypeGroupConfirmed, newGroup.Namespace, nil, newGroup.Hash)
		if err = gm.database.UpsertEvent(ctx, event, false); err != nil {
			return nil, err
		}
		return &newGroup, nil
	}

	// Get the existing group
	group, err := gm.database.GetGroupByHash(ctx, msg.Header.Group)
	if err != nil {
		return group, err
	}
	if group == nil {
		log.L(ctx).Warnf("Group %s not found for first message in context. type=%s namespace=%s", msg.Header.Group, msg.Header.Type, msg.Header.Namespace)
		return nil, nil
	}
	return group, nil
}
