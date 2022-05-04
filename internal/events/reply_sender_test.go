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
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/broadcastmocks"
	"github.com/hyperledger/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/mock"
)

func TestSendReplyBroadcastFail(t *testing.T) {
	ed, cancel := newTestEventDispatcher(&subscription{
		definition: &fftypes.Subscription{},
	})
	defer cancel()

	mms := &sysmessagingmocks.MessageSender{}
	mbm := ed.broadcast.(*broadcastmocks.Manager)
	mbm.On("NewBroadcast", "ns1", mock.Anything).Return(mms)
	mms.On("Send", context.Background()).Return(fmt.Errorf("pop"))

	ed.sendReply(context.Background(), &fftypes.Event{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}, &fftypes.MessageInOut{})

	mbm.AssertExpectations(t)
	mms.AssertExpectations(t)
}

func TestSendReplyPrivateFail(t *testing.T) {
	ed, cancel := newTestEventDispatcher(&subscription{
		definition: &fftypes.Subscription{},
	})
	defer cancel()

	mms := &sysmessagingmocks.MessageSender{}
	mpm := ed.messaging.(*privatemessagingmocks.Manager)
	mpm.On("NewMessage", "ns1", mock.Anything).Return(mms)
	mms.On("Send", context.Background()).Return(fmt.Errorf("pop"))

	ed.sendReply(context.Background(), &fftypes.Event{
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
	mms.AssertExpectations(t)
}

func TestSendReplyPrivateOk(t *testing.T) {
	ed, cancel := newTestEventDispatcher(&subscription{
		definition: &fftypes.Subscription{},
	})
	defer cancel()

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			Group: fftypes.NewRandB32(),
		},
	}

	mms := &sysmessagingmocks.MessageSender{}
	mpm := ed.messaging.(*privatemessagingmocks.Manager)
	mpm.On("NewMessage", "ns1", mock.Anything).Return(mms)
	mms.On("Send", context.Background()).Return(nil)

	ed.sendReply(context.Background(), &fftypes.Event{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}, &fftypes.MessageInOut{
		Message: *msg,
	})

	mpm.AssertExpectations(t)
	mms.AssertExpectations(t)
}
