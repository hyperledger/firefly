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

	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (bm *broadcastManager) BroadcastTokenPool(ctx context.Context, ns string, pool *fftypes.TokenPoolAnnouncement, waitConfirm bool) (msg *fftypes.Message, err error) {
	if err := pool.Pool.Validate(ctx); err != nil {
		return nil, err
	}
	if err := bm.data.VerifyNamespaceExists(ctx, pool.Pool.Namespace); err != nil {
		return nil, err
	}

	msg, err = bm.BroadcastDefinitionAsNode(ctx, ns, pool, fftypes.SystemTagDefinePool, waitConfirm)
	if msg != nil {
		pool.Pool.Message = msg.Header.ID
	}
	return msg, err
}
