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

package txcommon

import (
	"context"

	"github.com/hyperledger/firefly/pkg/fftypes"
)

func AddTokenTransferInputs(op *fftypes.Operation, transfer *fftypes.TokenTransfer) {
	op.Input = fftypes.JSONObject{
		"id": transfer.LocalID.String(),
	}
}

func RetrieveTokenTransferInputs(ctx context.Context, op *fftypes.Operation, transfer *fftypes.TokenTransfer) (err error) {
	if transfer.LocalID, err = fftypes.ParseUUID(ctx, op.Input.GetString("id")); err != nil {
		return err
	}
	return nil
}
