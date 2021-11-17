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

package definitions

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/assets"
	"github.com/hyperledger/firefly/internal/broadcast"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/privatemessaging"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

// DefinitionHandlers interface allows components to call broadcast/private messaging functions internally (without import cycles)
type DefinitionHandlers interface {
	privatemessaging.GroupManager

	HandleDefinitionBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error)
	SendReply(ctx context.Context, event *fftypes.Event, reply *fftypes.MessageInOut)
}

type definitionHandlers struct {
	database  database.Plugin
	exchange  dataexchange.Plugin
	data      data.Manager
	broadcast broadcast.Manager
	messaging privatemessaging.Manager
	assets    assets.Manager
	txhelper  txcommon.Helper
}

func NewDefinitionHandlers(di database.Plugin, dx dataexchange.Plugin, dm data.Manager, bm broadcast.Manager, pm privatemessaging.Manager, am assets.Manager) DefinitionHandlers {
	return &definitionHandlers{
		database:  di,
		exchange:  dx,
		data:      dm,
		broadcast: bm,
		messaging: pm,
		assets:    am,
		txhelper:  txcommon.NewTransactionHelper(di),
	}
}

func (dh *definitionHandlers) GetGroupByID(ctx context.Context, id string) (*fftypes.Group, error) {
	return dh.messaging.GetGroupByID(ctx, id)
}

func (dh *definitionHandlers) GetGroups(ctx context.Context, filter database.AndFilter) ([]*fftypes.Group, *database.FilterResult, error) {
	return dh.messaging.GetGroups(ctx, filter)
}

func (dh *definitionHandlers) ResolveInitGroup(ctx context.Context, msg *fftypes.Message) (*fftypes.Group, error) {
	return dh.messaging.ResolveInitGroup(ctx, msg)
}

func (dh *definitionHandlers) EnsureLocalGroup(ctx context.Context, group *fftypes.Group) (ok bool, err error) {
	return dh.messaging.EnsureLocalGroup(ctx, group)
}

func (dh *definitionHandlers) HandleDefinitionBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)
	l.Infof("Confirming system broadcast '%s' [%s]", msg.Header.Tag, msg.Header.ID)
	switch fftypes.SystemTag(msg.Header.Tag) {
	case fftypes.SystemTagDefineDatatype:
		return dh.handleDatatypeBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineNamespace:
		return dh.handleNamespaceBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineOrganization:
		return dh.handleOrganizationBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineNode:
		return dh.handleNodeBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefinePool:
		return dh.handleTokenPoolBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineContractInterface:
		return dh.handleContractInterfaceBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineContractAPI:
		return dh.handleContractAPIBroadcast(ctx, msg, data)
	default:
		l.Debugf("Unknown topic '%s' for system broadcast or definition ID '%s'", msg.Header.Tag, msg.Header.ID)
	}
	return false, nil
}

func (dh *definitionHandlers) getSystemBroadcastPayload(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data, res fftypes.Definition) (valid bool) {
	l := log.L(ctx)
	if len(data) != 1 {
		l.Warnf("Unable to process system broadcast %s - expecting 1 attachment, found %d", msg.Header.ID, len(data))
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
