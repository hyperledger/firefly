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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hyperledger-labs/firefly/internal/i18n"
)

type GroupIdentity struct {
	Ledger    *UUID   `json:"ledger,omitempty"`
	Namespace string  `json:"namespace,omitempty"`
	Name      string  `json:"name"`
	Members   Members `json:"members"`
}

type Group struct {
	GroupIdentity
	Message *UUID    `json:"message,omitempty"`
	Hash    *Bytes32 `json:"hash,omitempty"`
	Created *FFTime  `json:"created,omitempty"`
}

type Members []*Member

func (m Members) Len() int           { return len(m) }
func (m Members) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m Members) Less(i, j int) bool { return m[i].Identity < m[j].Identity } // Note there's a dupcheck in validate

type Member struct {
	Identity string `json:"identity,omitempty"`
	Node     *UUID  `json:"node,omitempty"`
}

type MemberInput struct {
	Identity string `json:"identity,omitempty"`
	Node     string `json:"node,omitempty"`
}

func (man *GroupIdentity) Hash() *Bytes32 {
	b, _ := json.Marshal(&man)
	hash := Bytes32(sha256.Sum256(b))
	return &hash
}

func (group *Group) Validate(ctx context.Context, existing bool) (err error) {
	if err = ValidateFFNameField(ctx, group.Namespace, "namespace"); err != nil {
		return err
	}
	// We allow a blank name for a group (for auto creation)
	if group.Name != "" {
		if err = ValidateFFNameField(ctx, group.Name, "name"); err != nil {
			return err
		}
	}
	if len(group.Members) == 0 {
		return i18n.NewError(ctx, i18n.MsgGroupMustHaveMembers)
	}
	dupCheck := make(map[string]bool)
	for i, r := range group.Members {
		if r.Identity == "" {
			return i18n.NewError(ctx, i18n.MsgEmptyMemberIdentity, i)
		}
		if err = ValidateLength(ctx, r.Identity, "identity", 1024); err != nil {
			return err
		}
		if r.Node == nil {
			return i18n.NewError(ctx, i18n.MsgEmptyMemberNode, i)
		}
		key := fmt.Sprintf("%s:%s", r.Node, r.Identity)
		if dupCheck[key] {
			return i18n.NewError(ctx, i18n.MsgDuplicateMember, i)
		}
		dupCheck[key] = true
	}
	if existing {
		hash := group.GroupIdentity.Hash()
		if !group.Hash.Equals(hash) {
			return i18n.NewError(ctx, i18n.MsgGroupInvalidHash, group.Hash, hash)
		}
	}
	return nil
}

func (group *Group) Seal() {
	sort.Sort(group.Members)
	group.Hash = group.GroupIdentity.Hash()
}

func (group *Group) Topic() string {
	return group.Hash.String()
}

func (group *Group) SetBroadcastMessage(msgID *UUID) {
	group.Message = msgID
}
