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

package broadcast

import (
	"context"
	"encoding/json"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (bm *broadcastManager) BroadcastMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error) {
	if err := bm.prepareMessage(ctx, ns, in); err != nil {
		return nil, err
	}
	return bm.resolveAndSend(ctx, in, nil, waitConfirm)
}

func (bm *broadcastManager) prepareMessage(ctx context.Context, ns string, msg *fftypes.MessageInOut) error {
	msg.Header.ID = fftypes.NewUUID()
	msg.Header.Namespace = ns
	if msg.Header.Type == "" {
		msg.Header.Type = fftypes.MessageTypeBroadcast
	}
	if msg.Header.TxType == "" {
		msg.Header.TxType = fftypes.TransactionTypeBatchPin
	}

	if !bm.isRootOrgBroadcast(ctx, &msg.Message) {
		// Resolve the sending identity
		if err := bm.identity.ResolveInputIdentity(ctx, &msg.Header.Identity); err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
		}
	}

	return nil
}

func (bm *broadcastManager) resolveAndSend(ctx context.Context, unresolved *fftypes.MessageInOut, resolved *fftypes.Message, waitConfirm bool) (out *fftypes.Message, err error) {
	if unresolved != nil {
		resolved = &unresolved.Message
	}

	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin)
	sent := false
	var dataToPublish []*fftypes.DataAndBlob
	err = bm.database.RunAsGroup(ctx, func(ctx context.Context) (err error) {
		if unresolved != nil {
			dataToPublish, err = bm.resolveMessage(ctx, unresolved)
			if err != nil {
				return err
			}
		}

		// For the simple case where we have no data to publish and aren't waiting for blockchain confirmation,
		// insert the local message immediately within the same DB transaction.
		// Otherwise, break out of the DB transaction (since those operations could take multiple seconds).
		if len(dataToPublish) == 0 && !waitConfirm {
			out, err = bm.broadcastMessageAsync(ctx, resolved)
			sent = true
		}
		return err
	})

	if err != nil || sent {
		return out, err
	}

	// Perform deferred processing
	if len(dataToPublish) > 0 {
		if err := bm.publishBlobs(ctx, dataToPublish); err != nil {
			return nil, err
		}
	}
	return bm.broadcastMessageCommon(ctx, resolved, waitConfirm)
}

func (bm *broadcastManager) resolveMessage(ctx context.Context, in *fftypes.MessageInOut) (dataToPublish []*fftypes.DataAndBlob, err error) {
	// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
	in.Data, dataToPublish, err = bm.data.ResolveInlineDataBroadcast(ctx, in.Header.Namespace, in.InlineData)
	return dataToPublish, err
}

func (bm *broadcastManager) isRootOrgBroadcast(ctx context.Context, message *fftypes.Message) bool {
	// Look into message to see if it contains a data item that is a root organization definition
	if message.Header.Type == fftypes.MessageTypeDefinition {
		messageData, ok, err := bm.data.GetMessageData(ctx, message, true)
		if ok && err == nil {
			if len(messageData) > 0 {
				dataItem := messageData[0]
				if dataItem.Validator == fftypes.MessageTypeDefinition {
					var org *fftypes.Organization
					err := json.Unmarshal(dataItem.Value, &org)
					if err != nil {
						return false
					}
					if org != nil && org.Name != "" && org.ID != nil && org.Parent == "" {
						return true
					}
				}
			}
		}
	}
	return false
}

func (bm *broadcastManager) publishBlobs(ctx context.Context, dataToPublish []*fftypes.DataAndBlob) error {

	for _, d := range dataToPublish {

		// Stream from the local data exchange ...
		reader, err := bm.exchange.DownloadBLOB(ctx, d.Blob.PayloadRef)
		if err != nil {
			return i18n.WrapError(ctx, err, i18n.MsgDownloadBlobFailed, d.Blob.PayloadRef)
		}
		defer reader.Close()

		// ... to the public storage
		publicRef, err := bm.publicstorage.PublishData(ctx, reader)
		if err != nil {
			return err
		}
		log.L(ctx).Infof("Published blob with hash '%s' for data '%s' to public storage: '%s'", d.Data.Blob, d.Data.ID, publicRef)

		// Update the data in the database, with the public reference.
		// We do this independently for each piece of data
		update := database.DataQueryFactory.NewUpdate(ctx).Set("blob.public", publicRef)
		err = bm.database.UpdateData(ctx, d.Data.ID, update)
		if err != nil {
			return err
		}
	}

	return nil
}
