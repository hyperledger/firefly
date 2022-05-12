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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

func (bm *definitionSender) CreateDatatype(ctx context.Context, datatype *core.Datatype, waitConfirm bool) (*core.Message, error) {

	if !bm.multiparty {
		return nil, i18n.NewError(ctx, coremsgs.MsgActionOnlyValidMultiparty)
	}

	// Validate the input data definition data
	datatype.ID = fftypes.NewUUID()
	datatype.Created = fftypes.Now()
	datatype.Namespace = bm.namespace
	if datatype.Validator == "" {
		datatype.Validator = core.ValidatorTypeJSON
	}
	if err := datatype.Validate(ctx, false); err != nil {
		return nil, err
	}
	datatype.Hash = datatype.Value.Hash()

	// Verify the data type is now all valid, before we broadcast it
	if err := bm.data.CheckDatatype(ctx, datatype); err != nil {
		return nil, err
	}
	msg, err := bm.CreateDefinition(ctx, datatype, core.SystemTagDefineDatatype, waitConfirm)
	if msg != nil {
		datatype.Message = msg.Header.ID
	}
	return msg, err
}
