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

	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
)

func (or *orchestrator) GetConfigRecord(ctx context.Context, key string) (*fftypes.ConfigRecord, error) {
	return or.database.GetConfigRecord(ctx, key)
}

func (or *orchestrator) GetConfigRecords(ctx context.Context, filter database.AndFilter) ([]*fftypes.ConfigRecord, error) {
	return or.database.GetConfigRecords(ctx, filter)
}

func (or *orchestrator) PutConfigRecord(ctx context.Context, key string, value fftypes.Byteable) (outputValue fftypes.Byteable, err error) {
	configRecord := &fftypes.ConfigRecord{
		Key:   key,
		Value: value,
	}
	if err := or.database.UpsertConfigRecord(ctx, configRecord, true); err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		or.cancelCtx()
	}()

	return value, nil
}

func (or *orchestrator) DeleteConfigRecord(ctx context.Context, key string) (err error) {
	return or.database.DeleteConfigRecord(ctx, key)
}
