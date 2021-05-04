// Copyright © 2021 Kaleido, Inc.
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

package engine

import (
	"context"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
)

func (e *engine) GetTransactionById(ctx context.Context, ns, id string) (*fftypes.Transaction, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgInvalidUUID)
	}
	return e.persistence.GetTransactionById(ctx, &u)
}

func (e *engine) GetMessageById(ctx context.Context, ns, id string) (*fftypes.MessageRefsOnly, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgInvalidUUID)
	}
	return e.persistence.GetMessageById(ctx, &u)
}

func (e *engine) GetBatchById(ctx context.Context, ns, id string) (*fftypes.Batch, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgInvalidUUID)
	}
	return e.persistence.GetBatchById(ctx, &u)
}

func (e *engine) GetDataById(ctx context.Context, ns, id string) (*fftypes.Data, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgInvalidUUID)
	}
	return e.persistence.GetDataById(ctx, &u)
}
