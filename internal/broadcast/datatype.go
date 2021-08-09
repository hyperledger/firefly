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

	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (bm *broadcastManager) BroadcastDatatype(ctx context.Context, ns string, datatype *fftypes.Datatype, waitConfirm bool) (*fftypes.Message, error) {

	// Validate the input data definition data
	datatype.ID = fftypes.NewUUID()
	datatype.Created = fftypes.Now()
	datatype.Namespace = ns
	if datatype.Validator == "" {
		datatype.Validator = fftypes.ValidatorTypeJSON
	}
	if err := datatype.Validate(ctx, false); err != nil {
		return nil, err
	}
	if err := bm.data.VerifyNamespaceExists(ctx, datatype.Namespace); err != nil {
		return nil, err
	}
	datatype.Hash = datatype.Value.Hash()

	// Verify the data type is now all valid, before we broadcast it
	if err := bm.data.CheckDatatype(ctx, ns, datatype); err != nil {
		return nil, err
	}
	msg, err := bm.broadcastDefinitionAsNode(ctx, datatype, fftypes.SystemTagDefineDatatype, waitConfirm)
	if msg != nil {
		datatype.Message = msg.Header.ID
	}
	return msg, err
}
