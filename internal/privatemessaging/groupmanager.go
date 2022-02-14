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

package privatemessaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/karlseguin/ccache"
)

type GroupManager interface {
	GetGroupByID(ctx context.Context, id string) (*fftypes.Group, error)
	GetGroupsNS(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Group, *database.FilterResult, error)
	ResolveInitGroup(ctx context.Context, msg *fftypes.Message) (*fftypes.Group, error)
	EnsureLocalGroup(ctx context.Context, group *fftypes.Group) (ok bool, err error)
}

type groupManager struct {
	database      database.Plugin
	data          data.Manager
	groupCacheTTL time.Duration
	groupCache    *ccache.Cache
}

type groupHashEntry struct {
	group *fftypes.Group
	nodes []*fftypes.Node
}

func (gm *groupManager) EnsureLocalGroup(ctx context.Context, group *fftypes.Group) (ok bool, err error) {
	if group == nil {
		return false, i18n.NewError(ctx, i18n.MsgGroupRequired)
	}

	// In the case that we've received a private message for a group, it's possible (likely actually)
	// that the private message using the group will arrive before the group init message confirming
	// the group via the blockchain.
	// So this method checks if a group exists, and if it doesn't inserts it.
	// We do assume the other side has sent the batch init of the group (rather than generating a second one)
	if g, err := gm.database.GetGroupByHash(ctx, group.Hash); err != nil {
		return false, err
	} else if g != nil {
		// The group already exists
		return true, nil
	}

	err = group.Validate(ctx, true)
	if err != nil {
		log.L(ctx).Errorf("Attempt to insert invalid group %s:%s: %s", group.Namespace, group.Hash, err)
		return false, nil
	}
	err = gm.database.UpsertGroup(ctx, group, database.UpsertOptimizationNew /* it could have been created by another thread, but we think we're first */)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (gm *groupManager) groupInit(ctx context.Context, signer *fftypes.IdentityRef, group *fftypes.Group) (err error) {

	// Serialize it into a data object, as a piece of data we can write to a message
	data := &fftypes.Data{
		Validator: fftypes.ValidatorTypeSystemDefinition,
		ID:        fftypes.NewUUID(),
		Namespace: group.Namespace, // must go in the same ordering context as the message
		Created:   fftypes.Now(),
	}
	b, err := json.Marshal(&group)
	if err == nil {
		data.Value = fftypes.JSONAnyPtrBytes(b)
		err = group.Validate(ctx, true)
		if err == nil {
			err = data.Seal(ctx, nil)
		}
	}
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgSerializationFailed)
	}

	// In the case of groups, we actually write the unconfirmed group directly to our database.
	// So it can be used straight away.
	// We're able to do this by making the identifier of the group a hash of the identity fields
	// (name, ledger and member list), as that is all the group contains. There's no data in there.
	if err = gm.database.UpsertGroup(ctx, group, database.UpsertOptimizationNew /* we think we're first */); err != nil {
		return err
	}

	// Write as data to the local store
	if err = gm.database.UpsertData(ctx, data, database.UpsertOptimizationNew); err != nil {
		return err
	}

	// Create a private send message referring to the data
	msg := &fftypes.Message{
		State: fftypes.MessageStateReady,
		Header: fftypes.MessageHeader{
			Group:       group.Hash,
			Namespace:   group.Namespace, // Must go into the same ordering context as the message itself
			Type:        fftypes.MessageTypeGroupInit,
			IdentityRef: *signer,
			Tag:         string(fftypes.SystemTagDefineGroup),
			Topics:      fftypes.FFStringArray{group.Topic()},
			TxType:      fftypes.TransactionTypeBatchPin,
		},
		Data: fftypes.DataRefs{
			{ID: data.ID, Hash: data.Hash},
		},
	}

	// Seal the message
	err = msg.Seal(ctx)
	if err == nil {
		// Store the message - this asynchronously triggers the next step in process
		err = gm.database.UpsertMessage(ctx, msg, database.UpsertOptimizationNew)
	}
	if err == nil {
		log.L(ctx).Infof("Created new group %s", group.Hash)
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

func (gm *groupManager) GetGroupsNS(ctx context.Context, ns string, filter database.AndFilter) ([]*fftypes.Group, *database.FilterResult, error) {
	return gm.GetGroups(ctx, filter.Condition(filter.Builder().Eq("namespace", ns)))
}

func (gm *groupManager) GetGroups(ctx context.Context, filter database.AndFilter) ([]*fftypes.Group, *database.FilterResult, error) {
	return gm.database.GetGroups(ctx, filter)
}

func (gm *groupManager) getGroupNodes(ctx context.Context, groupHash *fftypes.Bytes32) (*fftypes.Group, []*fftypes.Node, error) {

	if cached := gm.groupCache.Get(groupHash.String()); cached != nil {
		cached.Extend(gm.groupCacheTTL)
		ghe := cached.Value().(*groupHashEntry)
		return ghe.group, ghe.nodes, nil
	}

	group, err := gm.database.GetGroupByHash(ctx, groupHash)
	if err != nil {
		return nil, nil, err
	}
	if group == nil {
		return nil, nil, i18n.NewError(ctx, i18n.MsgGroupNotFound, groupHash)
	}

	// We de-duplicate nodes in the case that the payload needs to be received by multiple org identities
	// that share a single node.
	nodes := make([]*fftypes.Node, 0, len(group.Members))
	knownIDs := make(map[fftypes.UUID]bool)
	for _, r := range group.Members {
		node, err := gm.database.GetNodeByID(ctx, r.Node)
		if err != nil {
			return nil, nil, err
		}
		if node == nil {
			return nil, nil, i18n.NewError(ctx, i18n.MsgNodeNotFound, r.Node)
		}
		if !knownIDs[*node.ID] {
			knownIDs[*node.ID] = true
			nodes = append(nodes, node)
		}
	}

	gm.groupCache.Set(group.Hash.String(), &groupHashEntry{
		group: group,
		nodes: nodes,
	}, gm.groupCacheTTL)
	return group, nodes, nil
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
		err = json.Unmarshal(data[0].Value.Bytes(), &newGroup)
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
		err = gm.database.UpsertGroup(ctx, &newGroup, database.UpsertOptimizationNew /* we think we're first to create this */)
		if err != nil {
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
