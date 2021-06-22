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

package events

import (
	"context"

	"github.com/hyperledger-labs/firefly/internal/broadcast"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/internal/privatemessaging"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

type replySender struct {
	broadcast broadcast.Manager
	messaging privatemessaging.Manager
}

func (rs *replySender) sendReply(ctx context.Context, event *fftypes.Event, reply *fftypes.MessageInput) {
	var err error
	var msg *fftypes.Message
	if reply.Header.Group != nil {
		msg, err = rs.messaging.SendMessage(ctx, event.Namespace, reply)
	} else {
		msg, err = rs.broadcast.BroadcastMessage(ctx, event.Namespace, reply)
	}
	if err != nil {
		log.L(ctx).Errorf("Failed to send reply: %s", err)
	} else {
		log.L(ctx).Infof("Sent reply '%s' (%s) to event '%s'", msg.Header.ID, msg.Header.Type, event.ID)
	}
}
