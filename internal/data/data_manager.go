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

package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/dataexchange"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/sharedstorage"
	"github.com/karlseguin/ccache"
)

type Manager interface {
	CheckDatatype(ctx context.Context, ns string, datatype *fftypes.Datatype) error
	ValidateAll(ctx context.Context, data fftypes.DataArray) (valid bool, err error)
	GetMessageWithDataCached(ctx context.Context, msgID *fftypes.UUID, options ...CacheReadOption) (msg *fftypes.Message, data fftypes.DataArray, foundAllData bool, err error)
	GetMessageDataCached(ctx context.Context, msg *fftypes.Message, options ...CacheReadOption) (data fftypes.DataArray, foundAll bool, err error)
	UpdateMessageCache(msg *fftypes.Message, data fftypes.DataArray)
	ResolveInlineDataPrivate(ctx context.Context, ns string, inData fftypes.InlineData) (fftypes.DataArray, error)
	ResolveInlineDataBroadcast(ctx context.Context, ns string, inData fftypes.InlineData) (fftypes.DataArray, []*fftypes.DataAndBlob, error)
	VerifyNamespaceExists(ctx context.Context, ns string) error

	UploadJSON(ctx context.Context, ns string, inData *fftypes.DataRefOrValue) (*fftypes.Data, error)
	UploadBLOB(ctx context.Context, ns string, inData *fftypes.DataRefOrValue, blob *fftypes.Multipart, autoMeta bool) (*fftypes.Data, error)
	CopyBlobPStoDX(ctx context.Context, data *fftypes.Data) (blob *fftypes.Blob, err error)
	DownloadBLOB(ctx context.Context, ns, dataID string) (*fftypes.Blob, io.ReadCloser, error)
	HydrateBatch(ctx context.Context, persistedBatch *fftypes.BatchPersisted) (*fftypes.Batch, error)
}

type dataManager struct {
	blobStore
	database          database.Plugin
	exchange          dataexchange.Plugin
	validatorCache    *ccache.Cache
	validatorCacheTTL time.Duration
	messageCache      *ccache.Cache
	messageCacheTTL   time.Duration
}

type messageCacheEntry struct {
	msg  *fftypes.Message
	data []*fftypes.Data
	size int64
}

func (mce *messageCacheEntry) Size() int64 {
	return mce.size
}

type CacheReadOption int

const (
	CRORequirePublicBlobRefs CacheReadOption = iota
)

