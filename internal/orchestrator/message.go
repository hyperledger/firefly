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

package orchestrator

import (
	"context"

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (or *orchestrator) RequestReply(ctx context.Context, ns string, msg *fftypes.MessageInOut) (reply *fftypes.MessageInOut, err error) {
	if msg.Header.Group == nil && (msg.Group == nil || len(msg.Group.Members) == 0) {
		return nil, i18n.NewError(ctx, i18n.MsgRequestMustBePrivate)
	}
	return or.PrivateMessaging().RequestReply(ctx, ns, msg)
}
