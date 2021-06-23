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
	"io"
	"time"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/dataexchange"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/hyperledger-labs/firefly/pkg/publicstorage"
	"github.com/karlseguin/ccache"
)

type Manager interface {
	CheckDatatype(ctx context.Context, ns string, datatype *fftypes.Datatype) error
	ValidateAll(ctx context.Context, data []*fftypes.Data) (valid bool, err error)
	GetMessageData(ctx context.Context, msg *fftypes.Message, withValue bool) (data []*fftypes.Data, foundAll bool, err error)
	ResolveInputDataPrivate(ctx context.Context, ns string, inData fftypes.InputData) (fftypes.DataRefs, error)
	ResolveInputDataBroadcast(ctx context.Context, ns string, inData fftypes.InputData) (fftypes.DataRefs, []*fftypes.DataAndBlob, error)
	VerifyNamespaceExists(ctx context.Context, ns string) error

	UploadJSON(ctx context.Context, ns string, inData *fftypes.DataRefOrValue) (*fftypes.Data, error)
	UploadBLOB(ctx context.Context, ns string, inData *fftypes.DataRefOrValue, blob *fftypes.Multipart, autoMeta bool) (*fftypes.Data, error)
	CopyBlobPStoDX(ctx context.Context, data *fftypes.Data) (blob *fftypes.Blob, err error)
	DownloadBLOB(ctx context.Context, ns, dataID string) (io.ReadCloser, error)
}

type dataManager struct {
	blobStore

	database          database.Plugin
	publicstorage     publicstorage.Plugin
	exchange          dataexchange.Plugin
	validatorCache    *ccache.Cache
	validatorCacheTTL time.Duration
}