func NewDataManager(ctx context.Context, di database.Plugin, pi sharedstorage.Plugin, dx dataexchange.Plugin) (Manager, error) {
	if di == nil || pi == nil || dx == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	dm := &dataManager{
		database:          di,
		exchange:          dx,
		validatorCacheTTL: config.GetDuration(config.ValidatorCacheTTL),
	}
	dm.blobStore = blobStore{
		dm:            dm,
		database:      di,
		sharedstorage: pi,
		exchange:      dx,
	}
	dm.validatorCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(config.ValidatorCacheSize)),
	)
	dm.messageCache = ccache.New(
		// We use a LRU cache with a size-aware max
		ccache.Configure().
			MaxSize(config.GetByteSize(config.MessageCacheTTL)),
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

// GetMessageWithData performs a cached lookup of a message with all of the associated data.
// - Use this in performance sensitive code, but note mutable fields like the status of the
//   message CANNOT be relied upon (due to the caching).
func (dm *dataManager) GetMessageWithDataCached(ctx context.Context, msgID *fftypes.UUID, options ...CacheReadOption) (msg *fftypes.Message, data fftypes.DataArray, foundAllData bool, err error) {
	if mce := dm.queryMessageCache(ctx, msgID, options...); mce != nil {
		return mce.msg, mce.data, true, nil
	}
	msg, err = dm.database.GetMessageByID(ctx, msgID)
	if err != nil || msg == nil {
		return nil, nil, false, err
	}
	data, foundAllData, err = dm.dataLookupAndCache(ctx, msg)
	return msg, data, foundAllData, err
}

// GetMessageData looks for all the data attached to the message, including caching.
// It only returns persistence errors.
// For all cases where the data is not found (or the hashes mismatch)
func (dm *dataManager) GetMessageDataCached(ctx context.Context, msg *fftypes.Message, options ...CacheReadOption) (data fftypes.DataArray, foundAll bool, err error) {
	if mce := dm.queryMessageCache(ctx, msg.Header.ID, options...); mce != nil {
		return mce.data, true, nil
	}
	return dm.dataLookupAndCache(ctx, msg)
}

// cachedMessageAndDataLookup is the common function that can lookup and cache a message with its data
func (dm *dataManager) dataLookupAndCache(ctx context.Context, msg *fftypes.Message) (data fftypes.DataArray, foundAllData bool, err error) {
	data, foundAllData, err = dm.getMessageData(ctx, msg)
	if err != nil {
		return nil, false, err
	}
	if !foundAllData {
		return data, false, err
	}
	dm.UpdateMessageCache(msg, data)
	return data, true, nil
}

func (dm *dataManager) queryMessageCache(ctx context.Context, id *fftypes.UUID, options ...CacheReadOption) *messageCacheEntry {
	cached := dm.messageCache.Get(id.String())
	if cached == nil {
		return nil
	}
	mce := cached.Value().(*messageCacheEntry)
	for _, opt := range options {
		if opt == CRORequirePublicBlobRefs {
			for idx, d := range mce.data {
				if d.Blob != nil && d.Blob.Public == "" {
					log.L(ctx).Debugf("Cache entry for data %d (%s) in message %s is missing public blob ref", idx, d.ID, mce.msg.Header.ID)
					return nil
				}
			}
		}
	}
	log.L(ctx).Debugf("Returning msg %s from cache", id)
	return mce
}

// UpdateMessageCache pushes an entry to the message cache. It is exposed out of the package, so that
// code which generates (or augments) message/data can populate the cache.
func (dm *dataManager) UpdateMessageCache(msg *fftypes.Message, data fftypes.DataArray) {
	cacheEntry := &messageCacheEntry{
		msg:  msg,
		data: data,
		size: msg.EstimateSize(true),
	}
	dm.messageCache.Set(msg.Header.ID.String(), cacheEntry, dm.messageCacheTTL)
}

func (dm *dataManager) getMessageData(ctx context.Context, msg *fftypes.Message) (data fftypes.DataArray, foundAll bool, err error) {
	// Load all the data - must all be present for us to send
	data = make(fftypes.DataArray, 0, len(msg.Data))
	foundAll = true
	for i, dataRef := range msg.Data {
		d, err := dm.resolveRef(ctx, msg.Header.Namespace, dataRef)
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

func (dm *dataManager) ValidateAll(ctx context.Context, data fftypes.DataArray) (valid bool, err error) {
	for _, d := range data {
		if d.Datatype != nil && d.Validator != fftypes.ValidatorTypeNone {
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

func (dm *dataManager) resolveRef(ctx context.Context, ns string, dataRef *fftypes.DataRef) (*fftypes.Data, error) {
	if dataRef == nil || dataRef.ID == nil {
		log.L(ctx).Warnf("data is nil")
		return nil, nil
	}
	d, err := dm.database.GetDataByID(ctx, dataRef.ID, true)
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

func (dm *dataManager) checkValidation(ctx context.Context, ns string, validator fftypes.ValidatorType, datatype *fftypes.DatatypeRef, value *fftypes.JSONAny) error {
	if validator == "" {
		validator = fftypes.ValidatorTypeJSON
	}
	if err := fftypes.CheckValidatorType(ctx, validator); err != nil {
		return err
	}
	// If a datatype is specified, we need to verify the payload conforms
	if datatype != nil && validator != fftypes.ValidatorTypeNone {
		if datatype.Name == "" || datatype.Version == "" {
			return i18n.NewError(ctx, i18n.MsgDatatypeNotFound, datatype)
		}
		if validator != fftypes.ValidatorTypeNone {
			v, err := dm.getValidatorForDatatype(ctx, ns, validator, datatype)
			if err != nil {
				return err
			}
			if v == nil {
				return i18n.NewError(ctx, i18n.MsgDatatypeNotFound, datatype)
			}
			err = v.ValidateValue(ctx, value, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (dm *dataManager) validateAndStore(ctx context.Context, ns string, validator fftypes.ValidatorType, datatype *fftypes.DatatypeRef, value *fftypes.JSONAny, blobRef *fftypes.BlobRef) (data *fftypes.Data, blob *fftypes.Blob, err error) {

	if err := dm.checkValidation(ctx, ns, validator, datatype, value); err != nil {
		return nil, nil, err
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
	err = data.Seal(ctx, blob)
	if err == nil {
		err = dm.database.UpsertData(ctx, data, database.UpsertOptimizationNew)
	}
	if err != nil {
		return nil, nil, err
	}
	return data, blob, nil
}

func (dm *dataManager) validateAndStoreInlined(ctx context.Context, ns string, value *fftypes.DataRefOrValue) (*fftypes.Data, *fftypes.Blob, error) {
	data, blob, err := dm.validateAndStore(ctx, ns, value.Validator, value.Datatype, value.Value, value.Blob)
	if err != nil {
		return nil, nil, err
	}

	// Return a ref to the newly saved data
	return data, blob, nil
}

func (dm *dataManager) UploadJSON(ctx context.Context, ns string, inData *fftypes.DataRefOrValue) (*fftypes.Data, error) {
	data, _, err := dm.validateAndStore(ctx, ns, inData.Validator, inData.Datatype, inData.Value, inData.Blob)
	return data, err
}

func (dm *dataManager) ResolveInlineDataPrivate(ctx context.Context, ns string, inData fftypes.InlineData) (fftypes.DataArray, error) {
	data, _, err := dm.resolveInlineData(ctx, ns, inData, false)
	return data, err
}

// ResolveInlineDataBroadcast ensures the data object are stored, and returns a list of any data that does not currently
// have a shared storage reference, and hence must be published to sharedstorage before a broadcast message can be sent.
// We deliberately do NOT perform those publishes inside of this action, as we expect to be in a RunAsGroup (trnasaction)
// at this point, and hence expensive things like a multi-megabyte upload should be decoupled by our caller.
func (dm *dataManager) ResolveInlineDataBroadcast(ctx context.Context, ns string, inData fftypes.InlineData) (data fftypes.DataArray, dataToPublish []*fftypes.DataAndBlob, err error) {
	return dm.resolveInlineData(ctx, ns, inData, true)
}

func (dm *dataManager) resolveInlineData(ctx context.Context, ns string, inData fftypes.InlineData, broadcast bool) (data fftypes.DataArray, dataToPublish []*fftypes.DataAndBlob, err error) {

	data = make(fftypes.DataArray, len(inData))
	if broadcast {
		dataToPublish = make([]*fftypes.DataAndBlob, 0, len(inData))
	}
	for i, dataOrValue := range inData {
		var d *fftypes.Data
		var blob *fftypes.Blob
		switch {
		case dataOrValue.ID != nil:
			// If an ID is supplied, then it must be a reference to existing data
			d, err = dm.resolveRef(ctx, ns, &dataOrValue.DataRef)
			if err != nil {
				return nil, nil, err
			}
			if d == nil {
				return nil, nil, i18n.NewError(ctx, i18n.MsgDataReferenceUnresolvable, i)
			}
			data[i] = d
			if blob, err = dm.resolveBlob(ctx, d.Blob); err != nil {
				return nil, nil, err
			}
		case dataOrValue.Value != nil || dataOrValue.Blob != nil:
			// We've got a Value, so we can validate + store it
			if data[i], blob, err = dm.validateAndStoreInlined(ctx, ns, dataOrValue); err != nil {
				return nil, nil, err
			}
		default:
			// We have nothing - this must be a mistake
			return nil, nil, i18n.NewError(ctx, i18n.MsgDataMissing, i)
		}

		// If the data is being resolved for public broadcast, and there is a blob attachment, that blob
		// needs to be published by our calller
		if broadcast && blob != nil && d.Blob.Public == "" {
			dataToPublish = append(dataToPublish, &fftypes.DataAndBlob{
				Data: d,
				Blob: blob,
			})
		}
	}
	return data, dataToPublish, nil
}

// HydrateBatch fetches the full messages for a persited batch, ready for transmission
func (dm *dataManager) HydrateBatch(ctx context.Context, persistedBatch *fftypes.BatchPersisted) (*fftypes.Batch, error) {

	var manifest fftypes.BatchManifest
	err := json.Unmarshal([]byte(persistedBatch.Manifest), &manifest)
	if err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgJSONObjectParseFailed, fmt.Sprintf("batch %s manifest", persistedBatch.ID))
	}

	batch := &fftypes.Batch{
		BatchHeader: persistedBatch.BatchHeader,
		PayloadRef:  persistedBatch.PayloadRef,
		Payload: fftypes.BatchPayload{
			TX:       persistedBatch.TX,
			Messages: make([]*fftypes.Message, len(manifest.Messages)),
			Data:     make(fftypes.DataArray, len(manifest.Data)),
		},
	}

	for i, mr := range manifest.Messages {
		m, err := dm.database.GetMessageByID(ctx, mr.ID)
		if err != nil || m == nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgFailedToRetrieve, "message", mr.ID)
		}
		// BatchMessage removes any fields that could change after the batch was first assembled on the sender
		batch.Payload.Messages[i] = m.BatchMessage()
	}
	for i, dr := range manifest.Data {
		d, err := dm.database.GetDataByID(ctx, dr.ID, true)
		if err != nil || d == nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgFailedToRetrieve, "data", dr.ID)
		}
		// BatchData removes any fields that could change after the batch was first assembled on the sender
		batch.Payload.Data[i] = d.BatchData(persistedBatch.Type)
	}

	return batch, nil
}
