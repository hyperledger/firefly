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

package defsender

import (
	"context"

	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (bm *definitionSender) BroadcastNamespace(ctx context.Context, ns *fftypes.Namespace, waitConfirm bool) (*fftypes.Message, error) {

	// Validate the input data definition data
	ns.ID = fftypes.NewUUID()
	ns.Created = fftypes.Now()
	ns.Type = fftypes.NamespaceTypeBroadcast
	if err := ns.Validate(ctx, false); err != nil {
		return nil, err
	}
	msg, err := bm.BroadcastDefinitionAsNode(ctx, fftypes.SystemNamespace, ns, fftypes.SystemTagDefineNamespace, waitConfirm)
	if msg != nil {
		ns.Message = msg.Header.ID
	}
	return msg, err
}
