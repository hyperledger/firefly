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

package data

import (
	"context"
	"fmt"
	"time"

	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/karlseguin/ccache"
)

type Manager interface {
	GetValidator(ctx context.Context, data *fftypes.Data) (Validator, error)
}

type dataManager struct {
	ctx               context.Context
	database          database.Plugin
	validatorCache    *ccache.Cache
	validatorCacheTTL time.Duration
}

func NewDataManager(ctx context.Context, di database.Plugin) (Manager, error) {
	if di == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	dm := &dataManager{
		ctx:               ctx,
		database:          di,
		validatorCacheTTL: config.GetDuration(config.ValidatorCacheTTL),
	}
	dm.validatorCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(config.ValidatorCacheSize)),
	)
	return dm, nil
}

func (dm *dataManager) GetValidator(ctx context.Context, data *fftypes.Data) (Validator, error) {
	if data.Validator == "" {
		data.Validator = fftypes.ValidatorTypeJSON
	}
	if data.Validator != fftypes.ValidatorTypeJSON {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownValidatorType, data.Validator)
	}

	if data.Datatype == nil || data.Datatype.Name == "" || data.Datatype.Version == "" {
		return nil, i18n.NewError(ctx, i18n.MsgDatatypeNotFound, data.Datatype)
	}

	key := fmt.Sprintf("%s:%s:%s", data.Namespace, data.Validator, data.Datatype)
	validator, err := dm.validatorCache.Fetch(key, dm.validatorCacheTTL, func() (interface{}, error) {
		datatype, err := dm.database.GetDatatypeByName(dm.ctx, data.Namespace, data.Datatype.Name, data.Datatype.Version)
		if err != nil {
			return nil, err
		}
		if datatype == nil {
			return nil, i18n.NewError(ctx, i18n.MsgDatatypeNotFound, data.Datatype.Name)
		}
		return newJSONValidator(ctx, datatype)
	})
	if err != nil {
		return nil, err
	}
	return validator.Value().(Validator), err
}
