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

package broadcast

import (
	"context"
	"encoding/json"

	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

func (bm *broadcastManager) HandleSystemBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)
	l.Infof("Confirming system broadcast '%s' [%s]", msg.Header.Topic, msg.Header.ID)
	switch msg.Header.Topic {
	case fftypes.SystemTopicBroadcastDatatype:
		return bm.handleDatatypeBroadcast(ctx, msg, data)
	case fftypes.SystemTopicBroadcastNamespace:
		return bm.handleNamespaceBroadcast(ctx, msg, data)
	case fftypes.SystemTopicBroadcastOrganization:
		return bm.handleOrganizationBroadcast(ctx, msg, data)
	case fftypes.SystemTopicBroadcastNode:
		return bm.handleNodeBroadcast(ctx, msg, data)
	default:
		l.Warnf("Unknown topic '%s' for system broadcast ID '%s'", msg.Header.Topic, msg.Header.ID)
	}
	return false, nil
}

func (bm *broadcastManager) getSystemBroadcastPayload(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data, res fftypes.Definition) (valid bool) {
	l := log.L(ctx)
	if len(data) != 1 {
		l.Warnf("Unable to process system broadcast %s - expecting 1 attachement, found %d", msg.Header.ID, len(data))
		return false
	}
	err := json.Unmarshal(data[0].Value, &res)
	if err != nil {
		l.Warnf("Unable to process system broadcast %s - unmarshal failed: %s", msg.Header.ID, err)
		return false
	}
	res.SetBroadcastMessage(msg.Header.ID)
	return true
}

func (bm *broadcastManager) handleNamespaceBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)

	var ns fftypes.Namespace
	valid = bm.getSystemBroadcastPayload(ctx, msg, data, &ns)
	if !valid {
		return false, nil
	}
	if err := ns.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process namespace broadcast %s - validate failed: %s", msg.Header.ID, err)
		return false, nil
	}

	existing, err := bm.database.GetNamespace(ctx, ns.Name)
	if err != nil {
		return false, err // We only return database errors
	}
	if existing != nil && existing.Type != fftypes.NamespaceTypeLocal {
		l.Warnf("Unable to process namespace broadcast %s (name=%s) - duplicate of %v", msg.Header.ID, existing.Name, existing.ID)
		return false, nil
	}

	if err = bm.database.UpsertNamespace(ctx, &ns, true); err != nil {
		return false, err
	}
	return true, nil
}

func (bm *broadcastManager) handleDatatypeBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)

	var dt fftypes.Datatype
	valid = bm.getSystemBroadcastPayload(ctx, msg, data, &dt)
	if !valid {
		return false, nil
	}

	if err = dt.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process datatype broadcast %s - validate failed: %s", msg.Header.ID, err)
		return false, nil
	}

	if err = bm.data.CheckDatatype(ctx, msg.Header.Namespace, &dt); err != nil {
		l.Warnf("Unable to process datatype broadcast %s - schema check: %s", msg.Header.ID, err)
		return false, nil
	}

	existing, err := bm.database.GetDatatypeByName(ctx, dt.Namespace, dt.Name, dt.Version)
	if err != nil {
		return false, err // We only return database errors
	}
	if existing != nil {
		l.Warnf("Unable to process datatype broadcast %s (%s:%s) - duplicate of %v", msg.Header.ID, dt.Namespace, dt, existing.ID)
		return false, nil
	}

	if err = bm.database.UpsertDatatype(ctx, &dt, false); err != nil {
		return false, err
	}

	return true, nil
}

func (bm *broadcastManager) handleOrganizationBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)

	var org fftypes.Organization
	valid = bm.getSystemBroadcastPayload(ctx, msg, data, &org)
	if !valid {
		return false, nil
	}

	if err = org.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process organization broadcast %s - validate failed: %s", msg.Header.ID, err)
		return false, nil
	}

	signingIdentity := org.Identity
	if org.Parent != "" {
		signingIdentity = org.Parent
		parent, err := bm.database.GetOrganization(ctx, org.Parent)
		if err != nil {
			return false, err // We only return database errors
		}
		if parent == nil {
			l.Warnf("Unable to process organization broadcast %s - parent identity not found: %s", msg.Header.ID, org.Parent)
			return false, nil
		}
	}

	id, err := bm.identity.Resolve(ctx, signingIdentity)
	if err != nil {
		l.Warnf("Unable to process organization broadcast %s - resolve identity failed: %s", msg.Header.ID, err)
		return false, nil
	}

	if msg.Header.Author != id.OnChain {
		l.Warnf("Unable to process organization broadcast %s - incorrect signature. Expected=%s Received=%s", msg.Header.ID, id.OnChain, msg.Header.Author)
		return false, nil
	}

	existing, err := bm.database.GetOrganization(ctx, org.Identity)
	if err != nil {
		return false, err // We only return database errors
	}
	if existing != nil {
		if existing.Parent != org.Parent {
			l.Warnf("Unable to process organization broadcast %s - mismatch with existing %v", msg.Header.ID, existing.ID)
			return false, nil
		}
		org.ID = nil // we keep the existing ID
	}

	if err = bm.database.UpsertOrganization(ctx, &org, true); err != nil {
		return false, err
	}

	return true, nil
}

func (bm *broadcastManager) handleNodeBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)

	var node fftypes.Node
	valid = bm.getSystemBroadcastPayload(ctx, msg, data, &node)
	if !valid {
		return false, nil
	}

	if err = node.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process node broadcast %s - validate failed: %s", msg.Header.ID, err)
		return false, nil
	}

	owner, err := bm.database.GetOrganization(ctx, node.Owner)
	if err != nil {
		return false, err // We only return database errors
	}
	if owner == nil {
		l.Warnf("Unable to process node broadcast %s - parent identity not found: %s", msg.Header.ID, node.Owner)
		return false, nil
	}

	id, err := bm.identity.Resolve(ctx, node.Owner)
	if err != nil {
		l.Warnf("Unable to process node broadcast %s - resolve owner identity failed: %s", msg.Header.ID, err)
		return false, nil
	}

	if msg.Header.Author != id.OnChain {
		l.Warnf("Unable to process node broadcast %s - incorrect signature. Expected=%s Received=%s", msg.Header.ID, id.OnChain, msg.Header.Author)
		return false, nil
	}

	existing, err := bm.database.GetNode(ctx, node.Identity)
	if err != nil {
		return false, err // We only return database errors
	}
	if existing != nil {
		if existing.Owner != node.Owner {
			l.Warnf("Unable to process node broadcast %s - mismatch with existing %v", msg.Header.ID, existing.ID)
			return false, nil
		}
		node.ID = nil // we keep the existing ID
	}

	if err = bm.database.UpsertNode(ctx, &node, true); err != nil {
		return false, err
	}

	return true, nil
}
