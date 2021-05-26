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

func (bm *broadcastManager) HandleSystemBroadcast(ctx context.Context, msg *fftypes.Message) error {
	l := log.L(ctx)
	switch msg.Header.Topic {
	case fftypes.SystemTopicBroadcastDatatype:
		return bm.handleDatatypeBroadcast(ctx, msg)
	case fftypes.SystemTopicBroadcastNamespace:
		return bm.handleNamespaceBroadcast(ctx, msg)
	default:
		l.Warnf("Unknown topic '%s' for system broadcast ID '%s'", msg.Header.Topic, msg.Header.ID)
	}
	return nil
}

func (bm *broadcastManager) getSystemBroadcastPayload(ctx context.Context, msg *fftypes.Message, res interface{}) (valid bool, err error) {
	l := log.L(ctx)
	data, allFound, err := bm.data.GetMessageData(ctx, msg)
	if err != nil {
		return false, err // only database errors are returned as an error (driving retry until we succeed)
	}
	if !allFound {
		l.Warnf("Unable to process system broadcast %s - missing data", msg.Header.ID)
		return false, nil
	}
	if len(data) != 1 {
		l.Warnf("Unable to process system broadcast %s - expecting 1 attachement, found %d", msg.Header.ID, len(data))
		return false, nil
	}
	err = json.Unmarshal(data[0].Value, &res)
	if err != nil {
		l.Warnf("Unable to process system broadcast %s - unmarshal failed: %s", msg.Header.ID, err)
		return false, nil
	}
	return true, nil
}

func (bm *broadcastManager) handleNamespaceBroadcast(ctx context.Context, msg *fftypes.Message) error {
	l := log.L(ctx)

	var ns fftypes.Namespace
	valid, err := bm.getSystemBroadcastPayload(ctx, msg, &ns)
	if !valid || err != nil {
		return err
	}
	if err = ns.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process namespace broadcast %s - validate failed: %s", msg.Header.ID, err)
		return nil
	}

	existing, err := bm.database.GetNamespace(ctx, ns.Name)
	if err != nil {
		return err // We only return database errors
	}
	if existing != nil && existing.Type != fftypes.NamespaceTypeLocal {
		l.Warnf("Unable to process namespace broadcast %s (name=%s) - duplicate of %v", msg.Header.ID, existing.Name, existing.ID)
		return nil
	}

	return bm.database.UpsertNamespace(ctx, &ns, true)
}

func (bm *broadcastManager) handleDatatypeBroadcast(ctx context.Context, msg *fftypes.Message) error {
	l := log.L(ctx)

	var dt fftypes.Datatype
	valid, err := bm.getSystemBroadcastPayload(ctx, msg, &dt)
	if !valid || err != nil {
		return err
	}

	if err = dt.Validate(ctx, true); err != nil {
		l.Warnf("Unable to process data broadcast %s - validate failed: %s", msg.Header.ID, err)
		return nil
	}

	if err = bm.data.CheckDatatype(ctx, &dt); err != nil {
		l.Warnf("Unable to process datatype broadcast %s - schema check: %s", msg.Header.ID, err)
		return nil
	}

	existing, err := bm.database.GetDatatypeByName(ctx, dt.Namespace, dt.Name, dt.Version)
	if err != nil {
		return err // We only return database errors
	}
	if existing != nil {
		l.Warnf("Unable to process datatype broadcast %s (%s:%s) - duplicate of %v", msg.Header.ID, dt.Namespace, dt, existing.ID)
		return nil
	}

	return bm.database.UpsertDatatype(ctx, &dt, false)
}
