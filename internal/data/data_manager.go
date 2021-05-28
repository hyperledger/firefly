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
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/fftypes"
	"github.com/karlseguin/ccache"
)

type Manager interface {
	CheckDatatype(ctx context.Context, datatype *fftypes.Datatype) error
	GetValidator(ctx context.Context, data *fftypes.Data) (Validator, error)
	GetMessageData(ctx context.Context, msg *fftypes.Message) (data []*fftypes.Data, foundAll bool, err error)
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

func (dm *dataManager) CheckDatatype(ctx context.Context, datatype *fftypes.Datatype) error {
	_, err := newJSONValidator(ctx, datatype)
	return err
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
	v := validator.Value().(*jsonValidator)
	log.L(ctx).Debugf("Found JSON schema validator for %s: %v", key, v.id)
	return v, err
}

// GetMessageData looks for all the data attached to the message.
// It only returns persistence errors.
// For all cases where the data is not found (or the hashes mismatch)
func (dm *dataManager) GetMessageData(ctx context.Context, msg *fftypes.Message) (data []*fftypes.Data, foundAll bool, err error) {
	// Load all the data - must all be present for us to send
	data = make([]*fftypes.Data, 0, len(msg.Data))
	foundAll = true
	for i, dataRef := range msg.Data {
		if dataRef == nil || dataRef.ID == nil {
			log.L(ctx).Warnf("Message %v data %d is nil", msg.Header.ID, i)
			foundAll = false
			continue
		}
		d, err := dm.database.GetDataByID(ctx, dataRef.ID)
		if err != nil {
			return nil, false, err
		}
		switch {
		case d == nil:
			log.L(ctx).Warnf("Message %v missing data %v", msg.Header.ID, dataRef.ID)
			foundAll = false
		case d.Hash == nil || dataRef.Hash == nil || *d.Hash != *dataRef.Hash:
			log.L(ctx).Warnf("Message %v data %d hash does not match. Hash=%v Expected=%v", msg.Header.ID, i, d.Hash, dataRef.Hash)
			foundAll = false
		default:
			data = append(data, d)
		}
	}
	return data, foundAll, nil
}
