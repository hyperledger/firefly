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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/mock"
)

func TestSendReplyBroadcastFail(t *testing.T) {
	sh := newTestSystemHandlers(t)
	mbs := &broadcastmocks.Broadcast{}
	mbm := sh.broadcast.(*broadcastmocks.Manager)
	mbm.On("NewBroadcast", "ns1", mock.Anything).Return(mbs)
	mbs.On("Send", context.Background()).Return(fmt.Errorf("pop"))

	sh.SendReply(context.Background(), &fftypes.Event{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}, &fftypes.MessageInOut{})

	mbm.AssertExpectations(t)
	mbs.AssertExpectations(t)
}

func TestSendReplyPrivateFail(t *testing.T) {
	sh := newTestSystemHandlers(t)
	mps := &privatemessagingmocks.PrivateMessage{}
	mpm := sh.messaging.(*privatemessagingmocks.Manager)
	mpm.On("NewMessage", "ns1", mock.Anything).Return(mps)
	mps.On("Send", context.Background()).Return(fmt.Errorf("pop"))

	sh.SendReply(context.Background(), &fftypes.Event{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}, &fftypes.MessageInOut{
		Message: fftypes.Message{
			Header: fftypes.MessageHeader{
				Group: fftypes.NewRandB32(),
			},
		},
	})

	mpm.AssertExpectations(t)
	mps.AssertExpectations(t)
}

func TestSendReplyPrivateOk(t *testing.T) {
	sh := newTestSystemHandlers(t)

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Group: fftypes.NewRandB32(),
		},
	}

	mps := &privatemessagingmocks.PrivateMessage{}
	mpm := sh.messaging.(*privatemessagingmocks.Manager)
	mpm.On("NewMessage", "ns1", mock.Anything).Return(mps)
	mps.On("Send", context.Background()).Return(nil)

	sh.SendReply(context.Background(), &fftypes.Event{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}, &fftypes.MessageInOut{
		Message: *msg,
	})

	mpm.AssertExpectations(t)
	mps.AssertExpectations(t)
}
