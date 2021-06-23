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

	"github.com/hyperledger-labs/firefly/mocks/broadcastmocks"
	"github.com/hyperledger-labs/firefly/mocks/privatemessagingmocks"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/mock"
)

func TestSendreplyFail(t *testing.T) {
	mbm := &broadcastmocks.Manager{}
	mpm := &privatemessagingmocks.Manager{}
	rs := &replySender{
		broadcast: mbm,
		messaging: mpm,
	}
	mbm.On("BroadcastMessage", mock.Anything, "ns1", mock.Anything).Return(nil, fmt.Errorf("pop"))
	rs.sendReply(context.Background(), &fftypes.Event{
		ID:        fftypes.NewUUID(),
		Namespace: "ns1",
	}, &fftypes.MessageInput{})
	mbm.AssertExpectations(t)
}
