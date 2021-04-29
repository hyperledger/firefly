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

package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/apitypes"
	"github.com/kaleido-io/firefly/internal/persistence"
)

type Postgres struct {
	sqlCommon persistence.PeristenceInterface
}

func (e *Postgres) ConfigInterface() interface{} { return &Config{} }

func (e *Postgres) Init(ctx context.Context, conf interface{}, events persistence.Events) (*persistence.Capabilities, error) {
	return &persistence.Capabilities{}, nil
}

func (e *Postgres) UpsertMessage(ctx context.Context, message *apitypes.MessageRefsOnly) (id uuid.UUID, err error) {
	return e.sqlCommon.UpsertMessage(ctx, message)
}

func (e *Postgres) GetMessageById(ctx context.Context, id uuid.UUID) (message *apitypes.MessageRefsOnly, err error) {
	return e.sqlCommon.GetMessageById(ctx, id)
}

func (e *Postgres) GetMessages(ctx context.Context, filter *persistence.MessageFilter, skip, limit uint) (message *apitypes.MessageRefsOnly, err error) {
	return e.sqlCommon.GetMessages(ctx, filter, skip, limit)
}
