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

package syshandlers

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/firefly/internal/broadcast"
	"github.com/hyperledger-labs/firefly/internal/data"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/privatemessaging"
	"github.com/hyperledger-labs/firefly/internal/sysmessaging"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/dataexchange"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/identity"
)

// SystemHandlers interface allows components to call broadcast/private messaging functions internally (without import cycles)
type SystemHandlers interface {
	privatemessaging.GroupManager
	sysmessaging.MessageSender

	HandleSystemBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error)
}

type systemHandlers struct {
	database  database.Plugin
	identity  identity.Plugin
	exchange  dataexchange.Plugin
	data      data.Manager
	broadcast broadcast.Manager
	messaging privatemessaging.Manager
}

func NewSystemHandlers(di database.Plugin, ii identity.Plugin, dx dataexchange.Plugin, dm data.Manager, bm broadcast.Manager, pm privatemessaging.Manager) SystemHandlers {
	return &systemHandlers{
		database:  di,
		identity:  ii,
		exchange:  dx,
		data:      dm,
		broadcast: bm,
		messaging: pm,
	}
}

func (sh *systemHandlers) GetGroupByID(ctx context.Context, id string) (*fftypes.Group, error) {
	return sh.messaging.GetGroupByID(ctx, id)
}

func (sh *systemHandlers) GetGroups(ctx context.Context, filter database.AndFilter) ([]*fftypes.Group, *database.FilterResult, error) {
	return sh.messaging.GetGroups(ctx, filter)
}

func (sh *systemHandlers) ResolveInitGroup(ctx context.Context, msg *fftypes.Message) (*fftypes.Group, error) {
	return sh.messaging.ResolveInitGroup(ctx, msg)
}

func (sh *systemHandlers) EnsureLocalGroup(ctx context.Context, group *fftypes.Group) (ok bool, err error) {
	return sh.messaging.EnsureLocalGroup(ctx, group)
}

func (sh *systemHandlers) SendMessageWithID(ctx context.Context, ns string, unresolved *fftypes.MessageInOut, resolved *fftypes.Message, waitConfirm bool) (out *fftypes.Message, err error) {
	if resolved.Header.Group == nil {
		return sh.broadcast.BroadcastMessageWithID(ctx, ns, unresolved, resolved, waitConfirm)
	}
	return sh.messaging.SendMessageWithID(ctx, ns, unresolved, resolved, waitConfirm)
}

func (sh *systemHandlers) HandleSystemBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)
	l.Infof("Confirming system broadcast '%s' [%s]", msg.Header.Tag, msg.Header.ID)
	switch fftypes.SystemTag(msg.Header.Tag) {
	case fftypes.SystemTagDefineDatatype:
		return sh.handleDatatypeBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineNamespace:
		return sh.handleNamespaceBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineOrganization:
		return sh.handleOrganizationBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineNode:
		return sh.handleNodeBroadcast(ctx, msg, data)
	default:
		l.Warnf("Unknown topic '%s' for system broadcast ID '%s'", msg.Header.Tag, msg.Header.ID)
	}
	return false, nil
}

func (sh *systemHandlers) getSystemBroadcastPayload(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data, res fftypes.Definition) (valid bool) {
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
