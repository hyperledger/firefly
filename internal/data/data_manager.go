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
	CheckDatatype(ctx context.Context, ns string, datatype *fftypes.Datatype) error
	GetValidator(ctx context.Context, data *fftypes.Data) (Validator, error)
	GetMessageData(ctx context.Context, msg *fftypes.Message, withValue bool) (data []*fftypes.Data, foundAll bool, err error)
	ResolveInputData(ctx context.Context, ns string, inData fftypes.InputData) (fftypes.DataRefs, error)
}

type dataManager struct {

	// Note that we don't store a context in dataManager, as it doesn't currently have long-running
	// background tasks. All of the APIs for actions take a context, which might include a database
	// transaction - so it's important that we use that passed context for all of those synchronous calls.
	// If a context is introduce for long lived tasks, to avoid accidental use of it on API calls,
	// that should be stored on a sub structure associated with that long-lived task.

	database          database.Plugin
	validatorCache    *ccache.Cache
	validatorCacheTTL time.Duration
}

func NewDataManager(ctx context.Context, di database.Plugin) (Manager, error) {
	if di == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	dm := &dataManager{
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

func (dm *dataManager) CheckDatatype(ctx context.Context, ns string, datatype *fftypes.Datatype) error {
	_, err := newJSONValidator(ctx, ns, datatype)
	return err
}

func (dm *dataManager) GetValidator(ctx context.Context, data *fftypes.Data) (Validator, error) {
	return dm.getValidatorForDatatype(ctx, data.Namespace, data.Validator, data.Datatype)
}

func (dm *dataManager) getValidatorForDatatype(ctx context.Context, ns string, validator fftypes.ValidatorType, datatypeRef *fftypes.DatatypeRef) (Validator, error) {
	if validator == "" {
		validator = fftypes.ValidatorTypeJSON
	}
	if validator != fftypes.ValidatorTypeJSON {
		return nil, i18n.NewError(ctx, i18n.MsgUnknownValidatorType, validator)
	}

	if datatypeRef == nil || datatypeRef.Name == "" || datatypeRef.Version == "" {
		return nil, i18n.NewError(ctx, i18n.MsgDatatypeNotFound, datatypeRef)
	}

	key := fmt.Sprintf("%s:%s:%s", ns, validator, datatypeRef)
	v, err := dm.validatorCache.Fetch(key, dm.validatorCacheTTL, func() (interface{}, error) {
		datatype, err := dm.database.GetDatatypeByName(ctx, ns, datatypeRef.Name, datatypeRef.Version)
		if err != nil {
			return nil, err
		}
		if datatype == nil {
			return nil, i18n.NewError(ctx, i18n.MsgDatatypeNotFound, datatypeRef.Name)
		}
		return newJSONValidator(ctx, ns, datatype)
	})
	if err != nil {
		return nil, err
	}
	return v.Value().(Validator), err
}

// GetMessageData looks for all the data attached to the message.
// It only returns persistence errors.
// For all cases where the data is not found (or the hashes mismatch)
func (dm *dataManager) GetMessageData(ctx context.Context, msg *fftypes.Message, withValue bool) (data []*fftypes.Data, foundAll bool, err error) {
	// Load all the data - must all be present for us to send
	data = make([]*fftypes.Data, 0, len(msg.Data))
	foundAll = true
	for i, dataRef := range msg.Data {
		d, err := dm.resolveRef(ctx, msg.Header.Namespace, dataRef, withValue)
		if err != nil {
			return nil, false, err
		}
		if d == nil {
			log.L(ctx).Warnf("Message %v data %d mising", msg.Header.ID, i)
			foundAll = false
			continue
		}
		data = append(data, d)
	}
	return data, foundAll, nil
}

func (dm *dataManager) resolveRef(ctx context.Context, ns string, dataRef *fftypes.DataRef, withValue bool) (*fftypes.Data, error) {
	if dataRef == nil || dataRef.ID == nil {
		log.L(ctx).Warnf("data is nil")
		return nil, nil
	}
	d, err := dm.database.GetDataByID(ctx, dataRef.ID, withValue)
	if err != nil {
		return nil, err
	}
	switch {
	case d == nil || d.Namespace != ns:
		log.L(ctx).Warnf("Data %s not found in namespace %s", dataRef.ID, ns)
		return nil, nil
	case d.Hash == nil || (dataRef.Hash != nil && *d.Hash != *dataRef.Hash):
		log.L(ctx).Warnf("Data hash does not match. Hash=%v Expected=%v", d.Hash, dataRef.Hash)
		return nil, nil
	default:
		return d, nil
	}
}

func (dm *dataManager) validateAndStore(ctx context.Context, ns string, value *fftypes.DataRefOrValue) (*fftypes.DataRef, error) {

	// If a datatype is specified, we need to verify the payload conforms
	if value.Datatype != nil {
		v, err := dm.getValidatorForDatatype(ctx, ns, value.Validator, value.Datatype)
		if err != nil {
			return nil, err
		}
		err = v.ValidateValue(ctx, value.Value, nil)
		if err != nil {
			return nil, err
		}
	}

	// Ok, we're good to generate the full data payload and save it
	data := &fftypes.Data{
		Validator: value.Validator,
		Datatype:  value.Datatype,
		Namespace: ns,
		Hash:      value.Hash,
		Value:     value.Value,
	}
	err := data.Seal(ctx)
	if err == nil {
		err = dm.database.UpsertData(ctx, data, false, false)
	}
	if err != nil {
		return nil, err
	}

	// Return a ref to the newly saved data
	return &fftypes.DataRef{
		ID:   data.ID,
		Hash: data.Hash,
	}, nil
}

func (dm *dataManager) ResolveInputData(ctx context.Context, ns string, inData fftypes.InputData) (refs fftypes.DataRefs, err error) {

	refs = make(fftypes.DataRefs, len(inData))
	for i, dataOrValue := range inData {
		switch {
		case dataOrValue.ID != nil:
			// If an ID is supplied, then it must be a reference to existing data
			d, err := dm.resolveRef(ctx, ns, &dataOrValue.DataRef, false /* do not need the value */)
			if err != nil {
				return nil, err
			}
			if d == nil {
				return nil, i18n.NewError(ctx, i18n.MsgDataReferenceUnresolvable, i)
			}
			refs[i] = &fftypes.DataRef{
				ID:   d.ID,
				Hash: d.Hash,
			}
		case dataOrValue.Value != nil:
			// We've got a Value, so we can validate + store it
			if refs[i], err = dm.validateAndStore(ctx, ns, dataOrValue); err != nil {
				return nil, err
			}
		default:
			// We have neither - this must be a mistake
			return nil, i18n.NewError(ctx, i18n.MsgDataMissing, i)
		}
	}
	return refs, nil
}