func NewDataManager(ctx context.Context, di database.Plugin, pi publicstorage.Plugin, dx dataexchange.Plugin) (Manager, error) {
	if di == nil || pi == nil || dx == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	dm := &dataManager{
		database:          di,
		publicstorage:     pi,
		exchange:          dx,
		validatorCacheTTL: config.GetDuration(config.ValidatorCacheTTL),
		blobStore: blobStore{
			database:      di,
			publicstorage: pi,
			exchange:      dx,
		},
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

func (dm *dataManager) VerifyNamespaceExists(ctx context.Context, ns string) error {
	err := fftypes.ValidateFFNameField(ctx, ns, "namespace")
	if err != nil {
		return err
	}
	namespace, err := dm.database.GetNamespace(ctx, ns)
	if err != nil {
		return err
	}
	if namespace == nil {
		return i18n.NewError(ctx, i18n.MsgNamespaceNotExist)
	}
	return nil
}

// getValidatorForDatatype only returns database errors - not found (of all kinds) is a nil
func (dm *dataManager) getValidatorForDatatype(ctx context.Context, ns string, validator fftypes.ValidatorType, datatypeRef *fftypes.DatatypeRef) (Validator, error) {
	if validator == "" {
		validator = fftypes.ValidatorTypeJSON
	}

	if ns == "" || datatypeRef == nil || datatypeRef.Name == "" || datatypeRef.Version == "" {
		log.L(ctx).Warnf("Invalid datatype reference '%s:%s:%s'", validator, ns, datatypeRef)
		return nil, nil
	}

	key := fmt.Sprintf("%s:%s:%s", validator, ns, datatypeRef)
	if cached := dm.validatorCache.Get(key); cached != nil {
		cached.Extend(dm.validatorCacheTTL)
		return cached.Value().(Validator), nil
	}

	datatype, err := dm.database.GetDatatypeByName(ctx, ns, datatypeRef.Name, datatypeRef.Version)
	if err != nil {
		return nil, err
	}
	if datatype == nil {
		return nil, nil
	}
	v, err := newJSONValidator(ctx, ns, datatype)
	if err != nil {
		log.L(ctx).Errorf("Invalid validator stored for '%s:%s:%s': %s", validator, ns, datatypeRef, err)
		return nil, nil
	}

	dm.validatorCache.Set(key, v, dm.validatorCacheTTL)
	return v, err
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

func (dm *dataManager) ValidateAll(ctx context.Context, data []*fftypes.Data) (valid bool, err error) {
	for _, d := range data {
		if d.Datatype != nil {
			v, err := dm.getValidatorForDatatype(ctx, d.Namespace, d.Validator, d.Datatype)
			if err != nil {
				return false, err
			}
			if v == nil {
				log.L(ctx).Errorf("Datatype %s:%s:%s not found", d.Validator, d.Namespace, d.Datatype)
				return false, err
			}
			err = v.ValidateValue(ctx, d.Value, d.Hash)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
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

func (dm *dataManager) checkValidatorType(ctx context.Context, validator fftypes.ValidatorType) error {
	switch validator {
	case "", fftypes.ValidatorTypeJSON:
		return nil
	default:
		return i18n.NewError(ctx, i18n.MsgUnknownValidatorType, validator)
	}
}

func (dm *dataManager) resolveBlob(ctx context.Context, blobRef *fftypes.BlobRef) (*fftypes.Blob, error) {
	if blobRef != nil && blobRef.Hash != nil {
		blob, err := dm.database.GetBlobMatchingHash(ctx, blobRef.Hash)
		if err != nil {
			return nil, err
		}
		if blob == nil {
			return nil, i18n.NewError(ctx, i18n.MsgBlobNotFound, blobRef.Hash)
		}
		return blob, nil
	}
	return nil, nil
}

func (dm *dataManager) validateAndStore(ctx context.Context, ns string, validator fftypes.ValidatorType, datatype *fftypes.DatatypeRef, value fftypes.Byteable, blobRef *fftypes.BlobRef) (data *fftypes.Data, blob *fftypes.Blob, err error) {
	// If a datatype is specified, we need to verify the payload conforms
	if datatype != nil {
		if err := dm.checkValidatorType(ctx, validator); err != nil {
			return nil, nil, err
		}
		if datatype == nil || datatype.Name == "" || datatype.Version == "" {
			return nil, nil, i18n.NewError(ctx, i18n.MsgDatatypeNotFound, datatype)
		}
		v, err := dm.getValidatorForDatatype(ctx, ns, validator, datatype)
		if err != nil {
			return nil, nil, err
		}
		if v == nil {
			return nil, nil, i18n.NewError(ctx, i18n.MsgDatatypeNotFound, datatype)
		}
		err = v.ValidateValue(ctx, value, nil)
		if err != nil {
			return nil, nil, err
		}
	} else {
		validator = ""
	}

	if blob, err = dm.resolveBlob(ctx, blobRef); err != nil {
		return nil, nil, err
	}

	// Ok, we're good to generate the full data payload and save it
	data = &fftypes.Data{
		Validator: validator,
		Datatype:  datatype,
		Namespace: ns,
		Value:     value,
		Blob:      blobRef,
	}
	err = data.Seal(ctx)
	if err == nil {
		err = dm.database.UpsertData(ctx, data, false, false)
	}
	if err != nil {
		return nil, nil, err
	}
	return data, blob, nil
}

func (dm *dataManager) validateAndStoreInlined(ctx context.Context, ns string, value *fftypes.DataRefOrValue) (*fftypes.Data, *fftypes.Blob, *fftypes.DataRef, error) {
	data, blob, err := dm.validateAndStore(ctx, ns, value.Validator, value.Datatype, value.Value, value.Blob)
	if err != nil {
		return nil, nil, nil, err
	}

	// Return a ref to the newly saved data
	return data, blob, &fftypes.DataRef{
		ID:   data.ID,
		Hash: data.Hash,
	}, nil
}

func (dm *dataManager) UploadJSON(ctx context.Context, ns string, inData *fftypes.DataRefOrValue) (*fftypes.Data, error) {
	data, _, err := dm.validateAndStore(ctx, ns, inData.Validator, inData.Datatype, inData.Value, inData.Blob)
	return data, err
}

func (dm *dataManager) ResolveInputDataPrivate(ctx context.Context, ns string, inData fftypes.InputData) (refs fftypes.DataRefs, err error) {
	refs, _, err = dm.resolveInputData(ctx, ns, inData, false)
	return refs, err
}

// ResolveInputDataBroadcast ensures the data object are stored, and returns a list of any data that does not currently
// have a public storage reference, and hence must be published to publicstorage before a broadcast message can be sent.
// We deliberately do NOT perform those publishes inside of this action, as we expect to be in a RunAsGroup (trnasaction)
// at this point, and hence expensive things like a multi-megabyte upload should be decoupled by our caller.
func (dm *dataManager) ResolveInputDataBroadcast(ctx context.Context, ns string, inData fftypes.InputData) (refs fftypes.DataRefs, dataToPublish []*fftypes.DataAndBlob, err error) {
	return dm.resolveInputData(ctx, ns, inData, true)
}

func (dm *dataManager) resolveInputData(ctx context.Context, ns string, inData fftypes.InputData, broadcast bool) (refs fftypes.DataRefs, dataToPublish []*fftypes.DataAndBlob, err error) {

	refs = make(fftypes.DataRefs, len(inData))
	if broadcast {
		dataToPublish = make([]*fftypes.DataAndBlob, 0, len(inData))
	}
	for i, dataOrValue := range inData {
		var data *fftypes.Data
		var blob *fftypes.Blob
		switch {
		case dataOrValue.ID != nil:
			// If an ID is supplied, then it must be a reference to existing data
			data, err = dm.resolveRef(ctx, ns, &dataOrValue.DataRef, false /* do not need the value */)
			if err != nil {
				return nil, nil, err
			}
			if data == nil {
				return nil, nil, i18n.NewError(ctx, i18n.MsgDataReferenceUnresolvable, i)
			}
			refs[i] = &fftypes.DataRef{
				ID:   data.ID,
				Hash: data.Hash,
			}
			if blob, err = dm.resolveBlob(ctx, data.Blob); err != nil {
				return nil, nil, err
			}
		case dataOrValue.Value != nil || dataOrValue.Blob != nil:
			// We've got a Value, so we can validate + store it
			if data, blob, refs[i], err = dm.validateAndStoreInlined(ctx, ns, dataOrValue); err != nil {
				return nil, nil, err
			}
		default:
			// We have nothing - this must be a mistake
			return nil, nil, i18n.NewError(ctx, i18n.MsgDataMissing, i)
		}

		// If the data is being resolved for public broadcast, and there is a blob attachment, that blob
		// needs to be published by our calller
		if broadcast && blob != nil && data.Blob.Public == "" {
			dataToPublish = append(dataToPublish, &fftypes.DataAndBlob{
				Data: data,
				Blob: blob,
			})
		}
	}
	return refs, dataToPublish, nil
}
