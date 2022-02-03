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

package txcommon

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func AddTokenPoolCreateInputs(op *fftypes.Operation, pool *fftypes.TokenPool) {
	op.Input = fftypes.JSONObject{
		"id":        pool.ID.String(),
		"namespace": pool.Namespace,
		"name":      pool.Name,
		"symbol":    pool.Symbol,
		"config":    pool.Config,
	}
}

func RetrieveTokenPoolCreateInputs(ctx context.Context, op *fftypes.Operation, pool *fftypes.TokenPool) (err error) {
	input := &op.Input
	pool.ID, err = fftypes.ParseUUID(ctx, input.GetString("id"))
	if err != nil {
		return err
	}
	pool.Namespace = input.GetString("namespace")
	pool.Name = input.GetString("name")
	if pool.Namespace == "" || pool.Name == "" {
		return fmt.Errorf("namespace or name missing from inputs")
	}
	pool.Symbol = input.GetString("symbol")
	pool.Config = input.GetObject("config")
	return nil
}

func AddTokenTransferInputs(op *fftypes.Operation, transfer *fftypes.TokenTransfer) (err error) {
	var j []byte
	if j, err = json.Marshal(transfer); err == nil {
		err = json.Unmarshal(j, &op.Input)
	}
	return err
}

func RetrieveTokenTransferInputs(ctx context.Context, op *fftypes.Operation, transfer *fftypes.TokenTransfer) (err error) {
	var t fftypes.TokenTransfer
	s := op.Input.String()
	if err = json.Unmarshal([]byte(s), &t); err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, s)
	}
	if t.LocalID == nil {
		return i18n.NewError(ctx, i18n.MsgInvalidUUID)
	}
	// The LocalID is the only thing that needs to be read back out when processing an event
	// (everything else should be unpacked from the event)
	transfer.LocalID = t.LocalID
	return nil
}
