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

	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (bm *broadcastManager) HandleSystemBroadcast(ctx context.Context, msg *fftypes.Message, data []*fftypes.Data) (valid bool, err error) {
	l := log.L(ctx)
	l.Infof("Confirming system broadcast '%s' [%s]", msg.Header.Tag, msg.Header.ID)
	switch fftypes.SystemTag(msg.Header.Tag) {
	case fftypes.SystemTagDefineDatatype:
		return bm.handleDatatypeBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineNamespace:
		return bm.handleNamespaceBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineOrganization:
		return bm.handleOrganizationBroadcast(ctx, msg, data)
	case fftypes.SystemTagDefineNode:
		return bm.handleNodeBroadcast(ctx, msg, data)
	default:
		l.Warnf("Unknown topic '%s' for system broadcast ID '%s'", msg.Header.Tag, msg.Header.ID)
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
