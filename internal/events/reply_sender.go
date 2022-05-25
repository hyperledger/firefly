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

package events

import (
	"context"

	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/pkg/core"
)

func (ed *eventDispatcher) sendReply(ctx context.Context, event *core.Event, reply *core.MessageInOut) {
	var err error
	if reply.Header.Group != nil {
		err = ed.messaging.NewMessage(event.Namespace, reply).Send(ctx)
	} else {
		err = ed.broadcast.NewBroadcast(event.Namespace, reply).Send(ctx)
	}
	if err != nil {
		log.L(ctx).Errorf("Failed to send reply: %s", err)
	} else {
		log.L(ctx).Infof("Sent reply %s:%s (%s) cid=%s to event '%s'", reply.Header.Namespace, reply.Header.ID, reply.Header.Type, reply.Header.CID, event.ID)
	}
}
